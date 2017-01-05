package orm

import "github.com/bysir-zl/bygo/log"

func warn(a ...interface{}) {
	if Debug {
		log.Warn("ORM", a...)
	}
}

func info(a ...interface{}) {
	if Debug {
		log.Info("ORM", a...)
	}
}

func init() {
	log.SetCallDepth(4)
}