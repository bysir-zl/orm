package orm

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bysir-zl/bygo/util"
	"reflect"
	"strings"
	"time"
	"github.com/bysir-zl/bygo/log"
)

type WithModel struct {
	WithOutModel
	modelInfo ModelInfo

	link map[string]linkData // objFieldName => linkData

	preLinkData map[InOneSql]PreLink
}

func newWithModel(ptrModel interface{}) *WithModel {

	w := &WithModel{}

	typ := reflect.TypeOf(ptrModel).String()
	typ = strings.Replace(typ, "*", "", -1)
	typ = strings.Replace(typ, "[]", "", -1)
	mInfo, ok := modelInfo[typ]
	if !ok {
		w.err = errors.New("can't found model " + typ + ",forget register?")
	} else {
		w.modelInfo = mInfo
	}
	w.table = w.modelInfo.Table
	w.connect = w.modelInfo.ConnectName
	return w
}

func (p *WithModel) checkPtrModel(ptrModel interface{}) interface{} {
	m := reflect.ValueOf(ptrModel)
	if m.Kind() != reflect.Ptr {
		p.err = errors.New("ptrModel is't a ptr interface")
		return nil
	}

	for {
		if m.Elem().Kind() == reflect.Ptr {
			m = m.Elem()
		} else {
			break
		}
	}
	//if !m.Elem().IsNil(){
	//	m.Elem().Set(reflect.New(m.Elem().Type()).Elem())
	//}
	//info(m.Interface())
	return m.Interface()
}

func (p *WithModel) Table(table string) *WithModel {
	p.WithOutModel.Table(table)
	return p
}

func (p *WithModel) Connect(connect string) *WithModel {
	p.WithOutModel.Connect(connect)
	return p
}

func (p *WithModel) Fields(fields ...string) *WithModel {
	p.WithOutModel.Fields(fields...)
	return p
}

func (p *WithModel) Where(condition string, args ...interface{}) *WithModel {
	p.WithOutModel.Where(condition, args...)
	return p
}

func (p *WithModel) WhereIn(condition string, args ...interface{}) *WithModel {
	p.WithOutModel.WhereIn(condition, args...)
	return p
}

func (p *WithModel) Insert(prtModel interface{}) (err error) {
	if p.err != nil {
		err = p.err
		return
	}

	fieldData := map[string]interface{}{}
	// 读取保存的键值对
	mapper := util.ObjToMap(prtModel, "")
	for k, v := range mapper {
		// 在插入的时候过滤空值
		if !util.IsEmptyValue(v) {
			fieldData[k] = v
		}
	}
	// 自动添加字段
	autoSet, err := p.GetAutoSetField("insert")
	if err != nil {
		return
	}
	if autoSet != nil && len(autoSet) != 0 {
		for k, v := range autoSet {
			fieldData[k] = v
		}
		// 将自动添加的字段附加到model里，方便返回
		util.MapToObj(prtModel, autoSet, "")
	}

	// 转换值
	p.tranSaveData(&fieldData)

	// mapToDb
	dbData := map[string]interface{}{}
	for k, v := range fieldData {
		dbKey, ok := p.modelInfo.FieldMap[k]
		if ok {
			dbData[dbKey] = v
		}
	}

	id, err := p.WithOutModel.
		Insert(dbData)
	if err != nil {
		return
	}
	id = id
	return
}

func (p *WithModel) Select(ptrSliceModel interface{}) (has bool, err error) {
	if p.err != nil {
		err = p.err
		return
	}
	result, has, err := p.WithOutModel.
		Select()
	if err != nil || !has {
		return
	}

	col2Field := util.ReverseMap(p.modelInfo.FieldMap) // 数据库字段to结构体字段

	// 是数组还是一个对象
	if strings.Contains(reflect.TypeOf(ptrSliceModel).String(), "[") {
		structData := make([]map[string]interface{}, len(result))
		for i, re := range result {
			structItem := make(map[string]interface{}, len(re))
			for k, v := range re {
				// 字段映射
				if structField, ok := col2Field[k]; ok {
					structItem[structField] = v
				}
			}
			// 转换值
			p.tranStructData(&structItem)
			p.preLink(&structItem)
			structData[i] = structItem
		}
		p.doLinkMulti(&structData)
		errInfo := util.MapListToObjList(ptrSliceModel, structData, "")
		if errInfo != "" {
			warn("table("+p.table+")", "tran", errInfo)
		}
	} else {
		resultItem := result[0]
		structItem := make(map[string]interface{}, len(resultItem))
		for k, v := range resultItem {
			// 字段映射
			if structField, ok := col2Field[k]; ok {
				structItem[structField] = v
			}
		}
		// 转换值
		p.tranStructData(&structItem)
		p.doLink(&structItem)
		_, errInfo := util.MapToObj(ptrSliceModel, structItem, "")
		if errInfo != "" {
			warn("table("+p.table+")", "tran", errInfo)
		}
	}

	return
}

