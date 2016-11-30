package orm

import (
	"errors"
	"fmt"
	"github.com/bysir-zl/bygo/util"
	"math"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const dbFieldName = "db"

type Model struct {
	config           Config
	out              interface{}                // 要输出的Model,可以是slice也可以是单个model
	sqls             []string                   // queried sqls

	connectName      string
	connectReadName  string
	connectWriteName string

	modelInfo        modelInfo

	fields           []string                   // to controller update select's fields
	where            map[string]([]interface{}) // condition, string=>params
	limit            [2]int                     // limit $0,$1
	order            []orderItem
	err              error                      // 在 where,limit等过程中出现的error
}

type orderItem struct {
	Field string
	Desc  string
}

func (p *Model) Field(fields ...string) *Model {
	tagFieldMap := p.modelInfo.tagMap.GetFieldMapByTagName(dbFieldName)
	if tagFieldMap != nil && len(tagFieldMap) != 0 {
		p.fields = make([]string, len(fields))
		for i, field := range fields {
			if ne, ok := tagFieldMap[field]; ok {
				field = ne
			}
			p.fields[i] = field
		}
	} else {
		p.fields = fields
	}
	return p
}

func (p *Model) Limit(skip int, len int) *Model {
	p.limit = [2]int{skip, len}
	return p
}

func (p *Model) OrderBy(field string, desc bool) *Model {
	if p.order == nil {
		p.order = []orderItem{}
	}
	tagFieldMap := p.modelInfo.tagMap.GetFieldMapByTagName(dbFieldName)

	if tagFieldMap != nil && len(tagFieldMap) != 0 {
		if ne, ok := tagFieldMap[field]; ok {
			field = ne
		}
	}

	item := orderItem{}
	item.Field = field
	if desc {
		item.Desc = "DESC"
	} else {
		item.Desc = ""
	}

	p.order = append(p.order, item)
	return p
}

func (p *Model) Where(condition string, values ...interface{}) *Model {
	if p.where == nil {
		p.where = map[string]([]interface{}){}
	}
	tagFieldMap := p.modelInfo.tagMap.GetFieldMapByTagName(dbFieldName)
	// replace `struct field name` to db field name
	if tagFieldMap != nil && len(tagFieldMap) != 0 {
		reg, err := regexp.Compile("`(.+?)`")
		if err != nil {
			p.err = errors.New("where string error")
		} else {
			condition = reg.ReplaceAllStringFunc(condition, func(in string) string {
				k := string(in)[1 : len(in) - 1] // 去掉左右`号
				var ne string = tagFieldMap[k]
				if ne == "" {
					p.err = errors.New("the where field(in '" + condition + "') " + k + " is undefined in model")
				}
				return "`" + ne + "`"
			})

		}
	}

	p.where[condition] = values
	if strings.Count(condition, "?") != len(values) {
		p.err = errors.New("where condition params len is must same as values len!")
	}

	return p
}

// 清除已经有的Where,fields,limit,order条件
func (p *Model) Reset() *Model {
	p.where = nil
	p.fields = nil
	p.limit = [2]int{}
	p.order = nil
	return p
}

// 读取执行的sql,用于调试,获取后将清空
func (p *Model) Sqls() []string {
	t := p.sqls
	p.sqls = []string{}
	return t
}

func (p *Model) saveSql(sql string, args ...interface{}) {
	if p.sqls == nil {
		p.sqls = []string{sql}
	} else {
		p.sqls = append(p.sqls, sql)
		lenSqls := len(p.sqls)
		if lenSqls > 10 {
			p.sqls = p.sqls[lenSqls - 10:]
		}
	}
}

func (p *Model) WhereIn(field string, params ...interface{}) *Model {
	if p.where == nil {
		p.where = map[string]([]interface{}){}
	}
	tagFieldMap := p.modelInfo.tagMap.GetFieldMapByTagName(dbFieldName)
	if tagFieldMap != nil && len(tagFieldMap) != 0 {
		if ne, ok := tagFieldMap[field]; ok {
			field = ne
		}
	}
	dataHolder := strings.Repeat(",?", len(params))
	dataHolder = dataHolder[1:]

	condition := field + " IN (" + dataHolder + ")"
	p.where[condition] = params
	return p
}

// 指定哪个连接
// connectName 必须是已经是在config/db配置好了的
func (p *Model) Connect(connectName string) *Model {
	c := p.config[connectName]
	if c.Port == 0 {
		p.err = errors.New("the connect `" + connectName + "` is undefined in dbConfig")
	} else {
		p.connectName = connectName
	}
	return p
}

// 指定哪个连接
// 并将传入的连接放入配置项, 下一次可直接使用
func (p *Model) ConnectAdd(connectName string, connect Connect) *Model {
	p.config[connectName] = connect
	p.connectName = connectName

	return p
}

//指定表
func (p *Model) Table(table string) *Model {
	p.modelInfo.table = table
	return p
}

// 获取连接
func (p *Model) getReadConnect() (conn *Connect, err error) {
	// 在未指定模型的时候这些值为空
	if p.out == nil {
		if p.connectReadName == "" {
			if p.connectName == "" {
				p.connectName = "default"
			}

			// 存在read,就保存
			readName := p.connectName + "_read"
			if c, ok := p.config[p.connectReadName]; ok {
				p.connectReadName = readName
				conn = &c
				return
			} else {
				// 不存在就默认
				p.connectReadName = p.connectName
			}
		}
	}

	// 打死都找不到就报错
	c, ok := p.config[p.connectReadName];
	if !ok {
		err = errors.New("the connect `" + p.connectReadName + "` is undefined in dbConfig")
		return
	}
	conn = &c
	return
}

func (p *Model) getWriteConnect() (conn *Connect, err error) {
	// 在未指定模型的时候这些值为空
	if p.out == nil {
		if p.connectWriteName == "" {
			if p.connectName == "" {
				p.connectName = "default"
			}

			// 存在read,就保存
			readName := p.connectName + "_read"
			if c, ok := p.config[p.connectWriteName]; ok {
				p.connectWriteName = readName
				conn = &c
				return
			} else {
				// 不存在就默认
				p.connectWriteName = p.connectName
			}
		}
	}

	// 打死都找不到就报错
	c, ok := p.config[p.connectWriteName];
	if !ok {
		err = errors.New("the connect `" + p.connectWriteName + "` is undefined in dbConfig")
		return
	}
	conn = &c
	return
}

// 原始方法 查询sql返回map
func (p *Model) QuerySql(sql string, args ...interface{}) (data []map[string]interface{}, err error) {
	data = nil
	c, err := p.getReadConnect()
	if err != nil {
		return
	}
	dbDriver, err := Singleton(c)
	if err != nil {
		return
	}
	out, err := dbDriver.Query(sql, args...)
	p.saveSql(sql, args...)
	if err != nil {
		return
	}
	data = out
	p.Reset()
	return
}

// 查询主方法,返回[]Map原数据给get和first使用
func (p *Model) QueryToMap() (data []map[string]interface{}, err error) {
	if p.err != nil {
		err = p.err
		return
	}

	sql, args, e := buildSelectSql(p.fields, p.modelInfo.table, p.where, p.order, p.limit)
	if e != nil {
		err = e
		return
	}

	dataMap, err := p.QuerySql(sql, args...)
	if err != nil {
		return
	}
	if len(dataMap) == 0 {
		return
	}

	data = dataMap
	return
}

//查询返回一个数组
func (p *Model) Get() (err error) {
	//从数组interface中获取一个元素
	mo := reflect.ValueOf(p.out).Type().Elem()

	if mo.String()[0] != '[' {
		err = errors.New("Get function need one slice(model) param")
		return
	}
	data, err := p.QueryToMap()
	if err != nil {
		return
	}

	if len(data) == 0 {
		err = NewNotFoundError()
		return
	}

	util.MapListToObjList(p.out, data, dbFieldName)
	return
}

func (p *Model) Page(page int, pageSize int) (pageData Page, err error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	} else if pageSize > 200 {
		pageSize = 40
	}

	p.limit = [2]int{(page - 1) * pageSize, pageSize}

	data, err := p.QueryToMap()
	if err != nil {
		return
	}

	util.MapListToObjList(p.out, data, dbFieldName)

	count, err := p.Count()
	if err != nil {
		return
	}
	pageTotal := int(math.Ceil(float64(count) / float64(pageSize)))
	pageData = Page{Total: count, Page: page, PageSize: pageSize, PageTotal: pageTotal}

	return
}

