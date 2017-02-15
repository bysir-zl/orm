package tests

import (
	"github.com/bysir-zl/bygo/util"
	"github.com/bysir-zl/orm"
	"log"
	"strings"
	"testing"
	"time"
)

func TestTagmap(t *testing.T) {
	test := &User{}
	tag := "orm"

	tagMap := util.GetTagMapperFromPool(test)
	table := tagMap.GetFieldMapByTagName("table")["orm"]
	connect := tagMap.GetFieldMapByTagName("connect")["orm"]

	// field=>dbData
	fieldMap := tagMap.GetFieldMapByTagName(tag)

	field2Tag := map[string]string{}
	autoPk := ""
	for field, db := range fieldMap {
		column := DecodeColumn(db)
		field2Tag[field] = column.Name
		if column.Pk == "auto" {
			autoPk = field
		}
	}

	m := orm.ModelInfo{
		FieldMap:   field2Tag,
		AutoPk:     autoPk,
		Table:      table,
		ConnectName:connect,
	}

	log.Printf("%+v", m)
}

type Column struct {
	Name string
	Pk   string // "":不是pk, auto:自增pk
	Tran string // 自动转换规则 json:string转换为field
	Auto struct {
		Where string // 当什么时候自动更新字段
		Typ   string // 目前只支持时间的自动更新
	}
}

func DecodeColumn(dbData string) *Column {
	c := &Column{}
	ds := strings.Split(dbData, ";")
	l := len(ds)
	if l > 0 {
		c.Name = ds[0]
	}
	if l > 1 {
		for i := 1; i < l; i++ {
			kv := ds[i]
			key := ""
			values := []string{""}
			if !strings.Contains(kv, "(") {
				key = kv
			} else {
				kAndV := strings.Split(kv, "(")
				key = kAndV[0]
				v := strings.Split(kAndV[1], ")")[0]
				values = strings.Split(v, ",")
			}
			switch key {
			case "pk":
				c.Pk = values[0]
			case "tran":
				c.Tran = values[0]
			case "auto":
				c.Auto = struct {
					Where string
					Typ   string
				}{Where: values[0], Typ: values[1]}
			}

		}
	}

	return c
}

type Role struct {
	orm string `table:"role" connect:"default" json:"-"`

	Id   int    `orm:"col(id);pk(auto);" json:"id"`
	Name string `orm:"col(name)" json:"name"`
}

type User struct {
	orm string `table:"user" connect:"default" json:"-"`

	Id         int    `orm:"col(id);pk(auto);" json:"id"`
	Name       string `orm:"col(name)" json:"name"`
	Sex        bool `orm:"col(sex)" json:"sex"`
	Role_ids   []int `orm:"col(role_ids);tran(json)" json:"role_ids"`
	RoleId     int `orm:"col(role_id)"  json:"stime"`
	Created_at string `orm:"col(created_at);auto(insert,time)"  json:"stime"`
	Updated_at string `orm:"col(updated_at);auto(insert|update,time);tran(time)" json:"itime"`

	RoleRaw *Role `orm:"col(role_raw);tran(json)"`
	Roles   []Role `orm:"col(role_raws);tran(json)"`

	Role   *Role `orm:"link(RoleId,id)"`
	Roles2 []Role `orm:"link(Role_ids,id)"`
}

func TestInsert(t *testing.T) {
	test := User{
		Name:"bysir",
		RoleId:1,
		Role_ids:[]int{1,2,3},
		Sex:true,
		RoleRaw:&Role{
			Name:"inJson",
			Id:  1,
		},
	}
	test.Roles = []Role{
		{
			Id:  10086,
			Name:"sb",
		},
	}

	err := orm.Model(&test).Insert(&test)
	if err != nil {
		t.Error(err)
	}
}

func TestSelect(t *testing.T) {
	ts := []User{}
	_, err := orm.Model(&ts).
		Link("Role", "", []string{"name"}).
		Link("Roles2", "", nil).
		Select(&ts)
	if err != nil {
		t.Error(err)
	}
	for _, tt := range ts {
		log.Printf("%+v", tt.Role)
	}
	//log.Printf("        %+v", ts)-
	log.Printf("role    %+v", ts[0].Role)
	log.Printf("roleRaw %+v", ts[2].RoleRaw)
	log.Printf("roles   %+v", ts[2].Roles)
	log.Printf("roles2  %+v", ts[2].Roles2)
	time.Sleep(10000000)
}

func init() {
	orm.Debug = true

	orm.RegisterDb("default", "mysql", "root:root@tcp(localhost:3306)/test")
	orm.RegisterModel(new(User))
	orm.RegisterModel(new(Role))
}
