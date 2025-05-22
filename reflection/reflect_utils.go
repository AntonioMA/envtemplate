package reflection

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// GetAsFunction finds the method called methodName on the object represented by obj, and if possible
// returns on fnPtr a newly generated function that will invoke the method using obj as the receiver.
// The argument fnPtr must be a pointer to a function (possibly nil since it'll be overwritten)
// that will receive the resulting function. Also note that, a pointer to a function is actually a
// pointer to a pointer to a function, which is why this works. It returns nil if the function could
// be generated, or an error describing why not otherwise
func GetAsFunction(obj interface{}, methodName string, fnPtr interface{}) error {
	methodName = strings.Title(methodName) //nolint:staticcheck

	fn := reflect.ValueOf(fnPtr).Elem()
	fnType := fn.Type()
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("invalid output parameter: %+v", fnType.Kind())
	}

	// Check that the method has the same signature as the function that will be generated, as far as parameters go
	args := make([]interface{}, fnType.NumIn())
	for i := range args {
		args[i] = fnType.In(i)
	}

	method, err := CheckValidMethod(obj, methodName, args...)

	if err != nil {
		return fmt.Errorf("invalid method: %s: %+v", methodName, err)
	}

	// Something CheckValidMethod does *NOT* check is the return value. But this will panic if the return type
	// is not correct. So lets check it
	methodType := method.Type()
	outArgNum := methodType.NumOut()
	if fnType.NumOut() != methodType.NumOut() {
		return fmt.Errorf("incorrect number of output parameters. Expected: %d. Actual: %d", fnType.NumOut(), outArgNum)
	}

	for i := 0; i < outArgNum; i++ {

		if fnOut, methodOut := fnType.Out(i), methodType.Out(i); fnOut != methodOut {
			return fmt.Errorf("incorrect output type for parameter %d. Expected: %+v. Actual: %+v", i, fnOut, methodOut)
		}
	}

	// method.Call has the right signature for MakeFunc. So just get the type for the returned function...
	v := reflect.MakeFunc(fnType, method.Call)

	// And return it on the right place
	fn.Set(v)
	return nil
}

// CheckValidMethod returns a Value representing the method called methodName on the obj received
// as the first parameter, if the method exists, can be accessed, and it receives the same number
// and type of the arguments passed in args. So for example, calling
// CheckValidMethod(SomeObject{}, "aMethod", 1, 2, "hi") will return a value representing
// anObject.AMethod (note the caps on the method name!) if func(s SomeObject)AMethod(int, int, string) or
// func(s *SomeObject)AMethod(int, int, string) exist. It will return nil as the value, and an error
// if any of those conditions are not true.
func CheckValidMethod(obj interface{}, methodName string, args ...interface{}) (reflect.Value, error) {
	methodName = strings.Title(methodName) //nolint:staticcheck
	var objPtr reflect.Value

	objValue := reflect.ValueOf(obj)
	// We want to check for the method if the receiver is an object or a pointer to an object...
	// Note that we could force the caller to pass a pointer... but if the caller has an interface
	// that's a royal pain in the ass. So we just do it ourselves
	if objValue.Type().Kind() == reflect.Ptr {
		objPtr = objValue
		objValue = objPtr.Elem()
	} else {
		objPtr = reflect.New(reflect.TypeOf(obj))
		temp := objPtr.Elem()
		temp.Set(objValue)
	}

	method := objValue.MethodByName(methodName)
	if !method.IsValid() {
		method = objPtr.MethodByName(methodName)
		if !method.IsValid() {
			return method, fmt.Errorf("invalid method: %s", methodName)
		}
	}

	methodType := method.Type()
	if methodType.Kind() != reflect.Func { // Can this even happen?
		return method, fmt.Errorf("not a function: %+v", methodType.Kind())
	}

	numIn := methodType.NumIn()

	// logger.Debug("InputParameters", O2s(numIn), "Output", O2s(methodType.NumOut()))

	if numIn != len(args) {
		return method, fmt.Errorf("incorrect argument number. Expected: %d, actual: %d", len(args), numIn)
	}

	for i := range args {
		var argType reflect.Type
		if asType, isType := args[i].(reflect.Type); isType {
			argType = asType
		} else {
			argType = reflect.TypeOf(args[i])
		}
		if argType != methodType.In(i) { // Todo: Subtypes? Subinterfaces?
			return method, fmt.Errorf("invalid argument type. Expected: %+v. Actual: %+v", methodType.In(i), argType)
		}
	}
	return method, nil
}

