package commonlib

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// 使用map填充struct结构体
func FillStruct(data map[string]string, obj interface{}) error {
	for k, v := range data {
		tagName := GetFieldNameByTagName(obj, k)
		if tagName == "" {
			continue
		}
		err := SetField(obj, tagName, v)
		if err != nil {
			// 数据类型不正确,继续下一个字段的处理
			continue
			//return err
		}
	}
	return nil
}

// 根据标签名(数据库字段名)获取结构体属性名
func GetFieldNameByTagName(obj interface{}, tagName string) string {
	st := reflect.TypeOf(obj).Elem()
	for i := 0; i < st.NumField(); i++ {
		// 获取数据库表字段名
		fieldTagName := GetStructTagNameOfField(st, i)
		if tagName == fieldTagName {
			return st.Field(i).Name
		}
	}

	return ""
}

// 获取结构体标签
func GetStructTagName(rt reflect.Type, fieldIndex int, tagName string) string {
	field := rt.Field(fieldIndex)
	//Log.Debug("field numbers: ", rt.NumField(), "  st name: ", rt.Name(), "  field name: ", field.Type, "  field object: ", field.Name)

	return field.Tag.Get(tagName)
}

// 获取结构体标签为field的值
func GetStructTagNameOfField(rt reflect.Type, fieldIndex int) string {
	return GetStructTagName(rt, fieldIndex, "field")
}

// 用map的值替换结构的值
func SetField(obj interface{}, name string, value interface{}) error {
	// 结构体属性值
	structValue := reflect.ValueOf(obj).Elem()
	// 结构体单个属性值
	structFieldValue := structValue.FieldByName(name)

	if !structFieldValue.IsValid() {
		return fmt.Errorf("No such field: %s in obj", name)
	}

	if !structFieldValue.CanSet() {
		return fmt.Errorf("Cannot set %s field value", name)
	}
	// 结构体的类型
	structFieldType := structFieldValue.Type()
	// map值的反射值
	val := reflect.ValueOf(value)

	var err error
	if structFieldType != val.Type() {
		// 类型转换
		val, err = TypeConversion(fmt.Sprintf("%v", value), structFieldValue.Type().Name())
		if err != nil {
			Log.Error("类型转换异常:\nfield name: ", name, "\nvalue type: ", structFieldValue.Type().Name(), "\nvalue: ", fmt.Sprintf("%v", value), "\nerror: ", err)
			return err
		}
	}

	structFieldValue.Set(val)
	return nil
}

/**
 * 示例:
 * type MyStruct struct { name string `field:"name"` }
 * s := new(MyStruct)
 * res, err := ArrayMap2Struct(dMap, s)
 * if err != nil { ... }
 * for _, v := range result {
 * 	val, ok := v.(*MyStruct)
 *	if ok {
 *		Log.Error("value: ", val.name)
 *	}
 *	Log.Error(v)
 * }
 */
func ArrayMap2Struct(data []map[string]string, s interface{}) (result []interface{}, err error) {
	// 结构体属性
	st := reflect.TypeOf(s).Elem()
	for i := 0; i < len(data); i++ {
		// 创建新结构体对象
		sn := reflect.New(st).Interface()
		c := data[i]
		err = FillStruct(c, sn)
		if err != nil {
			Log.Error(err)
			// 数据处理异常,进行下一个记录处理
			continue
		}
		result = append(result, sn)
	}

	return result, nil
}

func Map2Struct(data map[string]string, s interface{}) (err error) {
	if len(data) == 0 {
		err = errors.New("map 结果集为空")
		return err
	}

	err = FillStruct(data, s)
	if err != nil {
		Log.Error(err)
	}

	return err
}

// 类型转换
func TypeConversion(value string, ntype string) (reflect.Value, error) {
	var res reflect.Value
	var err error
	var t time.Time
	var i int
	var i64 int64
	var f64 float64
	switch ntype {
	case "string":
		res = reflect.ValueOf(value)
	case "time.Time":
		t, err = time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
		res = reflect.ValueOf(t)
	case "Time":
		t, err = time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
		res = reflect.ValueOf(t)
	case "int":
		i, err = strconv.Atoi(value)
		res = reflect.ValueOf(i)
	case "int8":
		i64, err = strconv.ParseInt(value, 10, 64)
		res = reflect.ValueOf(int8(i64))
	case "int32":
		i64, err = strconv.ParseInt(value, 10, 64)
		res = reflect.ValueOf(int32(i64))
	case "int64":
		i64, err = strconv.ParseInt(value, 10, 64)
		res = reflect.ValueOf(int64(i64))
	case "float32":
		f64, err = strconv.ParseFloat(value, 64)
		res = reflect.ValueOf(float32(f64))
	case "float64":
		f64, err = strconv.ParseFloat(value, 64)
		res = reflect.ValueOf(float64(f64))
		// 根据实际需要补充更多类型的处理
	default:
		err = errors.New("未知的类型：" + ntype)
	}

	if err != nil {
		res = reflect.ValueOf(value)
	}

	return res, err
}
