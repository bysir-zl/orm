package orm

import (
	"reflect"
	"strings"
)

type FieldInfo struct {
	Name         string
	Typ          string
	CanInterface bool
	Tags         map[string]string
}

func DecodeStruct(prtStruct interface{}) map[string]FieldInfo {
	v := reflect.Indirect(reflect.ValueOf(prtStruct))
	t := v.Type()
	fieldNum := v.NumField()
	result := make(map[string]FieldInfo, fieldNum)
	for i := 0; i < fieldNum; i++ {
		field := t.Field(i)
		tags := EncodeTag(string(field.Tag))
		result[field.Name] = FieldInfo{
			Typ:         field.Type.String(),
			CanInterface:v.Field(i).CanInterface(),
			Name:        field.Name,
			Tags:        tags,
		}
	}
	return result
}

func EncodeTag(tag string) (data map[string]string) {
	data = map[string]string{}
	if tag == "" {
		return
	}
	for _, item := range strings.Split(tag, " ") {
		if item == "" {
			continue
		}
		key := strings.Split(item, ":")[0]
		value := strings.Split(item, "\"")[1]
		data[key] = value
	}

	return
}

func Field2TagMap(fieldInfo map[string]FieldInfo, tag string) map[string]string {
	result := map[string]string{}
	for _, info := range fieldInfo {
		for k, v := range info.Tags {
			if k == tag {
				result[info.Name] = v
			}
		}
	}
	return result
}

func FieldType(fieldInfo map[string]FieldInfo) map[string]string {
	result := map[string]string{}
	for _, info := range fieldInfo {
		result[info.Name] = info.Typ
	}
	return result
}

func DecodeColumn(dbData string) *Column {
	c := &Column{}
	ds := strings.Split(dbData, ";")
	l := len(ds)
	if l == 0 {
		return c
	}
	for i := 0; i < l; i++ {
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
		case "col":
			c.Name = values[0]
		case "tran":
			c.Tran = Tran{
				Typ:values[0],
			}
		case "auto":
			c.Auto = Auto{
				When:  values[0],
				Typ:   values[1],
			}
		}

	}

	return c
}
