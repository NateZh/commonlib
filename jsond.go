/*
json decoder
 */

package commonlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Result map[string]interface{}

var typeOfJSONNumber = reflect.TypeOf(json.Number(""))

// Get gets a field from Result.
//
// Field can be a dot separated string.
// If field name is "a.b.c", it will try to return value of res["a"]["b"]["c"].
//
// To access array items, use index value in field.
// For instance, field "a.0.c" means to read res["a"][0]["c"].
//
// It doesn't work with Result which has a key contains dot. Use GetField in this case.
//
// Returns nil if field doesn't exist.
func (res Result) Get(field string) interface{} {
	if field == "" {
		return res
	}

	f := strings.Split(field, ".")
	return res.get(f)
}

// GetField gets a field from Result.
//
// Arguments are treated as keys to access value in Result.
// If arguments are "a","b","c", it will try to return value of res["a"]["b"]["c"].
//
// To access array items, use index value as a string.
// For instance, args of "a", "0", "c" means to read res["a"][0]["c"].
//
// Returns nil if field doesn't exist.
func (res Result) GetField(fields ...string) interface{} {
	if len(fields) == 0 {
		return res
	}

	return res.get(fields)
}

func (res Result) get(fields []string) interface{} {
	v, ok := res[fields[0]]

	if !ok || v == nil {
		return nil
	}

	if len(fields) == 1 {
		return v
	}

	value := getValueField(reflect.ValueOf(v), fields[1:])

	if !value.IsValid() {
		return nil
	}

	return value.Interface()
}

func getValueField(value reflect.Value, fields []string) reflect.Value {
	valueType := value.Type()
	kind := valueType.Kind()
	field := fields[0]

	switch kind {
	case reflect.Array, reflect.Slice:
		// field must be a number.
		n, err := strconv.ParseUint(field, 10, 0)

		if err != nil {
			return reflect.Value{}
		}

		if n >= uint64(value.Len()) {
			return reflect.Value{}
		}

		// work around a reflect package pitfall.
		value = reflect.ValueOf(value.Index(int(n)).Interface())

	case reflect.Map:
		v := value.MapIndex(reflect.ValueOf(field))

		if !v.IsValid() {
			return v
		}

		// get real value type.
		value = reflect.ValueOf(v.Interface())

	default:
		return reflect.Value{}
	}

	if len(fields) == 1 {
		return value
	}

	return getValueField(value, fields[1:])
}

// Decode decodes full result to a struct.
// It only decodes fields defined in the struct.
//
// As all facebook response fields are lower case strings,
// Decode will convert all camel-case field names to lower case string.
// e.g. field name "FooBar" will be converted to "foo_bar".
// The side effect is that if a struct has 2 fields with only capital
// differences, decoder will map these fields to a same result value.
//
// If a field is missing in the result, Decode keeps it unchanged by default.
//
// Decode can read struct field tag value to change default behavior.
//
// Examples:
//
//     type Foo struct {
//         // "id" must exist in response. note the leading comma.
//         Id string `facebook:",required"`
//
//         // use "name" as field name in response.
//         TheName string `facebook:"name"`
//     }
//
// To change default behavior, set a struct tag `facebook:",required"` to fields
// should not be missing.
//
// Returns error if v is not a struct or any required v field name absents in res.
func (res Result) Decode(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}

			err = r.(error)
		}
	}()

	err = res.decode(reflect.ValueOf(v), "")
	return
}

// DecodeField decodes a field of result to any type, including struct.
// Field name format is defined in Result.Get().
//
// More details about decoding struct see Result.Decode().
func (res Result) DecodeField(field string, v interface{}) error {
	f := res.Get(field)

	if f == nil {
		return fmt.Errorf("field '%v' doesn't exist in result.", field)
	}

	return decodeField(reflect.ValueOf(f), reflect.ValueOf(v), field)
}

func (res Result) decode(v reflect.Value, fullName string) error {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("output value must be a struct.")
	}

	if !v.CanSet() {
		return fmt.Errorf("output value cannot be set.")
	}

	if fullName != "" {
		fullName += "."
	}

	var field reflect.Value
	var name, fbTag string
	var val interface{}
	var ok, required bool
	var err error

	vType := v.Type()
	num := vType.NumField()

	for i := 0; i < num; i++ {
		name = ""
		required = false
		field = v.Field(i)
		fbTag = vType.Field(i).Tag.Get("facebook")

		// parse struct field tag
		if fbTag != "" {
			index := strings.IndexRune(fbTag, ',')

			if index == -1 {
				name = fbTag
			} else {
				name = fbTag[:index]

				if fbTag[index:] == ",required" {
					required = true
				}
			}
		}

		if name == "" {
			name = camelCaseToUnderScore(v.Type().Field(i).Name)
		}

		val, ok = res[name]

		if !ok {
			// check whether the field is required. if so, report error.
			if required {
				return fmt.Errorf("cannot find field '%v%v' in result.", fullName, name)
			}

			continue
		}

		if err = decodeField(reflect.ValueOf(val), field, fmt.Sprintf("%v%v", fullName, name)); err != nil {
			return err
		}
	}

	return nil
}