// Invoke executes the method called methodName on the receiver obj, passing it the values received
// in args, if CheckValidMethod(obj, methodName, args) doesn't return an error, It will return an
// array of values with the result of the function invocation.
func Invoke(obj interface{}, methodName string, args ...interface{}) ([]reflect.Value, error) {
	methodName = strings.Title(methodName) //nolint:staticcheck
	method, validError := CheckValidMethod(obj, methodName, args...)
	if validError != nil {
		return nil, validError
	}

	methodType := method.Type()

	argsAsValues := make([]reflect.Value, methodType.NumIn())
	for i := range args {
		// logger.Debug("ArgNumber", O2s(i), "ArgType", O2s(methodType.In(i)))
		argsAsValues[i] = reflect.ValueOf(args[i]) // Note: those are the actual arguments
	}

	rv := method.Call(argsAsValues)

	return rv, nil

}

// GetCallerName returns the name of a function that is on the stack when this function is called.
// The argument skip is the number of stack frames to ascend, with 0 identifying the caller of
// GetCaller. It will return an empty string "" if it wasn't possible to find the correct name
func GetCallerName(skip int) string {
	if pc, _, _, ok := runtime.Caller(skip + 1); ok {
		return runtime.FuncForPC(pc).Name()
	} else {
		return ""
	}
}

// GetCallerInfo Returns the info (name, and file:lineNumber) of a function that is on the stack
// when this function is called.The argument skip is the number of stack frames to ascend, with 0
// identifying the caller of GetCallerInfo. It will return an empty string "" if it wasn't possible
// to find the correct name
func GetCallerInfo(skip int) (name string, file string) {
	if pc, file, line, ok := runtime.Caller(skip + 1); ok {
		fn := runtime.FuncForPC(pc)
		return fn.Name(), filepath.Base(file) + ":" + strconv.Itoa(line)
	} else {
		return "", ""
	}
}

// GetTypeAndValue gets the type and underlying reflect.Value of the object passed if it was not a
// pointer, or the type (and value) of the pointed object if it's a pointer. It does not dereference
// nil values, for obvious reasons
func GetTypeAndValue(object interface{}) (t reflect.Type, value reflect.Value) {
	value = reflect.ValueOf(object)
	if t = value.Type(); t.Kind() == reflect.Ptr && !value.IsNil() {
		value = value.Elem()
		t = value.Type()
	}
	return
}

// GetTypeName returns the type of the object received (the underlying type, so it'll return the
// same for struct{} and &struct{}
func GetTypeName(any interface{}) string {
	t, _ := GetTypeAndValue(any)
	return t.Name()
}

// CheckValidKind checks if the passed object (interface) is/fulfills/whatever the kind passed on expectedKind
// You can use this to, for example, check if an object is an struct, or a pointer to an struct
// or whatever.
func CheckValidKind(obj interface{}, expectedKind reflect.Kind, expectedPointer bool) (err error) {
	err = nil
	objElem := reflect.ValueOf(obj)
	objKind := objElem.Type().Kind()
	if expectedPointer {
		if objElem.Type().Kind() != reflect.Ptr {
			err = fmt.Errorf("the object passed is not a pointer, %+v", objElem.Type().Kind())
			return
		}
		objElem = objElem.Elem()
		objKind = objElem.Type().Kind()
	}

	if objKind != expectedKind {
		err = fmt.Errorf("unexpected type. Expected %+v, Got: %+v", expectedKind, objKind)
	}
	return
}

