package storage

import (
	"github.com/go-mysql-org/go-mysql/mysql"

	"cardappcanal/global"
)

type PositionStorage interface {
	Initialize() error
	Save(pos mysql.Position) error
	Get() (mysql.Position, error)
}

func NewPositionStorage() PositionStorage {
	if global.Cfg().IsCluster() {
		if global.Cfg().IsZk() {
			return &zkPositionStorage{}
		}
		if global.Cfg().IsEtcd() {
			return &etcdPositionStorage{}
		}
	}

	return &boltPositionStorage{}
}
