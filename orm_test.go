package orm

import (
	"log"
	"testing"
)

type TestModel struct {
	orm          string `table:"c_platform" connect:"default" json:"-"`
	Id           int    `db:"id" pk:"auto" json:"id"`
	Name         string `db:"platform_name" json:"name"`
	PlatformCode string `db:"platform_code" json:"platform_code"`
	Stime        string `db:"ctime" auto:"timestr,insert" json:"stime"`
	Itime        string `db:"itime" auto:"timeint,update" json:"itime"`
}

func TestOrm(t *testing.T) {
	m := []TestModel{}
	config := map[string]Connect{
		"default":{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Name:     "anyminisdk",
			User:     "root",
			Password: "root",
		},
	}
	// u can set it to globe
	orm := New(config)

	m2 := orm.Model(&m)
	has, err := m2.Where("`Id` > ?", "1").Get()
	log.Print("err 	: ", err)
	log.Print("sqls 	: ", m2.Sqls())
	log.Print("model 	: ", has, m)

	m2.Reset()
	has, err = m2.Where("`Id` < ?", "1").Get()
	if err != nil {
		log.Print("has 	: ", has)
	}
	log.Print("------")
	log.Print("err 	: ", err)
	log.Print("sqls 	: ", m2.Sqls())
	log.Print("model 	: ", m)

	m3 := orm.Table("c_platform")
	has, dat, err := m3.Where("`Id`>?", "1").QueryToMap()
	log.Print("------")
	log.Print("err 	: ", err)
	log.Print("sqls 	: ", m3.Sqls())
	log.Print("map 	: ", has, dat)

	m4 := TestModel{}
	m4.Name = "zl"
	m4.PlatformCode = "kf"
	m5 := orm.Model(&m4)
	err = m5.Insert()
	log.Print("------")
	log.Print("err 	: ", err)
	log.Print("sqls 	: ", m5.Sqls())
	log.Print("model 	: ", m4)

}