func (p *Model) PageWithOutTotal(page int, pageSize int) (pageData Page, err error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	} else if pageSize > 200 {
		pageSize = 40
	}

	p.limit = [2]int{(page - 1) * pageSize, pageSize}
	data, err := p.QueryToMap()
	if err != nil {
		return
	}

	util.MapListToObjList(p.out, data, dbFieldName)
	pageData = Page{Page: page, PageSize: pageSize}

	return
}

func (p *Model) First() (err error) {
	p.limit = [2]int{0, 1}
	datas, err := p.QueryToMap()
	if err != nil {
		return
	}
	if datas == nil || len(datas) == 0 {
		err = NewNotFoundError()
		return
	}

	util.MapToObj(p.out, datas[0], dbFieldName)
	return
}

func (p *Model) Count() (count int64, err error) {

	sql, args, e := buildCountSql(p.modelInfo.table, p.where)
	if e != nil {
		err = e
		return
	}

	data, e := p.QuerySql(sql, args...)
	if e != nil {
		err = e
		return
	}
	if len(data) == 0 {
		err = errors.New("query sql success,but return len is 0")
	}
	count = data[0]["count"].(int64)

	return
}


// 取得在method操作时需要自动填充的字段与值
func (p *Model) GetAutoSetField(method string) (needSet map[string]interface{}, err error) {
	autoField := p.modelInfo.tagMap.GetFieldMapByTagName("auto")
	if len(autoField) != 0 {
		needSet = map[string]interface{}{}
		for field, tagVal := range autoField {
			// time,insert
			tyAme := strings.Split(tagVal, ",")
			methods := tyAme[1]

			if util.ItemInArray(method, strings.Split(methods, "|")) {
				if tyAme[0] == "timestr" {
					needSet[field] = time.Now().Format("2006-01-02 15:04:05")
				} else if tyAme[0] == "timeint" {
					needSet[field] = time.Now().Unix()
				}
			}
		}
	}

	return
}

