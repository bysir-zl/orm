package orm

import (
	"reflect"
	"errors"
	"github.com/bysir-zl/bygo/util"
	"strings"
	"time"
	"encoding/json"
)

type WithModel struct {
	WithOutModel
	prtModel  interface{}
	modelInfo ModelInfo
}

func newWithModel(prtModel interface{}) *WithModel {
	w := &WithModel{
		prtModel:prtModel,
	}
	typ := reflect.TypeOf(prtModel).String()
	mInfo, ok := modelInfo[typ]
	if !ok {
		w.err = errors.New("can't found model " + typ + ",forget register?")
	} else {
		w.modelInfo = mInfo
	}
	return w
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
	p.TranSaveData(&fieldData)

	// mapToDb
	dbData := map[string]interface{}{}
	for k, v := range fieldData {
		dbKey, ok := p.modelInfo.FieldMap[k]
		if ok {
			dbData[dbKey] = v
		}
	}

	id, err := p.WithOutModel.
		Table(p.modelInfo.Table).
		Connect(p.modelInfo.ConnectName).
		Insert(dbData)
	if err != nil {
		return
	}
	id = id
	return
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
					if strings.Contains(p.modelInfo.FieldTyp[field], "int") {
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

// 将db的值 转换为struct的值
func (p *WithModel) TranStructData(saveData *map[string]interface{}) (err error) {
	for field, t := range p.modelInfo.Trans {
		v, ok := (*saveData)[field]
		if !ok {
			continue
		}

		switch t.Typ {
		case "json":
			s, ok := util.Interface2String(v, true)
			if !ok {
				err = errors.New(field + " is't string, can't tran 'json'")
				return
			}
			var value interface{} = 1
			e := json.Unmarshal(util.S2B(s), value)
			if e != nil {
				err = errors.New(field + " value " + s + ", can't Unmarshal")
				return
			}
			(*saveData)[field] = value
		case "time":
			if strings.Contains(p.modelInfo.FieldTyp[field], "int") {
				s, _ := util.Interface2Int(v, true)
				t := time.Unix(s, 0).Format("2006-01-02 15:04:05")
				(*saveData)[field] = t
			} else {
				s, ok := util.Interface2String(v, true)
				if !ok {
					err = errors.New(field + " is't string, can't tran 'time'")
					return
				}
				t, e := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
				if e != nil {
					err = e
					return
				}
				(*saveData)[field] = t.Unix()
			}
		}
	}
	return
}

// 将db的值 转换为struct的值
func (p *WithModel) TranSaveData(saveData *map[string]interface{}) (err error) {
	for field, t := range p.modelInfo.Trans {
		v, ok := (*saveData)[field]
		if !ok {
			continue
		}

		switch t.Typ {
		case "json":
			bs, e := json.Marshal(v)
			if e != nil {
				err = e
				return
			}
			(*saveData)[field] = util.B2S(bs)
		case "time":
			if strings.Contains(p.modelInfo.FieldTyp[field], "int") {
				s, _ := util.Interface2Int(v, true)
				t := time.Unix(s, 0).Format("2006-01-02 15:04:05")
				(*saveData)[field] = t
			} else {
				s, ok := util.Interface2String(v, true)
				if !ok {
					err = errors.New(field + " is't string, can't tran 'time'")
					return
				}
				t, e := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
				if e != nil {
					err = e
					return
				}
				(*saveData)[field] = t.Unix()
			}
		}
	}
	return
}
