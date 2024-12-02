package model

import "sync"

var mqRespondPool = sync.Pool{
	New: func() interface{} {
		return new(MQRespond)
	},
}

type MQRespond struct {
	Topic     string      `json:"-"`
	Action    string      `json:"action"`
	Timestamp uint32      `json:"timestamp"`
	Raw       interface{} `json:"raw,omitempty"`
	Date      interface{} `json:"date"`
	ByteArray []byte      `json:"-"`
}

type MysqlRespond struct {
	RuleKey    string
	Collection string
	Action     string
	Id         interface{}
	Table      map[string]interface{}
}