// GetTagMap returns a map of the tags in an object. The index of the map is the name of the field, and
// the value is the StructTag for that particular field. Obj must be a struct or a pointer to a struct
// (an empty map is returned otherwise).
func GetTagMap(obj interface{}) (rv map[string]reflect.StructTag) {
	rv = make(map[string]reflect.StructTag)
	objType, _ := GetTypeAndValue(obj)
	if objType.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		rv[field.Name] = field.Tag
	}
	return
}

func getField(obj interface{}, fieldName string) (reflect.Value, error) {
	objType, objValue := GetTypeAndValue(obj)

	if objValue.Kind() != reflect.Struct {
		return reflect.Zero(objType), fmt.Errorf("first argument is not an struct")
	}

	if _, exists := objType.FieldByName(fieldName); !exists {
		return reflect.Zero(objType), fmt.Errorf("field does not exist")
	}

	return objValue.FieldByName(fieldName), nil

}

// GetFieldPointer returns a pointer to the field named fieldName on the struct obj. Obj must be
// a pointer to an struct (it will return nil otherwise). The pointer is returned as an interface
// for what should be obvious reasons. It's up to the caller to convert that to the right kind of
// pointer before using it (or not...)
func GetFieldPointer(obj interface{}, fieldName string) (interface{}, error) {
	// This makes sense if you think about it a lot... you cannot get the address of an struct that's
	// passed by value because structs are value types, not reference types. So it'll be on the stack
	// and trying to get the address of its fields will end in disaster. Or just not work
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return nil, fmt.Errorf("need a pointer to an struct as input object")
	}

	fieldValue, err := getField(obj, fieldName)
	if err != nil {
		return nil, err
	}
	if !fieldValue.CanAddr() {
		return nil, fmt.Errorf("fieldValue %s is not addressable", fieldName)
	}
	return fieldValue.Addr().Interface(), nil

}

// GetFieldAsInterface returns the field named fieldName as an interface (which you can cast to the
// right type assuming you know it). It has the same signature as GetFieldPointer, but while the
// value returned by GetFieldPointer is actually a pointer to the value (and this it requires the
// input object to be a pointer itself, this function returns the actual value
func GetFieldAsInterface(obj interface{}, fieldName string) (interface{}, error) {
	fieldValue, err := getField(obj, fieldName)
	if err != nil {
		return nil, err
	}
	return fieldValue.Interface(), nil
}

// GetFieldsWithTag Returns two arrays:
//   - The field names of the fields that are tagged with the requested tag
//   - The value that said tag has for each field.
//     It'll return teo empty arrays if
//   - obj is not an struct
//   - there are no fields with the requested tag.
//     For example, if obj is
//     struct {
//     A string `mytag:"valueA"`
//     B string
//     C string `mytag:"valueC" otherTag:"whatever"`
//     }
//     then calling
//     GetFieldsWithTag(objm, "mytag")
//     will return
//     []string{"A", "C"}, []string{"valueA", "valueB"}
func GetFieldsWithTag(obj interface{}, tagName string) ([]string, []string) {
	objType, _ := GetTypeAndValue(obj)
	if objType.Kind() != reflect.Struct {
		return []string{}, []string{}
	}
	numFields := objType.NumField()
	fields := make([]string, 0, numFields)
	tagValues := make([]string, 0, numFields)
	for i := 0; i < numFields; i++ {
		field := objType.Field(i)
		if tagValue, exists := field.Tag.Lookup(tagName); exists {
			fields = append(fields, field.Name)
			tagValues = append(tagValues, tagValue)
		}
	}
	return fields, tagValues
}