func (p *Model) Insert() (err error) {
	if p.out == nil {
		err = errors.New("Insert func must called with model is not nil")
		return
	}
	if p.err != nil {
		err = p.err
		return
	}

	// 获取应该存数据库的键值对 数据库字段->值映射
	saveData := map[string]interface{}{}

	// 字段->值映射
	mapper := util.ObjToMap(p.out, dbFieldName)
	for key, value := range mapper {
		// 指定了fields 就只更新指定字段
		if p.fields != nil {
			if !util.ItemInArray(key, p.fields) {
				continue
			}
		}
		// 在插入的时候过滤空值
		if util.IsEmptyValue(value) {
			continue
		}
		saveData[key] = value
	}
	autoSet, e := p.GetAutoSetField("insert")
	if e != nil {
		err = e
		return
	}

	tagFieldMap := p.modelInfo.tagMap.GetFieldMapByTagName(dbFieldName)
	if autoSet != nil && len(autoSet) != 0 {
		for k, v := range autoSet {
			if k2, ok := tagFieldMap[k]; ok {
				saveData[k2] = v
			}
		}

		//将自动添加的字段附加到model里，方便返回
		util.MapToObj(p.out, autoSet, "")
	}

	sql, args, e := buildInsertSql(p.modelInfo.table, saveData)
	if e != nil {
		err = e
		return
	}

	_, _insertId, _err := p.ExecSql(sql, args...)
	if _err != nil {
		err = _err
		return
	}

	//找到主键，并且赋值为lastInsertId

	if p.modelInfo.autoPk != "" {
		ma := map[string]interface{}{}
		ma[p.modelInfo.autoPk] = _insertId

		util.MapToObj(p.out, ma, "")
	}

	return
}

func (p *Model) InsertMap(maps map[string]interface{}) (pk int64, err error) {
	sql, args, e := buildInsertSql(p.modelInfo.table, maps)
	if e != nil {
		err = e
		return
	}
	if p.err != nil {
		err = p.err
		return
	}
	if p.modelInfo.table == "" {
		err = errors.New("the model has not `table` field or Tag.name")
		return
	}

	_, pk, err = p.ExecSql(sql, args...)
	return
}

