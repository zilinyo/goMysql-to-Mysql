package election

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/juju/errors"
	"go.etcd.io/etcd/clientv3/concurrency"
	"go.uber.org/atomic"

	"cardappcanal/global"
	"cardappcanal/storage"
	"cardappcanal/util/etcds"
	"cardappcanal/util/logs"
)

const _electionNodeTTL = 2 //秒

type etcdElection struct {
	once sync.Once

	informCh chan bool

	selected atomic.Bool
	ensured  atomic.Bool
	leader   atomic.String
}

func newEtcdElection(_informCh chan bool) *etcdElection {
	return &etcdElection{
		informCh: _informCh,
	}
}

func (s *etcdElection) Elect() error {
	s.doElect()
	s.ensureFollower()
	return nil
}

func (s *etcdElection) doElect() {
	go func() {

		for {
			session, err := concurrency.NewSession(storage.EtcdConn(), concurrency.WithTTL(_electionNodeTTL))
			if err != nil {
				logs.Error(err.Error())
				return
			}
			elc := concurrency.NewElection(session, global.Cfg().ZkElectionDir())
			ctx := context.Background()
			if err = elc.Campaign(ctx, global.CurrentNode()); err != nil {
				logs.Error(errors.ErrorStack(err))
				session.Close()
				s.beFollower("")
				continue
			}

			select {
			case <-session.Done():
				s.beFollower("")
				continue
			default:
				s.beLeader()
				err = etcds.UpdateOrCreate(global.Cfg().ZkElectedDir(), elc.Key(), storage.EtcdOps())
				if err != nil {
					logs.Error(errors.ErrorStack(err))
					return
				}
			}

			shouldBreak := false
			for !shouldBreak {
				select {
				case <-session.Done():
					logs.Warn("etcd session has done")
					shouldBreak = true
					s.beFollower("")
					break
				case <-ctx.Done():
					ctxTmp, _ := context.WithTimeout(context.Background(), time.Second*_electionNodeTTL)
					elc.Resign(ctxTmp)
					session.Close()
					s.beFollower("")
					return
				}
			}
		}
	}()
}

func (s *etcdElection) IsLeader() bool {
	return s.selected.Load()
}

func (s *etcdElection) Leader() string {
	return s.leader.Load()
}

func (s *etcdElection) ensureFollower() {
	go func() {
		for {
			if s.selected.Load() {
				break
			}

			k, _, err := etcds.Get(global.Cfg().ZkElectedDir(), storage.EtcdOps())
			if err != nil {
				logs.Error(errors.ErrorStack(err))
				continue
			}

			var l []byte
			l, _, err = etcds.Get(string(k), storage.EtcdOps())
			if err != nil {
				logs.Error(errors.ErrorStack(err))
				continue
			}

			s.ensured.Store(true)
			s.beFollower(string(l))
			break
		}
	}()
}

func (s *etcdElection) Nodes() []string {
	var nodes []string
	ls, err := etcds.List("/transfer/myTransfer/election", storage.EtcdOps())
	if err == nil {
		for _, v := range ls {
			nodes = append(nodes, string(v.Value))
		}
	}
	return nodes
}

func (s *etcdElection) beLeader() {
	s.selected.Store(true)
	s.leader.Store(global.CurrentNode())
	s.informCh <- s.selected.Load()
	log.Println("the current node is the master")
}

func (s *etcdElection) beFollower(leader string) {
	s.selected.Store(false)
	s.informCh <- s.selected.Load()
	s.leader.Store(leader)
	log.Println(fmt.Sprintf("The current node is the follower, master node is : %s", s.leader.Load()))
}
