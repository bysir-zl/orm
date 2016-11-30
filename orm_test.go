package orm

import (
	"testing"
	"log"
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
	err := m2.Where("`Id`>?", "1").Get()
	log.Print("err 	: ", err)
	log.Print("sqls 	: ", m2.Sqls())
	log.Print("model 	: ", m)

	m2.Reset()
	err = m2.Where("`Id`<?", "1").Get()
	// Get or First maybe return err is NotFoundError
	// please check this error
	if err != nil {
		_, ok := err.(NotFoundError)
		log.Print("notfound 	: ", ok)
	}
	log.Print("------")
	log.Print("err 	: ", err)
	log.Print("sqls 	: ", m2.Sqls())
	log.Print("model 	: ", m)

	m3 := orm.Table("c_platform")
	dat, err := m3.Where("`Id`>?", "1").QueryToMap()
	log.Print("------")
	log.Print("err 	: ", err)
	log.Print("sqls 	: ", m3.Sqls())
	log.Print("map 	: ", dat)

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
