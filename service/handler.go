package service

import (
	"cardappcanal/metrics"
	"fmt"
	"log"
	"time"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/juju/errors"

	"cardappcanal/global"
	"cardappcanal/model"
	"cardappcanal/util/logs"
)

type handler struct {
	queue      chan interface{}
	tableQueue chan interface{}
	stop       chan struct{}
}

func (s *handler) OnGTID(header *replication.EventHeader, gtidEvent mysql.BinlogGTIDEvent) error {
	//TODO implement me
	return nil
}

func (s *handler) OnPosSynced(header *replication.EventHeader, pos mysql.Position, set mysql.GTIDSet, force bool) error {
	//TODO implement me
	return nil
}

func (s *handler) OnRowsQueryEvent(e *replication.RowsQueryEvent) error {
	//TODO implement me
	return nil
}

func newHandler() *handler {
	return &handler{
		queue:      make(chan interface{}, 4096),
		tableQueue: make(chan interface{}, 80),
		stop:       make(chan struct{}, 1),
	}
}

func (s *handler) OnRotate(header *replication.EventHeader, rotateEvent *replication.RotateEvent) error {
	s.queue <- model.PosRequest{
		Name:  string(rotateEvent.NextLogName),
		Pos:   uint32(rotateEvent.Position),
		Force: true,
	}
	return nil
}

func (s *handler) OnTableChanged(header *replication.EventHeader, schema string, table string) error {
	fmt.Print("-----------------------")
	fmt.Print("-----------------------")
	fmt.Print("-----------------------")
	err := _transferService.updateRule(schema, table)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (s *handler) OnDDL(header *replication.EventHeader, nextPos mysql.Position, q *replication.QueryEvent) error {
	fmt.Printf("-------------------%s", string(q.Query))
	s.queue <- model.TabelChangeRequest{
		Query: q.Query,
	}
	return nil
}

func (s *handler) OnXID(header *replication.EventHeader, nextPos mysql.Position) error {
	s.queue <- model.PosRequest{
		Name:  nextPos.Name,
		Pos:   nextPos.Pos,
		Force: false,
	}
	return nil
}

func (s *handler) OnRow(e *canal.RowsEvent) error {
	ruleKey := global.RuleKey(e.Table.Schema, e.Table.Name)
	if !global.RuleInsExist(ruleKey) {
		return nil
	}

	var requests []*model.RowRequest
	if e.Action != canal.UpdateAction {
		// 定长分配
		requests = make([]*model.RowRequest, 0, len(e.Rows))
	}

	if e.Action == canal.UpdateAction {
		for i := 0; i < len(e.Rows); i++ {
			if (i+1)%2 == 0 {
				v := new(model.RowRequest)
				v.RuleKey = ruleKey
				v.Action = e.Action
				v.Timestamp = e.Header.Timestamp
				if global.Cfg().IsReserveRawData() {
					v.Old = e.Rows[i-1]
				}
				v.Row = e.Rows[i]
				requests = append(requests, v)
			}
		}
	} else {
		for _, row := range e.Rows {
			v := new(model.RowRequest)
			v.RuleKey = ruleKey
			v.Action = e.Action
			v.Timestamp = e.Header.Timestamp
			v.Row = row
			requests = append(requests, v)
		}
	}
	s.queue <- requests

	return nil
}

func (s *handler) String() string {
	return "TransferHandler"
}

func (s *handler) startListener() {
	go func() {
		interval := time.Duration(global.Cfg().FlushBulkInterval)
		bulkSize := global.Cfg().BulkSize
		ticker := time.NewTicker(time.Millisecond * interval)
		defer ticker.Stop()
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			log.Fatal(err)
		}
		// 获取当前时间的上海时区时间
		lastSavedTime := time.Now().In(loc)
		requests := make([]*model.RowRequest, 0, bulkSize)
		var current mysql.Position
		from, _ := _transferService.positionDao.Get()
		for {
			needFlush := false
			needSavePos := false
			var query []byte
			select {
			case v := <-s.queue:
				switch v := v.(type) {
				case model.PosRequest:
					fmt.Print("PosRequest\n")

					now := time.Now()
					if v.Force || now.Sub(lastSavedTime) > 3*time.Second {
						lastSavedTime = now
						needFlush = true
						needSavePos = true
						current = mysql.Position{
							Name: v.Name,
							Pos:  v.Pos,
						}
					}
				case []*model.RowRequest:
					fmt.Print("RowRequest2\n")

					requests = append(requests, v...)
					needFlush = int64(len(requests)) >= global.Cfg().BulkSize
				case *model.TabelChangeRequest:
					fmt.Print("TabelChangeRequest3\n")
					query = v.Query
				}
			case <-ticker.C:
				needFlush = true
			case <-s.stop:
				return
			}
			if len(query) > 0 {
				_transferService.endpoint.ChangeTableName(query)
			}

			if needFlush && len(requests) > 0 && _transferService.endpointEnable.Load() {
				err := _transferService.endpoint.Consume(from, requests)
				if err != nil {
					_transferService.endpointEnable.Store(false)
					metrics.SetDestState(metrics.DestStateFail)
					logs.Error(err.Error())
					go _transferService.stopDump()
				}
				requests = requests[0:0]
			}
			if needSavePos && _transferService.endpointEnable.Load() {
				logs.Infof("save position %s %d", current.Name, current.Pos)
				if err := _transferService.positionDao.Save(current); err != nil {
					logs.Errorf("save sync position %s err %v, close sync", current, err)
					_transferService.Close()
					return
				}
				from = current
			}
		}
	}()
}

func (s *handler) stopListener() {
	log.Println("transfer stop")
	s.stop <- struct{}{}
}