type linkData struct {
	ExtCondition string
	Column       []string
}

// 连接对象
func (p *WithModel) Link(field string, extCondition string, columns []string) *WithModel {
	if p.link == nil {
		p.link = map[string]linkData{}
	}
	p.link[field] = linkData{ExtCondition:extCondition, Column:columns}
	return p
}

// 取得在method操作时需要自动填充的字段与值
func (p *WithModel) GetAutoSetField(method string) (needSet map[string]interface{}, err error) {
	autoFields := p.modelInfo.AutoFields
	if len(autoFields) != 0 {
		needSet = map[string]interface{}{}
		for field, auto := range autoFields {
			if util.ItemInArray(method, strings.Split(auto.When, "|")) {
				if auto.Typ == "time" {
					// 判断类型
					if strings.Contains(p.modelInfo.FieldTyp[field].String(), "int") {
						needSet[field] = time.Now().Unix()
					} else {
						needSet[field] = time.Now().Format("2006-01-02 15:04:05")
					}
				}
			}
		}
	}
	return
}

type PreLink struct {
	Args   []interface{} // 参数
	Column []string      // 要查询的字段
	Model  interface{}   // 要查询的模型(一个struct)
	ArgKey string        // 要连接的字段
}

// 能组装成一个条sql的
type InOneSql struct {
	Table      string
	WhereField string
}

func (p *InOneSql) String() string {
	return p.Table + "|" + p.WhereField
}

// 准备link
func (p *WithModel) preLink(data *map[string]interface{}) {
	if p.link == nil || len(p.link) == 0 {
		return
	}

	if p.preLinkData == nil {
		p.preLinkData = map[InOneSql]PreLink{} // key => PreLink
	}

	// 要link的字段
	for field, linkData := range p.link {
		// 判断有无field
		typ, ok := p.modelInfo.FieldTyp[field]
		if !ok {
			err := fmt.Errorf("have't %s field when link", field)
			warn("table("+p.table+")", err)
			continue
		}
		// 判断有无link属性
		link, ok := p.modelInfo.Links[field]
		if !ok {
			err := fmt.Errorf("have't link tag, plase use tag `orm:\"link(RoleId,Id)\"` on %s", field)
			warn("table("+p.table+")", err)
			continue
		}
		// 检查在原来的data中有无要连接的键的值
		//log.Info("xx",data)
		val := (*data)[link.SelfKey]
		if val == nil {
			err := fmt.Errorf("have't '%s' value to link", link.SelfKey)
			warn("table("+p.table+")", err)
			continue
		}

		linkPtrValue := reflect.New(typ)

		where := "`" + link.LinkKey + "` in (?) AND " + linkData.ExtCondition
		one := InOneSql{
			Table:     newWithModel(linkPtrValue.Interface()).GetTable(),
			WhereField:strings.Trim(where, "AND "),
		}
		args := []interface{}{}

		// 要连接的是否是一个slice
		if typ.Kind() == reflect.Slice {
			valValue := reflect.ValueOf(val)
			//info( valValue.Kind())
			if valValue.Kind() != reflect.Slice {
				err := fmt.Errorf("'%s' value is not slice to link slice", link.SelfKey)
				warn("table("+p.table+")", err)
				continue
			}
			vl := valValue.Len()
			vs := make([]interface{}, vl)
			for i := 0; i < vl; i++ {
				vs[i] = valValue.Index(i).Interface()
			}
			args = append(args, vs...)
		} else {
			args = append(args, val)
		}

		pre := PreLink{}
		pre.Model = util.GetElemInterface(linkPtrValue)
		pre.ArgKey = link.LinkKey
		if o, ok := p.preLinkData[one]; !ok {
			pre.Column = linkData.Column
			pre.Args = args
			p.preLinkData[one] = pre
		} else {
			pre.Args = append(o.Args, args...)
			p.preLinkData[one] = pre
		}
	}
	return
}