// GetFieldsOfKind returns a map with the public fields of the given kind that obj has. The key of
// the map will be the field name, while the value of the map will be the actual value of the field
// names. Returns an error if obj passed is not a struct or pointer to struct
func GetFieldsOfKind(kind reflect.Kind, obj interface{}) (map[string]interface{}, error) {
	asType, asValue := GetTypeAndValue(obj)
	if asType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid type, should be struct or pointer to struct, is: %+v", asType.Kind())
	}

	numFields := asType.NumField()
	rv := make(map[string]interface{}, numFields)
	for i := 0; i < numFields; i++ {
		field := asType.Field(i)
		if name := field.Name; name[0] >= 'A' && name[0] <= 'Z' && field.Type.Kind() == kind {
			rv[name] = asValue.FieldByName(name).Interface()
		}
	}
	return rv, nil
}

// GetFieldsNames returns a list with the field names of the passed in object (obj). If
// onlyPublic is true, it will return only the public (accessible from out of the package)
// names. Returns an error if obj passed is not a struct or pointer to struct
func GetFieldsNames(obj interface{}, onlyPublic bool) ([]string, error) {
	asType, _ := GetTypeAndValue(obj)

	if asType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid type, should be struct or pointer to struct, is: %+v", asType.Kind())
	}

	var rv []string
	numFields := asType.NumField()
	for i := 0; i < numFields; i++ {
		if name := asType.Field(i).Name; !onlyPublic || (name[0] >= 'A' && name[0] <= 'Z') {
			rv = append(rv, name)
		}
	}
	return rv, nil
}

// ConditionalCopy copies all the public (accessible from out of the package) values from the src
// struct to the dst struct. dst and src do not need to have the same type. Only the fields that
// have the same name and type on src and dst are copied. It will return an error if public fields
// of src cannot be got (probably cause it's not an struct or pointer to struct) or if dst is not a
// pointer to an struct.
// Only the fields for which checkerFn returns true will be copied
func ConditionalCopy(dst, src interface{}, checkerFn func(field string, dst, src interface{}) bool) error {
	publicFields, err := GetFieldsNames(src, true)
	if err != nil {
		return fmt.Errorf("invalid origin: %+v", err)
	}
	if err := CheckValidKind(dst, reflect.Struct, true); err != nil {
		return err
	}
	for _, field := range publicFields {
		dstFieldPointer, err := GetFieldPointer(dst, field)
		if err != nil {
			continue
		}
		dstField, err := GetFieldAsInterface(dst, field)
		if err != nil {
			continue
		}
		srcField, err := GetFieldAsInterface(src, field)
		if err != nil {
			continue
		}

		dstType, dstValue := GetTypeAndValue(dstFieldPointer)
		srcType, srcValue := GetTypeAndValue(srcField)
		if dstType != srcType {
			continue
		}

		if checkerFn(field, dstField, srcField) {
			dstValue.Set(srcValue)
		}

	}
	return nil
}

// GetNewElementForSlice returns a pointer to an empty element of the type hold on the out slice. Out must be a slice or a
// pointer to a Slice. If out is NOT a slice or a pointer to a slice, it returns nil
func GetNewElementForSlice(out interface{}) interface{} {
	var slice interface{}
	if CheckValidKind(out, reflect.Slice, true) == nil {
		slice = reflect.ValueOf(out).Elem().Interface()
	} else if CheckValidKind(out, reflect.Slice, false) == nil {
		slice = out
	} else {
		return nil
	}
	return reflect.New(reflect.ValueOf(slice).Type().Elem()).Interface()
}

// AddElementToSlice adds a new element to a slice. The element can be either the element or a pointer to an actual
// element. The shouldDereference parameter is used to desambiguate both usages. Out can be a slice or a pointer to
// a slice. On both cases a new value will be returned
func AddElementToSlice(out interface{}, elem interface{}, shouldDereference bool) interface{} {
	slice := reflect.ValueOf(out)
	if CheckValidKind(out, reflect.Slice, true) == nil {
		slice = slice.Elem()
	} else if CheckValidKind(out, reflect.Slice, false) != nil {
		return nil
	}

	var actualElem = reflect.ValueOf(elem)
	if shouldDereference {
		actualElem = actualElem.Elem()
	}
	return reflect.Append(slice, actualElem).Interface()
}

