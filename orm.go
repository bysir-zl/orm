package orm

import (
	"fmt"
	"reflect"
	"errors"
)

var (
	Debug = false
)

type Connect struct {
	Driver string `json:"driver"`
	// USER:PWD@tcp(HOST:PORT)/DBNAME
	Url string `json:"url"`
}

func (p *Connect) String() string {
	return fmt.Sprintf("%s~%s", p.Url, p.Driver)
}

func (p *Connect) SqlString() string {
	return p.Url
}

type Config map[string]Connect

var config = Config{}

func (p *Config) writeConnect(connect string) (conn *Connect, err error) {
	m := map[string]Connect(*p)
	if c, ok := m[connect + "-write"]; ok {
		conn = &c
		return
	}
	if c, ok := m[connect]; ok {
		conn = &c
		return
	}
	err = errors.New("can't found connect: " + connect)
	return
}
func (p *Config) readConnect(connect string) (conn *Connect, err error) {
	m := map[string]Connect(*p)
	if c, ok := m[connect + "-read"]; ok {
		conn = &c
		return
	}
	if c, ok := m[connect]; ok {
		conn = &c
		return
	}
	err = errors.New("can't found connect: " + connect)
	return
}

type Orm struct {
}

// 指定模型的入口
func Model(mo interface{}) *WithModel {
	return newWithModel(mo)
}

// 也可以不指定模型,但必须指定Table
func Table(table string) *WithOutModel {
	return newWithOutModel().Table(table)
}

func ExecSql(sql string, args ...interface{}) (affectCount int64, lastInsertId int64, err error) {
	return
}
func QuerySql(sql string, args ...interface{}) (has bool, data []map[string]interface{}, err error) {
	return
}

type ModelInfo struct {
	FieldMap         map[string]string // struct => db
	FieldTyp         map[string]string // 字段类型
	Table            string            // table name
	ConnectName      string            // connect name
	AutoPk           string            // 自增主键
	AutoFields       map[string]Auto
	Trans            map[string]Tran
	connectReadName  string
	connectWriteName string
}
type Tran struct {
	Typ string
}
type Auto struct {
	Where string // 当什么时候自动更新字段
	Typ   string // 目前只支持时间的自动更新
}

type Column struct {
	Name string
	Pk   string // "":不是pk, auto:自增pk
	Tran Tran   // 自动转换规则 json:string转换为field
	Auto Auto
}

// 存储模型信息
var modelInfo = map[string]ModelInfo{}

// 注册模型， 将字段对应写入map
func RegisterModel(prtModel interface{}) {
	RegisterModelCustom(prtModel, func(prtModel interface{}) ModelInfo {
		tag := "orm"
		fInfo := DecodeStruct(prtModel)
		table := Field2TagMap(fInfo, "table")["orm"]
		connect := Field2TagMap(fInfo, "connect")["orm"]
		fieldMap := Field2TagMap(fInfo, tag)
		fieldTyp := FieldType(fInfo)

		field2Db := map[string]string{}
		autoPk := ""
		autoFields := map[string]Auto{}
		trans := map[string]Tran{}
		for field, db := range fieldMap {
			column := DecodeColumn(db)
			if column.Name == "" {
				panic(errors.New(field + " Tag orm format error"))
			}
			field2Db[field] = column.Name
			if column.Pk == "auto" {
				autoPk = field
			}
			if column.Auto.Typ != "" {
				autoFields[field] = column.Auto
			}
			if column.Tran.Typ != "" {
				trans[field] = column.Tran
			}
		}

		m := ModelInfo{
			FieldMap:   field2Db,
			AutoPk:     autoPk,
			Table:      table,
			ConnectName:connect,
			AutoFields: autoFields,
			FieldTyp:   fieldTyp,
			Trans:      trans,
		}

		return m
	})
}

func RegisterModelCustom(prtModel interface{}, decoder func(prtModel interface{}) ModelInfo) {
	mInfo := decoder(prtModel)
	typ := reflect.TypeOf(prtModel).String()
	modelInfo[typ] = mInfo
}

// default,mysql,xxx:xxx
func RegisterDb(connect, driver, link string) {
	config[connect] = Connect{Url:link, Driver:driver}
}