func (p *Model) Update() (count int64, err error) {
	if p.out == nil {
		err = errors.New("Insert func must called with model is not nil")
		return
	}
	if p.where == nil {
		err = errors.New("you need set condition in Where()")
		return
	}
	if p.err != nil {
		err = p.err
		return
	}

	//获取应该存数据库的键值对 数据库字段->值映射
	saveData := map[string]interface{}{}

	//字段->值映射
	mapper := util.ObjToMap(p.out, dbFieldName)
	for key, value := range mapper {
		//指定了fields 就只更新指定字段
		if p.fields != nil {
			if !util.ItemInArray(key, p.fields) {
				continue
			}
		}

		saveData[key] = value
	}
	autoSet, e := p.GetAutoSetField("update")
	if e != nil {
		err = e
		return
	}

	tagFieldMap := p.modelInfo.tagMap.GetFieldMapByTagName(dbFieldName)
	if autoSet != nil && len(autoSet) != 0 {
		for k, v := range autoSet {
			if k2, ok := tagFieldMap[k]; ok {
				saveData[k2] = v
			}
		}

		//将自动添加的字段附加到model里，方便返回
		util.MapToObj(p.out, autoSet, "")
	}

	sql, args, e := buildUpdateSql(p.modelInfo.table, saveData, p.where)
	if e != nil {
		err = e
		return
	}

	c, _, e := p.ExecSql(sql, args...)
	if e != nil {
		err = e
		return
	}
	count = c

	return
}

func (p *Model) UpdateMap(mapper map[string]interface{}) (count int64, err error) {
	if p.where == nil {
		err = errors.New("you need set condition in Where()")
		return
	}
	if p.err != nil {
		err = p.err
		return
	}
	if p.modelInfo.table == "" {
		err = errors.New("the model has not `table` field or Tag.name")
		return
	}

	sql, args, e := buildUpdateSql(p.modelInfo.table, mapper, p.where)
	if e != nil {
		err = e
		return
	}

	c, _, e := p.ExecSql(sql, args...)
	if e != nil {
		err = e
		return
	}
	count = c

	return
}

func (p *Model) Delete() (count int64, err error) {
	if p.where == nil {
		err = errors.New("you need set condition in Where()")
		return
	}
	if p.err != nil {
		err = p.err
		return
	}

	sql, args, e := buildDeleteSql(p.modelInfo.table, p.where)
	if e != nil {
		err = e
		return
	}

	c, _, e := p.ExecSql(sql, args...)
	if e != nil {
		err = e
		return
	}
	count = c
	return
}

func (p *Model) ExecSql(sql string, args ...interface{}) (affectCount int64, lastInsertId int64, err error) {
	c, err := p.getWriteConnect()
	if err != nil {
		return
	}
	dbDriver, err := Singleton(c)
	if err != nil {
		return
	}
	att, insertId, err := dbDriver.Exec(sql, args...)
	p.saveSql(sql, args...)
	if err != nil {
		return
	}

	lastInsertId = insertId
	affectCount = att

	if p.sqls == nil {
		p.sqls = []string{sql}
	} else {
		p.sqls = append(p.sqls, sql)
	}
	p.Reset()
	return
}

func buildSelectSql(fields []string, tableName string,
where map[string]([]interface{}), order []orderItem, limit [2]int, ) (sql string, args []interface{}, err error) {

	args = []interface{}{}
	sql = "SELECT "

	//field
	fieldString := "*"
	if fields != nil && len(fields) != 0 {
		fieldString = strings.Join(fields, ",")
	}

	sql = sql + fieldString + " "

	//table
	sql = sql + "FROM `" + tableName + "` "

	//where
	if where != nil {
		whereString, as := buildWhere(where)
		for _, a := range as {
			args = append(args, a)
		}

		sql = sql + "WHERE " + whereString + " "
	}

	//orderBy
	if order != nil {
		orderString := ""
		for _, value := range order {
			orderString = orderString + "," + value.Field + " " + value.Desc
		}
		orderString = orderString[1:]

		sql = sql + "ORDER BY " + orderString + " "
	}

	//limit
	if limit[0] != 0 || limit[1] != 0 {
		sql = sql + "LIMIT " + fmt.Sprintf("%d,%d", limit[0], limit[1]) + " "
	}

	return
}

