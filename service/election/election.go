package election

import (
	"cardappcanal/global"
)

type Service interface {
	Elect() error
	IsLeader() bool
	Leader() string
	Nodes() []string
}

func NewElection(_informCh chan bool) Service {
	if global.Cfg().IsZk() {
		return newZkElection(_informCh)
	} else {
		return newEtcdElection(_informCh)
	}
}
