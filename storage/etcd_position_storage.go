package storage

import (
	"encoding/json"

	"github.com/go-mysql-org/go-mysql/mysql"

	"cardappcanal/global"
	"cardappcanal/util/etcds"
)

type etcdPositionStorage struct {
}

func (s *etcdPositionStorage) Initialize() error {
	data, err := json.Marshal(mysql.Position{})
	if err != nil {
		return err
	}

	err = etcds.CreateIfNecessary(global.Cfg().ZkPositionDir(), string(data), _etcdOps)
	if err != nil {
		return err
	}

	return nil
}

func (s *etcdPositionStorage) Save(pos mysql.Position) error {
	data, err := json.Marshal(pos)
	if err != nil {
		return err
	}

	return etcds.Save(global.Cfg().ZkPositionDir(), string(data), _etcdOps)
}

func (s *etcdPositionStorage) Get() (mysql.Position, error) {
	var entity mysql.Position

	data, _, err := etcds.Get(global.Cfg().ZkPositionDir(), _etcdOps)
	if err != nil {
		return entity, err
	}

	err = json.Unmarshal(data, &entity)

	return entity, err
}