// AddElementsToSlice adds a set (slice) of elements to an existing slice. It's the generic version of append(slice, elems...)
// out and elems must both be slices, and must have the same type. This is NOT checked, and if that's not true then this will
// panic in run time
// Axiom: elems is an slice of something, and out is a slice of the same something
func AddElementsToSlice(out interface{}, elems interface{}) interface{} {
	elemsValue := reflect.ValueOf(elems)
	l := reflect.ValueOf(elems).Len()
	for i := 0; i < l; i++ {
		out = AddElementToSlice(out, elemsValue.Index(i).Interface(), false)
	}
	return out
}

// StarSet implements the equivalent to *out = v. It will panic if:
//   - out is not a pointer
//   - *out and v are not of assignable types
func StarSet(out, v interface{}) {
	reflect.ValueOf(out).Elem().Set(reflect.ValueOf(v))
}

// GetMapOf returns a map[key type]value type (created, non nil)
func GetMapOf(key, value interface{}) interface{} {
	return reflect.MakeMap(reflect.MapOf(reflect.TypeOf(key), reflect.TypeOf(value))).Interface()
}

// GetSliceOfType returns a []elems of Type,
func GetSliceOfType(t reflect.Type, len, cap int) interface{} {
	return reflect.MakeSlice(reflect.SliceOf(t), len, cap).Interface()
}

// GetSliceOf returns a []type of elem,
func GetSliceOf(elem interface{}, len, cap int) interface{} {
	return reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(elem)), len, cap).Interface()
}

// SliceLen returns the length of in, assuming that in is a slice or string
// or a pointer to a slice or string. Returns 0 if in is not a slice or a
// string (or if it is a 0 len slice)
func SliceLen(in interface{}) int {
	if in == nil {
		return 0
	}
	asValue := reflect.ValueOf(in)
	if asValue.Kind() == reflect.Ptr {
		asValue = asValue.Elem()
	}
	if asValue.Kind() != reflect.Slice && asValue.Kind() != reflect.String {
		return 0
	}
	return asValue.Len()
}

// GetMapElem returns the value of obj[key] as an interface, if obj is a map
// directly or indirectly (that is, it'll also work if obj is some kind of
// renaming of map). Returns an error != nil if obj cannot be mapped to a map.
// It will panic if key is not of an assignable type to the key tupe on the obj
// underlying map
func GetMapElem(obj interface{}, key interface{}) (interface{}, error) {
	if kind := reflect.TypeOf(obj).Kind(); kind != reflect.Map {
		return nil, fmt.Errorf("obj is not a map (%v)", kind)
	}
	k := reflect.ValueOf(key)
	m := reflect.ValueOf(obj)
	v := m.MapIndex(k)

	if v.IsValid() && v.CanInterface() {
		return v.Interface(), nil
	} else {
		return nil, nil
	}
}

// SetMapElem sets obj[key] = value, assuming that obj is a map[typeOf(key)]typeOf(value). It returns
// an error if any of the assumptions is false.
func SetMapElem(obj, key, value interface{}) error {
	if kind := reflect.TypeOf(obj).Kind(); kind != reflect.Map {
		return fmt.Errorf("obj is not a map (%v)", kind)
	}
	m := reflect.ValueOf(obj)
	m.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	return nil
}

func canBeNil(kind reflect.Kind) bool {
	return kind == reflect.Ptr || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array
}