func decodeField(val reflect.Value, field reflect.Value, fullName string) error {
	if field.Kind() == reflect.Ptr {
		// reset Ptr field if val is nil.
		if !val.IsValid() {
			if !field.IsNil() && field.CanSet() {
				field.Set(reflect.Zero(field.Type()))
			}

			return nil
		}

		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}

		field = field.Elem()
	}

	if !field.CanSet() {
		return fmt.Errorf("field '%v' cannot be decoded. make sure the output value is able to be set.", fullName)
	}

	if !val.IsValid() {
		return fmt.Errorf("field '%v' is not a pointer. cannot assign nil to it.", fullName)
	}

	// if field implements Unmarshaler, let field unmarshals data itself.
	if unmarshaler := indirect(field); unmarshaler != nil {
		data, err := json.Marshal(val.Interface())

		if err != nil {
			return fmt.Errorf("fail to marshal value for field '%v'. %v", fullName, err)
		}

		return unmarshaler.UnmarshalJSON(data)
	}

	kind := field.Kind()
	valType := val.Type()

	switch kind {
	case reflect.Bool:
		if valType.Kind() == reflect.Bool {
			field.SetBool(val.Bool())
		} else {
			return fmt.Errorf("field '%v' is not a bool in result.", fullName)
		}

	case reflect.Int8:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < -128 || n > 127 {
				return fmt.Errorf("field '%v' value exceeds the range of int8.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > 127 {
				return fmt.Errorf("field '%v' value exceeds the range of int8.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < -128 || n > 127 {
				return fmt.Errorf("field '%v' value exceeds the range of int8.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseInt(val.String(), 10, 8)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid int8.", fullName)
			}

			field.SetInt(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Int16:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < -32768 || n > 32767 {
				return fmt.Errorf("field '%v' value exceeds the range of int16.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > 32767 {
				return fmt.Errorf("field '%v' value exceeds the range of int16.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < -32768 || n > 32767 {
				return fmt.Errorf("field '%v' value exceeds the range of int16.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseInt(val.String(), 10, 16)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid int16.", fullName)
			}

			field.SetInt(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Int32:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < -2147483648 || n > 2147483647 {
				return fmt.Errorf("field '%v' value exceeds the range of int32.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > 2147483647 {
				return fmt.Errorf("field '%v' value exceeds the range of int32.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < -2147483648 || n > 2147483647 {
				return fmt.Errorf("field '%v' value exceeds the range of int32.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseInt(val.String(), 10, 32)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid int32.", fullName)
			}

			field.SetInt(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Int64:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()
			field.SetInt(n)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > 9223372036854775807 {
				return fmt.Errorf("field '%v' value exceeds the range of int64.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < -9223372036854775808 || n > 9223372036854775807 {
				return fmt.Errorf("field '%v' value exceeds the range of int64.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseInt(val.String(), 10, 64)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid int64.", fullName)
			}

			field.SetInt(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Int:
		bits := field.Type().Bits()

		var min, max int64

		if bits == 32 {
			min = -2147483648
			max = 2147483647
		} else if bits == 64 {
			min = -9223372036854775808
			max = 9223372036854775807
		}

		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < min || n > max {
				return fmt.Errorf("field '%v' value exceeds the range of int.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > uint64(max) {
				return fmt.Errorf("field '%v' value exceeds the range of int.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < float64(min) || n > float64(max) {
				return fmt.Errorf("field '%v' value exceeds the range of int.", fullName)
			}

			field.SetInt(int64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseInt(val.String(), 10, bits)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid int%v.", fullName, bits)
			}

			field.SetInt(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Uint8:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < 0 || n > 0xFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint8.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > 0xFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint8.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < 0 || n > 0xFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint8.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseUint(val.String(), 10, 8)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid uint8.", fullName)
			}

			field.SetUint(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Uint16:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < 0 || n > 0xFFFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint16.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > 0xFFFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint16.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < 0 || n > 0xFFFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint16.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseUint(val.String(), 10, 16)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid uint16.", fullName)
			}

			field.SetUint(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Uint32:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < 0 || n > 0xFFFFFFFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint32.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > 0xFFFFFFFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint32.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < 0 || n > 0xFFFFFFFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint32.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseUint(val.String(), 10, 32)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid uint32.", fullName)
			}

			field.SetUint(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Uint64:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < 0 {
				return fmt.Errorf("field '%v' value exceeds the range of uint64.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()
			field.SetUint(n)

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < 0 || n > 0xFFFFFFFFFFFFFFFF {
				return fmt.Errorf("field '%v' value exceeds the range of uint64.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseUint(val.String(), 10, 64)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid uint64.", fullName)
			}

			field.SetUint(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Uint:
		bits := field.Type().Bits()

		var max uint64

		if bits == 32 {
			max = 0xFFFFFFFF
		} else if bits == 64 {
			max = 0xFFFFFFFFFFFFFFFF
		}

		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()

			if n < 0 || uint64(n) > max {
				return fmt.Errorf("field '%v' value exceeds the range of uint.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()

			if n > max {
				return fmt.Errorf("field '%v' value exceeds the range of uint.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()

			if n < 0 || n > float64(max) {
				return fmt.Errorf("field '%v' value exceeds the range of uint.", fullName)
			}

			field.SetUint(uint64(n))

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseUint(val.String(), 10, bits)

			if err != nil {
				return fmt.Errorf("field '%v' value is not a valid uint%v.", fullName, bits)
			}

			field.SetUint(n)

		default:
			return fmt.Errorf("field '%v' is not an integer in result.", fullName)
		}

	case reflect.Float32, reflect.Float64:
		switch valType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := val.Int()
			field.SetFloat(float64(n))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := val.Uint()
			field.SetFloat(float64(n))

		case reflect.Float32, reflect.Float64:
			n := val.Float()
			field.SetFloat(n)

		case reflect.String:
			// only json.Number is allowed to be used as number.
			if val.Type() != typeOfJSONNumber {
				return fmt.Errorf("field '%v' value is string, not a number.", fullName)
			}

			n, err := strconv.ParseFloat(val.String(), 64)

			if err != nil {
				return fmt.Errorf("field '%v' is not a valid float64.", fullName)
			}

			field.SetFloat(n)

		default:
			return fmt.Errorf("field '%v' is not a float in result.", fullName)
		}

	case reflect.String:
		if valType.Kind() != reflect.String {
			return fmt.Errorf("field '%v' is not a string in result.", fullName)
		}

		field.SetString(val.String())

	case reflect.Struct:
		if valType.Kind() != reflect.Map || valType.Key().Kind() != reflect.String {
			return fmt.Errorf("field '%v' is not a json object in result.", fullName)
		}

		// safe convert val to Result. type assertion doesn't work in this case.
		var r Result
		reflect.ValueOf(&r).Elem().Set(val)

		if err := r.decode(field, fullName); err != nil {
			return err
		}

	case reflect.Map:
		if valType.Kind() != reflect.Map || valType.Key().Kind() != reflect.String {
			return fmt.Errorf("field '%v' is not a json object in result.", fullName)
		}

		// map key must be string
		if field.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("field '%v' in struct is a map with non-string key type. it's not allowed.", fullName)
		}

		var needAddr bool
		valueType := field.Type().Elem()

		// shortcut for map[string]interface{}.
		if valueType.Kind() == reflect.Interface {
			field.Set(val)
			break
		}

		if field.IsNil() {
			field.Set(reflect.MakeMap(field.Type()))
		}

		if valueType.Kind() == reflect.Ptr {
			valueType = valueType.Elem()
			needAddr = true
		}

		for _, key := range val.MapKeys() {
			// val.MapIndex(key) returns a Value with wrong type.
			// use following trick to get correct Value.
			value := reflect.ValueOf(val.MapIndex(key).Interface())
			newValue := reflect.New(valueType)

			if err := decodeField(value, newValue, fmt.Sprintf("%v.%v", fullName, key)); err != nil {
				return err
			}

			if needAddr {
				field.SetMapIndex(key, newValue)
			} else {
				field.SetMapIndex(key, newValue.Elem())
			}
		}

	case reflect.Slice, reflect.Array:
		if valType.Kind() != reflect.Slice && valType.Kind() != reflect.Array {
			return fmt.Errorf("field '%v' is not a json array in result.", fullName)
		}

		valLen := val.Len()

		if kind == reflect.Array {
			if field.Len() < valLen {
				return fmt.Errorf("cannot copy all field '%v' values to struct. expected len is %v. actual len is %v.",
					fullName, field.Len(), valLen)
			}
		}

		var slc reflect.Value
		var needAddr bool

		valueType := field.Type().Elem()

		// shortcut for array of interface
		if valueType.Kind() == reflect.Interface {
			if kind == reflect.Array {
				for i := 0; i < valLen; i++ {
					field.Index(i).Set(val.Index(i))
				}
			} else { // kind is slice
				field.Set(val)
			}

			break
		}

		if kind == reflect.Array {
			slc = field.Slice(0, valLen)
		} else {
			// kind is slice
			slc = reflect.MakeSlice(field.Type(), valLen, valLen)
			field.Set(slc)
		}

		if valueType.Kind() == reflect.Ptr {
			needAddr = true
			valueType = valueType.Elem()
		}

		for i := 0; i < valLen; i++ {
			// val.Index(i) returns a Value with wrong type.
			// use following trick to get correct Value.
			valIndexValue := reflect.ValueOf(val.Index(i).Interface())
			newValue := reflect.New(valueType)

			if err := decodeField(valIndexValue, newValue, fmt.Sprintf("%v.%v", fullName, i)); err != nil {
				return err
			}

			if needAddr {
				slc.Index(i).Set(newValue)
			} else {
				slc.Index(i).Set(newValue.Elem())
			}
		}

	default:
		return fmt.Errorf("field '%v' in struct uses unsupported type '%v'.", fullName, kind)
	}

	return nil
}

// Indirect walks down v allocating pointers as needed until it gets to a non-pointer.
// If v implements json.Unmarshaler, indrect stops and returns it.
//
// This implementation is a modified version of http://golang.org/src/encoding/json/decode.go.
func indirect(v reflect.Value) json.Unmarshaler {
	// if v is a struct field and v's pointer may implement json.Unmarshaler,
	// try to discover this case.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	for {
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()

			if e.Kind() == reflect.Ptr && !e.IsNil() && e.Elem().Kind() == reflect.Ptr {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && v.CanSet() {
			break
		}

		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(json.Unmarshaler); ok {
				return u
			}
		}

		v = v.Elem()
	}

	if v.Type().NumMethod() > 0 {
		if u, ok := v.Interface().(json.Unmarshaler); ok {
			return u
		}
	}

	return nil
}

func camelCaseToUnderScore(str string) string {
	if len(str) == 0 {
		return ""
	}

	buf := &bytes.Buffer{}
	var prev, r0, r1 rune
	var size int

	r0 = '_'

	for len(str) > 0 {
		prev = r0
		r0, size = utf8.DecodeRuneInString(str)
		str = str[size:]

		switch {
		case r0 == utf8.RuneError:
			buf.WriteByte(byte(str[0]))

		case unicode.IsUpper(r0):
			if prev != '_' {
				buf.WriteRune('_')
			}

			buf.WriteRune(unicode.ToLower(r0))

			if len(str) == 0 {
				break
			}

			r0, size = utf8.DecodeRuneInString(str)
			str = str[size:]

			if !unicode.IsUpper(r0) {
				buf.WriteRune(r0)
				break
			}

			// find next non-upper-case character and insert `_` properly.
			// it's designed to convert `HTTPServer` to `http_server`.
			// if there are more than 2 adjacent upper case characters in a word,
			// treat them as an abbreviation plus a normal word.
			for len(str) > 0 {
				r1 = r0
				r0, size = utf8.DecodeRuneInString(str)
				str = str[size:]

				if r0 == utf8.RuneError {
					buf.WriteRune(unicode.ToLower(r1))
					buf.WriteByte(byte(str[0]))
					break
				}

				if !unicode.IsUpper(r0) {
					if r0 == '_' || r0 == ' ' || r0 == '-' {
						r0 = '_'

						buf.WriteRune(unicode.ToLower(r1))
					} else {
						buf.WriteRune('_')
						buf.WriteRune(unicode.ToLower(r1))
						buf.WriteRune(r0)
					}

					break
				}

				buf.WriteRune(unicode.ToLower(r1))
			}

			if len(str) == 0 || r0 == '_' {
				buf.WriteRune(unicode.ToLower(r0))
				break
			}

		default:
			if r0 == ' ' || r0 == '-' {
				r0 = '_'
			}

			buf.WriteRune(r0)
		}
	}

	return buf.String()
}
