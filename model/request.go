package model

import "sync"

var RowRequestPool = sync.Pool{
	New: func() interface{} {
		return new(RowRequest)
	},
}

type RowRequest struct {
	RuleKey   string
	Action    string
	Timestamp uint32
	Old       []interface{}
	Row       []interface{}
}

type PosRequest struct {
	Name  string
	Pos   uint32
	Force bool
}

type TabelChangeRequest struct {
	Query []byte
}
