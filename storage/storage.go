package storage

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	"go.etcd.io/bbolt"
	"go.etcd.io/etcd/clientv3"
	etcdlog "go.etcd.io/etcd/pkg/logutil"

	"cardappcanal/global"
	"cardappcanal/util/byteutil"
	"cardappcanal/util/files"
	"cardappcanal/util/logagent"
	"cardappcanal/util/zookeepers"
)

const (
	_boltFilePath = "db"
	_boltFileName = "data.db"
	_boltFileMode = 0600
)

var (
	_positionBucket = []byte("Position")
	_fixPositionId  = byteutil.Uint64ToBytes(uint64(1))

	_bolt           *bbolt.DB
	_zkConn         *zk.Conn
	_zkStatusSignal <-chan zk.Event
	_zkAddresses    []string

	_etcdConn *clientv3.Client
	_etcdOps  clientv3.KV
)

func Initialize() error {
	if err := initBolt(); err != nil {
		return err
	}

	if global.Cfg().IsZk() {
		if err := initZk(); err != nil {
			return err
		}
	}

	if global.Cfg().IsEtcd() {
		if err := initEtcd(); err != nil {
			return err
		}
	}

	return nil
}

func initBolt() error {
	blotStorePath := filepath.Join(global.Cfg().DataDir, _boltFilePath)
	if err := files.MkdirIfNecessary(blotStorePath); err != nil {
		return errors.New(fmt.Sprintf("create boltdb store : %s", err.Error()))
	}

	boltFilePath := filepath.Join(blotStorePath, _boltFileName)
	bolt, err := bbolt.Open(boltFilePath, _boltFileMode, bbolt.DefaultOptions)
	if err != nil {
		return errors.New(fmt.Sprintf("open boltdb: %s", err.Error()))
	}

	err = bolt.Update(func(tx *bbolt.Tx) error {
		tx.CreateBucketIfNotExists(_positionBucket)
		return nil
	})

	_bolt = bolt

	return err
}

func initZk() error {
	option := zk.WithLogger(logagent.NewZkLoggerAgent())
	list := strings.Split(global.Cfg().Cluster.ZkAddrs, ",")
	conn, sig, err := zk.Connect(list, time.Second, option) //*10)

	if err != nil {
		return err
	}

	if global.Cfg().Cluster.ZkAuthentication != "" {
		err = conn.AddAuth("digest", []byte(global.Cfg().Cluster.ZkAuthentication))
		if err != nil {
			return err
		}
	}

	err = zookeepers.CreateDirIfNecessary(global.Cfg().ZkRootDir(), conn)
	if err != nil {
		return err
	}

	err = zookeepers.CreateDirIfNecessary(global.Cfg().ZkClusterDir(), conn)
	if err != nil {
		return err
	}

	_zkAddresses = list
	_zkConn = conn
	_zkStatusSignal = sig

	return nil
}

func initEtcd() error {
	etcdlog.DefaultZapLoggerConfig = logagent.EtcdZapLoggerConfig()
	clientv3.SetLogger(logagent.NewEtcdLoggerAgent())

	list := strings.Split(global.Cfg().Cluster.EtcdAddrs, ",")
	config := clientv3.Config{
		Endpoints:   list,
		Username:    global.Cfg().Cluster.EtcdUser,
		Password:    global.Cfg().Cluster.EtcdPassword,
		DialTimeout: 1 * time.Second,
	}

	client, err := clientv3.New(config)
	if err != nil {
		return err
	}
	_etcdConn = client
	_etcdOps = clientv3.NewKV(_etcdConn)

	return nil
}

func ZKConn() *zk.Conn {
	return _zkConn
}

func ZKStatusSignal() <-chan zk.Event {
	return _zkStatusSignal
}

func ZKAddresses() []string {
	return _zkAddresses
}

func EtcdConn() *clientv3.Client {
	return _etcdConn
}

func EtcdOps() clientv3.KV {
	return _etcdOps
}

func Close() {
	if _bolt != nil {
		_bolt.Close()
	}
	if _zkConn != nil {
		_zkConn.Close()
	}
	if _etcdConn != nil {
		_etcdConn.Close()
	}
}
