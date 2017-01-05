package tests

import (
	"log"
	"reflect"
	"testing"
)

type X struct {
	Name string
}

func TestJson(t *testing.T) {
	//j := `{"Name":"zl"}`
	var x *X
	var y interface{} = &x
	//err := json.Unmarshal([]byte(j), &y)
	//if err != nil {
	//	t.Fatal(err)
	//}

	//log.Print(x.Name)

	ty := reflect.ValueOf(&y)
	tttt := ty.Elem().Elem().Elem().Type()
	log.Print(tttt)

	t2 := reflect.New(tttt.Elem()).Elem()
	log.Print(t2.Type())

	log.Print(t2.FieldByName("Name").Type())

	yy := indirect(ty,false)

	//log.Print(ty.Elem().Elem().Elem().Elem().FieldByName("Name").String())
	yy.FieldByName("Name").SetString("zxv")
	log.Print(x.Name)
}

func indirect(v reflect.Value, decodingNull bool) (reflect.Value) {
	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {
		xx:=v.Type().String()
		xx =xx
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Ptr) {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && decodingNull && v.CanSet() {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		v = v.Elem()
	}
	return  v
}