func (p *WithModel) doLinkMulti(data *[]map[string]interface{}) {
	linkResult := map[string]map[interface{}]map[string]interface{}{} // onesql => key => model

	// 查询数据库
	for oneSql, pre := range p.preLinkData {
		//reflect.New(reflect.SliceOf(reflect.TypeOf(pre.Model)))
		pre.Args = UnDuplicate(pre.Args)
		rs, _, err := newWithOutModel().
			Connect(p.connect).Table(oneSql.Table).
			WhereIn(oneSql.WhereField, pre.Args...).
			Select()
		if err != nil {
			info("err", err)
			continue
		}
		for i, l := 0, len(rs); i < l; i++ {
			r := rs[i]
			key, ok := r[pre.ArgKey]
			one := oneSql.String()
			if _, ok := linkResult[one]; !ok {
				linkResult[one] = map[interface{}]map[string]interface{}{}
			}
			if ok {
				linkResult[one][key] = r
			}
		}
	}
	for k, v := range linkResult {
		info("sbsb", k, v, )
	}

	for index, item := range *data {
		// 要link的字段
		for field, linkData := range p.link {
			// 判断有无field
			typ, _ := p.modelInfo.FieldTyp[field]

			// 判断有无link属性
			link, _ := p.modelInfo.Links[field]

			// 检查在原来的data中有无要连接的键的值
			//log.Info("xx",data)
			val := item[link.SelfKey]

			linkPtrValue := reflect.New(typ)

			where := "`" + link.LinkKey + "` in (?) AND " + linkData.ExtCondition
			one := InOneSql{
				Table:     newWithModel(linkPtrValue.Interface()).GetTable(),
				WhereField:strings.Trim(where, "AND "),
			}
			// 要连接的是否是一个slice
			has := false
			if typ.Kind() == reflect.Slice {
				valValue := reflect.ValueOf(val)
				//info( valValue.Kind())
				if valValue.Kind() != reflect.Slice {
					err := fmt.Errorf("'%s' value is not slice to link slice", link.SelfKey)
					warn("table("+p.table+")", err)
					continue
				}
				vl := valValue.Len()
				maps := []map[string]interface{}{}
				for i := 0; i < vl; i++ {
					k := valValue.Index(i).Interface()
					if ks, ok := linkResult[one.String()]; ok {
						for kkk,vvv:=range ks{
							info("test",reflect.TypeOf(k).String(),vvv,k,kkk==k)
						}
						info("ks", ks,k,ks[k])
						if ms, ok := ks[k]; ok {
							maps = append(maps, ms)
						}
					}
				}
				info("link", maps)
				e := util.MapListToObjList(linkPtrValue.Interface(), maps, "")
				if e != "" {
					has = true
				}
			} else {
				if ks, ok := linkResult[one.String()]; ok {
					if ms, ok := ks[val]; ok {
						_, e := util.MapToObj(linkPtrValue.Interface(), ms, "")
						if e != "" {
							has = true
						}
					}
				}
			}

			if has {
				(*data)[index][field] = linkPtrValue.Elem().Interface()
			}
		}
	}

}

