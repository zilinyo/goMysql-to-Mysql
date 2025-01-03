package logagent

import "cardappcanal/util/logs"

type ZkLoggerAgent struct {
}

func NewZkLoggerAgent() *ZkLoggerAgent {
	return &ZkLoggerAgent{}
}

func (s *ZkLoggerAgent) Printf(template string, args ...interface{}) {
	logs.Infof(template, args...)
}