func buildInsertSql(tableName string, saveData map[string]interface{}) (sql string, args []interface{}, err error) {
	if len(saveData) == 0 {
		err = errors.New("no save data on INSERT")
		return
	}

	args = []interface{}{}
	sql = "INSERT INTO " + tableName + " ("

	fieldsStr := ""
	holderStr := ""

	for key, value := range saveData {
		fieldsStr = fieldsStr + ", " + key
		holderStr = holderStr + ", ?"

		args = append(args, value)
	}

	fieldsStr = fieldsStr[2:]
	holderStr = holderStr[2:]

	sql = sql + fieldsStr + " ) VALUES ( " + holderStr + " )"

	return
}

func buildUpdateSql(tableName string, saveData map[string]interface{}, where map[string]([]interface{})) (sql string, args []interface{}, err error) {

	if len(saveData) == 0 {
		err = errors.New("no save data on INSERT")
		return
	}

	args = []interface{}{}
	sql = "UPDATE " + tableName + " SET "

	//value
	fieldsStr := ""
	for key, value := range saveData {
		fieldsStr = fieldsStr + ", " + key + "= ?"

		args = append(args, value)
	}

	fieldsStr = fieldsStr[2:]
	sql = sql + fieldsStr + " "

	//where
	if where != nil {
		whereString, as := buildWhere(where)
		for _, a := range as {
			args = append(args, a)
		}
		sql = sql + "WHERE " + whereString + " "
	}

	return
}

func buildDeleteSql(tableName string, where map[string]([]interface{})) (sql string, args []interface{}, err error) {
	args = []interface{}{}
	sql = "DELETE FROM " + tableName + " "

	//where
	if where != nil {
		whereString, as := buildWhere(where)
		args = as
		sql = sql + "WHERE (" + whereString + ") "
	}

	return
}

func buildCountSql(tableName string, where map[string]([]interface{})) (sql string, args []interface{}, err error) {
	sql = "SELECT COUNT(*) as count FROM " + tableName + " "

	//where
	if where != nil {
		whereString, as := buildWhere(where)
		args = as
		sql = sql + "WHERE (" + whereString + ") "
	}

	return
}

// 生成where 条件
func buildWhere(where map[string]([]interface{})) (whereString string, args []interface{}) {
	if where != nil {
		args = []interface{}{}
		whereString = " "

		for key, vaules := range where {
			whereString = whereString + " AND ( " + key + " )"
			for _, value := range vaules {
				args = append(args, value)
			}
		}

		whereString = whereString[5:]
	}

	return
}

type modelInfo struct {
	tagMap           util.FieldTagMapper
	table            string
	connectName      string
	connectReadName  string
	connectWriteName string
	autoPk           string
}

func (p *modelInfo) load(m interface{}, config Config) (err error) {
	if m != nil {
		// 如果是slice
		x := reflect.TypeOf(m).Elem()
		if x.Kind() == reflect.Slice {
			mo := reflect.New(x.Elem()).Interface()
			p.tagMap = util.GetTagMapperFromPool(mo)
		} else {
			p.tagMap = util.GetTagMapperFromPool(m)
		}

		// 获取表
		p.table = p.tagMap.GetFieldMapByTagName("table")["orm"]
		if p.table == "" {
			err = errors.New("the model has not `table` field or Tag.name")
			return
		}

		// 获得默认连接
		c := p.tagMap.GetFieldMapByTagName("connect")["orm"]
		if c == "" {
			c = "default"
		}
		p.connectName = c

		// 获取自增主键
		pkMap := p.tagMap.GetFieldMapByTagName("pk")
		if pkMap != nil && len(pkMap) != 0 {
			for key, value := range pkMap {
				if value == "auto" {
					p.autoPk = key
				}
			}
		}
	} else {
		p.connectName = "default"
	}

	if _, ok := config[p.connectName]; !ok {
		err = errors.New("the connect `" + p.connectName + "` is undefined in dbConfig")
		return
	}
	// 获取read与write
	connRead := p.connectName + "_read"
	if _, ok := config[connRead]; ok {
		p.connectReadName = connRead
	} else {
		p.connectReadName = p.connectName
	}
	connWrite := p.connectName + "_write"
	if _, ok := config[connWrite]; ok {
		p.connectWriteName = connWrite
	} else {
		p.connectWriteName = p.connectName
	}
	return
}

func (p *Model) Model(m interface{}) *Model {
	p.Reset()
	p.out = m
	p.modelInfo.load(m, p.config)
	p.connectName = p.modelInfo.connectName
	p.connectReadName = p.modelInfo.connectReadName
	p.connectWriteName = p.modelInfo.connectWriteName
	return p
}