// 连接对象
// todo 在需要多次link的时候, 优化查询相同表(where in)
func (p *WithModel) doLink(data *map[string]interface{}) {
	p.preLinkData = nil

	if p.link == nil || len(p.link) == 0 {
		return
	}
	//linkData:= map[interface{}]interface{}{} // 已经查询好的数据

	log.Info("sbsb", p.preLinkData)

	// 要link的字段
	for field, linkData := range p.link {
		// 判断有无field
		typ, ok := p.modelInfo.FieldTyp[field]
		if !ok {
			err := fmt.Errorf("have't %s field when link", field)
			warn("table("+p.table+")", err)
			continue
		}
		// 判断有无link属性
		link, ok := p.modelInfo.Links[field]
		if !ok {
			err := fmt.Errorf("have't link tag, plase use tag `orm:\"link(RoleId,Id)\"` on %s", field)
			warn("table("+p.table+")", err)
			continue
		}

		linkPtrValue := reflect.New(typ)

		// 检查在原来的data中有无要连接的键的值
		val := (*data)[link.SelfKey]
		if val == nil {
			err := fmt.Errorf("have't '%s' value to link", link.SelfKey)
			warn("table("+p.table+")", err)
			continue
		}

		m := newWithModel(linkPtrValue.Interface()).Fields(linkData.Column...)
		// 要连接的是否是一个slice
		if typ.Kind() == reflect.Slice {
			valValue := reflect.ValueOf(val)
			//info( valValue.Kind())
			if valValue.Kind() != reflect.Slice {
				err := fmt.Errorf("'%s' value is not slice to link slice", link.SelfKey)
				warn("table("+p.table+")", err)
				continue
			}
			vl := valValue.Len()
			vs := make([]interface{}, vl)
			for i := 0; i < vl; i++ {
				vs[i] = valValue.Index(i).Interface()
			}

			m = m.WhereIn("`"+link.LinkKey+"` in (?)", vs...)
		} else {
			m = m.Where("`"+link.LinkKey+"` = ?", val)
		}

		has, err := m.Select(linkPtrValue.Interface())

		if err != nil {
			warn("table("+p.table+")", err)
			continue
		}
		if has {
			(*data)[field] = linkPtrValue.Elem().Interface()
		}
	}

	return
}

// 将db的值 转换为struct的值
func (p *WithModel) tranStructData(data *map[string]interface{}) {
	for field, t := range p.modelInfo.Trans {
		v, ok := (*data)[field]
		if !ok {
			continue
		}

		switch t.Typ {
		case "json":
			s, ok := util.Interface2String(v, true)
			if !ok {
				err := errors.New(field + " is't string, can't tran 'json'")
				warn("table("+p.table+")", "tran", err)
				continue
			}

			ptrValue := reflect.New(p.modelInfo.FieldTyp[field])
			err := json.Unmarshal(util.S2B(s), ptrValue.Interface())
			if err != nil {
				warn("table("+p.table+")", "tran"+" field("+field+")", err)
				delete(*data, field)
			} else {
				value := ptrValue.Elem().Interface()
				(*data)[field] = value
			}
		case "time":
			if strings.Contains(p.modelInfo.FieldTyp[field].String(), "int") {
				// 如果struct的字段是int型的,还要转换,则数据库里的是string型的
				// timeString  => int
				s, ok := util.Interface2String(v, true)
				if !ok {
					err := errors.New(field + " is't string, can't tran 'time'")
					warn("table("+p.table+")", "tran", err)
					continue
				}
				t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
				if err != nil {
					warn("table("+p.table+")", "tran", err)
					continue
				}
				(*data)[field] = t.Unix()
			} else {
				// int => timeString
				s, _ := util.Interface2Int(v, true)
				t := time.Unix(s, 0).Format("2006-01-02 15:04:05")
				(*data)[field] = t
			}
		}
	}
	return
}

// 将struct的值 转换为db的值
func (p *WithModel) tranSaveData(saveData *map[string]interface{}) {
	for field, t := range p.modelInfo.Trans {
		v, ok := (*saveData)[field]
		if !ok {
			continue
		}

		switch t.Typ {
		case "json":
			// object => jsonString
			bs, err := json.Marshal(v)
			if err != nil {
				warn("table("+p.table+")", "tran", err)
				continue
			}
			(*saveData)[field] = util.B2S(bs)
		case "time":
			if strings.Contains(p.modelInfo.FieldTyp[field].String(), "int") {
				// int => timeString
				s, _ := util.Interface2Int(v, true)
				t := time.Unix(s, 0).Format("2006-01-02 15:04:05")
				(*saveData)[field] = t
			} else {
				// timeString => int
				s, ok := util.Interface2String(v, true)
				if !ok {
					err := errors.New(field + " is't string, can't tran 'time'")
					warn("table("+p.table+")", "tran", err)
					continue
				}
				t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
				if err != nil {
					warn("table("+p.table+")", "tran", err)
					continue
				}
				(*saveData)[field] = t.Unix()
			}
		}
	}
	return
}
