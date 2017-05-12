package orm

import "github.com/bysir-zl/bygo/log"

var byLoger log.Logger
func warn(a ...interface{}) {
	if Debug {
		byLoger.Warn("ORM", a...)
	}
}

func info(a ...interface{}) {
	if Debug {
		byLoger.Info("ORM", a...)
	}
}

func init() {
	byLoger.SetCallDepth(4)
}