// StructToMap returns a map that holds the public attributes of obj. The keys will be either
// the name of the field, or the json name annotation if the field has a json annotation. So for
// example
//
//	struct {
//	  A int `json:"fieldA"`
//	  B string
//	}{A: 1, B: "hi"}
//
// will be converted to
//
//	map[string]interface{}{ "fieldA": 1, "B": "hi" }
//
// If obj is a map[string]interface{} then the returned value will be obj (not a copy!). If obj is
// not an struct, a pointer to struct or a map[string]interface{} then nil will be returned
func StructToMap(obj interface{}) map[string]interface{} {
	return StructToMapUsingTag(obj, "json")
}

// StructToMapUsingTag returns a map that holds the public attributes of obj. The keys will be either
// the name of the field, or the  name annotation for the passed in tag if the field has an annotation
// with that tag. So for example, if tag is "whatever" and obj is
//
//	struct {
//	  A int `whatever:"fieldA"`
//	  B string
//	}{A: 1, B: "hi"}
//
// will be converted to
//
//	map[string]interface{}{ "fieldA": 1, "B": "hi" }
//
// If obj is a map[string]interface{} then the returned value will be obj (not a copy!). If obj is
// not an struct, a pointer to struct or a map[string]interface{} then nil will be returned
func StructToMapUsingTag(obj interface{}, tag string) map[string]interface{} {
	if obj == nil {
		return nil
	}
	if asMap, isMap := obj.(map[string]interface{}); isMap {
		return asMap
	}

	t, v := GetTypeAndValue(obj)
	if t.Kind() != reflect.Struct {
		return nil
	}

	numFields := t.NumField()
	rv := make(map[string]interface{}, numFields)
	for i := 0; i < numFields; i++ {
		field := t.Field(i)
		name := field.Name
		if name[0] < 'A' || name[0] > 'Z' {
			continue
		}
		jsonTag := field.Tag.Get(tag)
		if jsonTag != "" {
			name = strings.Split(jsonTag, ";")[0]
		}
		fieldValue := v.Field(i)
		if fk := fieldValue.Type().Kind(); (canBeNil(fk) && fieldValue.IsNil()) || !fieldValue.CanInterface() {
			rv[name] = nil
		} else {
			rv[name] = v.Field(i).Interface()
		}
	}
	return rv
}

// PtrToElem returns a (possibly nil) pointer to the base type of elem, and the base type of elem. That is,
// if elem is:
//
//	 A => returns *A, type(A)
//	*A => return *A, type(A)
//
// so basically it's just a nifty way of getting always a pointer, and it's type no matter what elem
// is. Note that this will panic if elem is nil!
func PtrToElem(elem interface{}) (ptrToBase interface{}, typeOfBase reflect.Type) {
	// We want to create a map[string]*Whatever always...
	if t := reflect.TypeOf(elem); t.Kind() != reflect.Ptr {
		return reflect.New(t).Interface(), reflect.TypeOf(elem)
	}

	return elem, reflect.ValueOf(elem).Elem().Type()
}

// GetBaseElem returns the "base" element of value, where "base" is defined as what's left once all
// the pointers have been referenced. So if value is &(&(&X)), then GetBaseElem(value) will be X (if
// X it's not a pointer) or *X if X is a pointer.
func GetBaseElem(value interface{}) interface{} {
	for ; reflect.ValueOf(value).Kind() == reflect.Ptr; value = reflect.ValueOf(value).Elem().Interface() {
	}
	return value
}

var alwaysTrue = func(_ string, _, _ interface{}) bool {
	return true
}

// CopyElement returns a new element that has *X as type, if e is any kind of direct or indirect reference to X (that is, if e is
// X, *X, **X and so on). The returned element will have a shallow copy of all the public fields of e.
func CopyElement(e interface{}) interface{} {
	newValue := reflect.New(reflect.ValueOf(GetBaseElem(e)).Type()).Interface()
	if err := ConditionalCopy(newValue, GetBaseElem(e), alwaysTrue); err != nil {
		return err
	}
	return newValue
}
