package storage

import (
	"encoding/json"

	"github.com/go-mysql-org/go-mysql/mysql"

	"cardappcanal/global"
	"cardappcanal/util/zookeepers"
)

type zkPositionStorage struct {
}

func (s *zkPositionStorage) Initialize() error {
	pos, err := json.Marshal(mysql.Position{})
	if err != nil {
		return err
	}

	err = zookeepers.CreateDirWithDataIfNecessary(global.Cfg().ZkPositionDir(), pos, _zkConn)
	if err != nil {
		return err
	}

	err = zookeepers.CreateDirIfNecessary(global.Cfg().ZkNodesDir(), _zkConn)
	return err
}

func (s *zkPositionStorage) Save(pos mysql.Position) error {
	_, stat, err := _zkConn.Get(global.Cfg().ZkPositionDir())
	if err != nil {
		return err
	}

	data, err := json.Marshal(pos)
	if err != nil {
		return err
	}

	_, err = _zkConn.Set(global.Cfg().ZkPositionDir(), data, stat.Version)

	return err
}

func (s *zkPositionStorage) Get() (mysql.Position, error) {
	var entity mysql.Position

	data, _, err := _zkConn.Get(global.Cfg().ZkPositionDir())
	if err != nil {
		return entity, err
	}

	err = json.Unmarshal(data, &entity)

	return entity, err
}
