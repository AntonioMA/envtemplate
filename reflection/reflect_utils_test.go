package reflection

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Object that we'll use to test the reflection
type TestObj struct {
	c int
}

func (s TestObj) TestMethodNP() int { return 1 }

func (s TestObj) TestMethod(a, b int) int { return a + b }

func (s *TestObj) TestMethodMod(a, b int) { s.c = a + b }

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Test information common for all tests here. What is common is searching for the method
type MethodSearchTestCase struct {
	TestObj       interface{}   // The object that we will reflect upon
	MethodName    string        // The name that we will find
	Args          []interface{} // Args for the method
	ExpectedValid bool          // Do we expect the search for the method to succeed?
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// And we'll use this interface to actually run the tests
type RUTester interface {
	Run(interface{}) (interface{}, error) // Run
	SpecificChecks(interface{}) error
	GetMethodSearchTestCase() MethodSearchTestCase
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func runTestCases(t *testing.T, tests map[string]interface{}) {

	for k, test := range tests {
		testCase := test.(RUTester)
		testObj := testCase.GetMethodSearchTestCase().TestObj
		testInfo := fmt.Sprintf("%s[%s]", t.Name(), k)
		m, err := testCase.Run(testObj)
		isValid := err == nil

		if expectedValid := testCase.GetMethodSearchTestCase().ExpectedValid; isValid != expectedValid {
			t.Errorf("%s Failure. Expected Error: %v. Actual Error: %+v", testInfo, !expectedValid, err)
			continue
		}

		if error := testCase.SpecificChecks(m); error != nil {
			t.Errorf("%s Failure: %+v", testInfo, error)
			continue
		}
		// t.Log(testInfo, "... ok")
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Basic Tests information
var methodSearchTestCases = map[string]MethodSearchTestCase{
	"Valid Call":           {TestObj{}, "TestMethod", []interface{}{1, 2}, true},
	"Valid Call Modifying": {TestObj{}, "TestMethodMod", []interface{}{1, 2}, true},
	"Invalid Args":         {TestObj{}, "TestMethod", []interface{}{1}, false},
	"Invalid Args2":        {TestObj{}, "TestMethodNP", []interface{}{1, 2, 3}, false},
	"Invalid Name":         {TestObj{}, "TestMethod1", []interface{}{1, 2}, false},
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Implements RUTester for CheckValidMethod
type checkValidMethodTestCase struct {
	MethodSearchTestCase
	ExpectedZero bool
}

func (tc checkValidMethodTestCase) GetMethodSearchTestCase() MethodSearchTestCase {
	return tc.MethodSearchTestCase
}

func (tc checkValidMethodTestCase) Run(testObj interface{}) (interface{}, error) {
	return CheckValidMethod(testObj, tc.MethodName, tc.Args...)
}

func (tc checkValidMethodTestCase) SpecificChecks(actual interface{}) error {
	if isZero := (reflect.Value{} == actual); tc.ExpectedZero != isZero {
		return fmt.Errorf("expected Zero: %v. Actual: %v", tc.ExpectedZero, isZero)
	}
	return nil
}

func TestCheckValidMethod(t *testing.T) {
	var checkValidMethodTestCases = map[string]interface{}{
		"Valid Call":           checkValidMethodTestCase{methodSearchTestCases["Valid Call"], false},
		"Valid Call Modifying": checkValidMethodTestCase{methodSearchTestCases["Valid Call Modifying"], false},
		"Invalid Args":         checkValidMethodTestCase{methodSearchTestCases["Invalid Args"], false},
		"Invalid Name":         checkValidMethodTestCase{methodSearchTestCases["Invalid Name"], true},
	}

	runTestCases(t, checkValidMethodTestCases)
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Implements RUTester for Invoke
type invokeTestCase struct {
	MethodSearchTestCase
	ExpectedResult []reflect.Value
}

func (tc invokeTestCase) GetMethodSearchTestCase() MethodSearchTestCase {
	return tc.MethodSearchTestCase
}

func (tc invokeTestCase) Run(testObj interface{}) (interface{}, error) {
	return Invoke(testObj, tc.MethodName, tc.Args...)
}

func (tc invokeTestCase) SpecificChecks(actual interface{}) error {
	result := actual.([]reflect.Value)
	expectedLen, actualLen := len(tc.ExpectedResult), len(result)
	if expectedLen != actualLen {
		return fmt.Errorf("wrong return values length. Expected: %v. Actual: %v", expectedLen, actualLen)
	}
	for i, v := range tc.ExpectedResult {
		if v.Kind() != result[i].Kind() {
			return fmt.Errorf("invalid result for %d. Expected: %v. Actual: %v", i, v.Kind(), result[i].Kind())
		}
		// Note that I'm assuming here that the returns are int. Cause it's not really worth it to do a generic comparison
		if v.Int() != result[i].Int() {
			return fmt.Errorf("invalid result for %d. Expected: %v. Actual: %v", i, v, result[i])
		}

	}

	return nil
}

var invokeTestCases = map[string]interface{}{
	"Valid Call":   invokeTestCase{methodSearchTestCases["Valid Call"], []reflect.Value{reflect.ValueOf(3)}},
	"Invalid Args": invokeTestCase{methodSearchTestCases["Invalid Args"], []reflect.Value{}},
	"Invalid Name": invokeTestCase{methodSearchTestCases["Invalid Name"], []reflect.Value{}},
}

func TestInvoke(t *testing.T) {
	runTestCases(t, invokeTestCases)
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Implements RUTester for GetAsFunction
type getAsFunctionTestCase struct {
	MethodSearchTestCase
	GeneratedFn     interface{}
	ExpectedValidFn bool
	TestFnValues    []int
}

type ValidFnType func(int, int) int
type InvalidFnArgs func(int) int
type InvalidFnResults func(int, int) string

func (tc getAsFunctionTestCase) GetMethodSearchTestCase() MethodSearchTestCase {
	return tc.MethodSearchTestCase
}

func (tc getAsFunctionTestCase) Run(testObj interface{}) (interface{}, error) {
	// This sucks beyond maximum suckitude. Starting to hate the typing in Go, with a passion.
	validFn, isValidFn := tc.GeneratedFn.(ValidFnType)
	if isValidFn {
		err := GetAsFunction(testObj, tc.MethodName, &validFn)
		return validFn, err
	}

	invalidFnArgs, isInvalidFnArgs := tc.GeneratedFn.(InvalidFnArgs)
	if isInvalidFnArgs {
		err := GetAsFunction(testObj, tc.MethodName, &invalidFnArgs)
		return invalidFnArgs, err
	}

	invalidFnResults, isInvalidFnResults := tc.GeneratedFn.(InvalidFnResults)
	if isInvalidFnResults {
		err := GetAsFunction(testObj, tc.MethodName, &invalidFnResults)
		return invalidFnResults, err
	}
	return nil, nil
}

func (tc getAsFunctionTestCase) SpecificChecks(actual interface{}) error {
	adder, isValid := actual.(ValidFnType)
	if !isValid && tc.ExpectedValidFn {
		return fmt.Errorf("the result value should be a valid function")
	}
	if !tc.ExpectedValidFn {
		return nil
	}

	if fnResult := adder(tc.TestFnValues[0], tc.TestFnValues[1]); fnResult != tc.TestFnValues[2] {
		return fmt.Errorf("invalid return from generated fn. Expected: %d. Actual: %d", tc.TestFnValues[2], fnResult)
	}
	return nil
}

func TestGetAsFunction(t *testing.T) {
	var validFn, validFn2 ValidFnType
	var invalidFnArgs InvalidFnArgs
	var invalidFnResults InvalidFnResults
	var getAsFunctionTestCases = map[string]interface{}{
		"Valid Call":        getAsFunctionTestCase{methodSearchTestCases["Valid Call"], validFn, true, []int{1, 2, 3}},
		"Invalid FnArgs":    getAsFunctionTestCase{methodSearchTestCases["Invalid Args"], invalidFnArgs, false, []int{}},
		"Invalid FnArgs2":   getAsFunctionTestCase{methodSearchTestCases["Invalid Args2"], invalidFnArgs, false, []int{}},
		"Invalid FnResults": getAsFunctionTestCase{methodSearchTestCases["Invalid Args"], invalidFnResults, false, []int{}},
		"Invalid Name":      getAsFunctionTestCase{methodSearchTestCases["Invalid Name"], validFn2, false, []int{}},
	}

	runTestCases(t, getAsFunctionTestCases)
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// 1
// 2 Do not erase this. Padding to keep the tests working
// 3
// 4
// 5
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func T2(depth, pos, desp int) (string, string) {
	if pos >= depth {
		return GetCallerInfo(depth + desp)
	}
	pos++
	return T1(depth, pos, desp)
}

func T1(depth, pos, desp int) (string, string) {
	if pos >= depth {
		return GetCallerInfo(depth + desp)
	}
	pos++
	return T2(depth, pos, desp)
}

func TestGetCallerInfo(t *testing.T) {
	expectedNames := []string{
		"T2", "T1", "TestGetCallerInfo",
	}
	expectedFiles := []string{
		"reflect_utils_test.go:243", "reflect_utils_test.go:251", "reflect_utils_test.go:263",
	}
	for i := 4; i < 14; i++ {
		desp := (i + 1) % 3
		name, file := T1(i, 0, desp-1)
		procName := strings.Split(name, ".")
		if simpleName := procName[len(procName)-1]; simpleName != expectedNames[desp] {
			t.Errorf("Invalid name, expected: %s, got: %s - %d %d", expectedNames[desp], simpleName, i, desp)
		}
		if file != expectedFiles[desp] {
			t.Errorf("Invalid file, expected: %s, got: %s", expectedFiles[desp], file)
		}
		// t.Log("Test Depth: ", i, "Desp:", desp-1, "... ok")
	}
}

// This is actually a subset of TestGetCallerInfo...
func T2CN(depth, pos, desp int) string {
	if pos >= depth {
		return GetCallerName(depth + desp)
	}
	pos++
	return T1CN(depth, pos, desp)
}

func T1CN(depth, pos, desp int) string {
	if pos >= depth {
		return GetCallerName(depth + desp)
	}
	pos++
	return T2CN(depth, pos, desp)
}

func TestGetCallerName(t *testing.T) {
	expectedNames := []string{
		"T2CN", "T1CN", "TestGetCallerName",
	}
	for i := 4; i < 14; i++ {
		desp := (i + 1) % 3
		name := T1CN(i, 0, desp-1)
		procName := strings.Split(name, ".")
		if simpleName := procName[len(procName)-1]; simpleName != expectedNames[desp] {
			t.Errorf("Invalid name, expected: %s, got: %s - %d %d", expectedNames[desp], simpleName, i, desp)
			continue
		}
		// t.Log("Test Depth: ", i, "Desp:", desp-1, "... ok")
	}
}

func TestGetTypeName(t *testing.T) {
	// Interface
	var x interface{} = TestObj{}
	tObj := TestObj{}
	aux := invokeTestCases["Valid Call"].(invokeTestCase)

	testCases := map[string]struct {
		inputObject  interface{}
		expectedType string
	}{
		"Direct Value": {
			inputObject:  TestObj{},
			expectedType: "TestObj",
		},
		"Direct Value Pointer": {
			inputObject:  &TestObj{},
			expectedType: "TestObj",
		},
		"Interface": {
			inputObject:  x,
			expectedType: "TestObj",
		},
		"Interface Pointer": {
			inputObject:  &x,
			expectedType: "",
		},
		"Intermediate Value": {
			inputObject:  tObj,
			expectedType: "TestObj",
		},
		"Intermediate Value Pointer": {
			inputObject:  &tObj,
			expectedType: "TestObj",
		},
		"Intermediate array ref": {
			inputObject:  invokeTestCases["Valid Call"],
			expectedType: "invokeTestCase",
		},
		"Intermediate pointer to array ref": {
			inputObject:  &aux,
			expectedType: "invokeTestCase",
		},
	}

	for testName, testCase := range testCases {
		if tn := GetTypeName(testCase.inputObject); tn != testCase.expectedType {
			t.Errorf("%s: Invalid Type. Expected: %s, got: %s", testName, testCase.expectedType, tn)
			continue
		}
		// t.Log(testName, "... ok")
	}
}

type checkValidKindTestCase struct {
	inputObject interface{}
	testForKind reflect.Kind
	isPointer   bool
	shouldFail  bool
}

func TestCheckValidKind(t *testing.T) {
	var testFn func()
	testStr := "Hello"
	testCases := map[string]checkValidKindTestCase{
		"Valid Struct": {
			inputObject: TestObj{},
			testForKind: reflect.Struct,
			isPointer:   false,
			shouldFail:  false,
		},
		"Valid PointerToStruct": {
			inputObject: &TestObj{},
			testForKind: reflect.Struct,
			isPointer:   true,
			shouldFail:  false,
		},
		"Valid Struct, check for pointer": {
			inputObject: TestObj{},
			testForKind: reflect.Struct,
			isPointer:   true,
			shouldFail:  true,
		},
		"Valid PointerToStruct, check for not pointer": {
			inputObject: &TestObj{},
			testForKind: reflect.Struct,
			isPointer:   false,
			shouldFail:  true,
		},
		"String": {
			inputObject: "Hello",
			testForKind: reflect.String,
			isPointer:   false,
			shouldFail:  false,
		},
		"Pointer to String": {
			inputObject: &testStr,
			testForKind: reflect.String,
			isPointer:   true,
			shouldFail:  false,
		},
		"Function": {
			inputObject: func() {},
			testForKind: reflect.Func,
			isPointer:   false,
			shouldFail:  false,
		},
		"Pointer to Function": {
			inputObject: &testFn,
			testForKind: reflect.Func,
			isPointer:   true,
			shouldFail:  false,
		},
	}
	for testName, testCase := range testCases {

		if err := CheckValidKind(testCase.inputObject, testCase.testForKind, testCase.isPointer); err != nil && !testCase.shouldFail || err == nil && testCase.shouldFail {
			t.Errorf("%s: failed. Should fail: %v. Actual result: %+v", testName, testCase.shouldFail, err)
			continue
		}
		// t.Log(testName, "... ok")
	}
}

type GetTypeAndValueTestCase struct {
	inputObject  interface{}
	expectedType reflect.Type
}

func TestGetTypeAndValue(t *testing.T) {
	// Interface
	var x interface{} = TestObj{}
	tObj := TestObj{}
	aux := invokeTestCases["Valid Call"].(invokeTestCase)
	tobjType := reflect.TypeOf(tObj)
	auxType := reflect.TypeOf(aux)

	testCases := map[string]GetTypeAndValueTestCase{
		"Direct Value": {
			inputObject:  TestObj{},
			expectedType: tobjType,
		},
		"Direct Value Pointer": {
			inputObject:  &TestObj{},
			expectedType: tobjType,
		},
		"Interface": {
			inputObject:  x,
			expectedType: tobjType,
		},
		"Interface Pointer": {
			inputObject:  &x,
			expectedType: nil,
		},
		"Intermediate Value": {
			inputObject:  tObj,
			expectedType: tobjType,
		},
		"Intermediate Value Pointer": {
			inputObject:  &tObj,
			expectedType: tobjType,
		},
		"Intermediate array ref": {
			inputObject:  invokeTestCases["Valid Call"],
			expectedType: auxType,
		},
		"Intermediate pointer to array ref": {
			inputObject:  &aux,
			expectedType: auxType,
		},
	}

	var checkResult = func(testCase GetTypeAndValueTestCase, tn reflect.Type, v reflect.Value) bool {
		validType := testCase.expectedType == nil || tn == testCase.expectedType
		validValue := v == reflect.ValueOf(testCase.inputObject) ||
			reflect.TypeOf(testCase.inputObject).Kind() == reflect.Ptr && reflect.ValueOf(testCase.inputObject).Elem() == v
		return validType && validValue
	}

	for testName, testCase := range testCases {
		if tn, v := GetTypeAndValue(testCase.inputObject); !checkResult(testCase, tn, v) {
			t.Errorf("%s: Invalid Type. Expected: %s, got: %s", testName, testCase.expectedType, tn)
			continue
		}
		// t.Log(testName, "... ok")
	}
}

func TestGetTagMap(t *testing.T) {
	t1 := struct {
		F1 string `k:"v1" k2:"v2" k3:"v3"`
		F2 string `` // empty tag
		F3 string // no tag
		f4 string `k:"Im hidden"`
		f5 string `k:"v1"k2:v3` //nolint:govet
	}{}
	expectedResult := map[string]string{
		"F1": "k:\"v1\" k2:\"v2\" k3:\"v3\"",
		"F2": "",
		"F3": "",
		"f4": `k:"Im hidden"`,
		"f5": `k:"v1"k2:v3`,
	}
	for field, tag := range GetTagMap(t1) {
		// t.Log("Field:", field, "Tag:", tag)
		if fmt.Sprintf("%v", tag) != expectedResult[field] {
			t.Errorf("%s: Unexpected tag: %q. Expected: %s", field, tag, expectedResult[field])
		}
	}
}

func TestGetFieldPointer(t *testing.T) {
	testObj := struct {
		StringField  string
		ComplexField TestObj
	}{
		StringField:  "Before",
		ComplexField: TestObj{c: 12},
	}

	TestCases := map[string]struct {
		changeTo        interface{}
		underlyingValue interface{}
	}{
		"StringField": {
			"Changed!",
			&(testObj.StringField),
		},
		"ComplexField": {
			TestObj{c: 12345},
			&(testObj.ComplexField),
		},
	}

	for fieldName, tc := range TestCases {
		// Let's have fun...
		if p, err := GetFieldPointer(&testObj, fieldName); err != nil {
			t.Errorf("unexpected error getting %s: %+v", fieldName, err)
		} else {
			switch p.(type) {
			case *string:
				ptr := p.(*string)
				expected := tc.underlyingValue.(*string)
				changeTo := tc.changeTo.(string)

				if ptr != expected {
					t.Errorf("The pointer does not point to the right place: %p != %p", ptr, expected)
					continue
				}
				*ptr = changeTo

				if *expected != changeTo {
					t.Errorf("Assignment failed: Invalid pointer received: %s, %s", *ptr, testObj.StringField)
					continue
				}
			case *TestObj: // I hate not having generics...
				ptr := p.(*TestObj)
				expected := tc.underlyingValue.(*TestObj)
				changeTo := tc.changeTo.(TestObj)

				if ptr != expected {
					t.Errorf("The pointer does not point to the right place: %p != %p", ptr, expected)
					continue
				}
				*ptr = changeTo

				if *expected != changeTo {
					t.Errorf("Assignment failed: Invalid pointer received: %+v, %s", *ptr, testObj.StringField)
					continue
				}
			default:
				t.Errorf("Invalid Type for stringField! I wanted a *string, got an %+v", reflect.TypeOf(p))
				continue
			}
		}
		// t.Log(fieldName, ".... ok")
	}

}

func TestGetFieldAsInterface(t *testing.T) {
	testObj := struct {
		StringField     string
		ComplexField    TestObj
		ComplexFieldPtr *TestObj
	}{
		StringField:     "Before",
		ComplexField:    TestObj{c: 12},
		ComplexFieldPtr: &TestObj{c: 12},
	}
	testCases := map[string]interface{}{
		"StringField":     testObj.StringField,
		"ComplexField":    testObj.ComplexField,
		"ComplexFieldPtr": testObj.ComplexFieldPtr,
	}

	for fieldName, expectedResult := range testCases {
		if f, err := GetFieldAsInterface(testObj, fieldName); err != nil {
			t.Errorf("Unexpected error %+v getting %s", err, fieldName)
		} else if !reflect.DeepEqual(f, expectedResult) {
			t.Errorf("Unexpected value. Got %+v, expected: %+v", f, expectedResult)
		}
	}
}

func TestGetFieldsWithTag(t *testing.T) {
	testObj := struct {
		F1 string `emptyTag:""`
		F2 string `filledTag:"hi" repeatedTag:""`
		F3 string
		F4 string `oneTag:"" twoTag:""`
		F5 string `repeatedTag:"aV"`
	}{"F1", "F2", "F3", "F4", "F5"}

	testCases := map[string]struct {
		values []string
		tags   []string
	}{
		"emptyTag":    {values: []string{"F1"}, tags: []string{""}},
		"filledTag":   {values: []string{"F2"}, tags: []string{"hi"}},
		"oneTag":      {values: []string{"F4"}, tags: []string{""}},
		"twoTag":      {values: []string{"F4"}, tags: []string{""}},
		"repeatedTag": {values: []string{"F2", "F5"}, tags: []string{"", "aV"}},
	}

	for tag, eR := range testCases {
		if f, tV := GetFieldsWithTag(testObj, tag); !reflect.DeepEqual(f, eR.values) || !reflect.DeepEqual(tV, eR.tags) {
			t.Errorf("Unexpected result getting tag %s. Got: %+v, expected: %+v", tag, f, eR)
		}
	}
}

func TestGetFieldsNames(t *testing.T) {
	out := struct {
		Field1 string
		Field2 int
		hidden string
	}{}
	expected := []string{"Field1", "Field2"}
	expectedAll := []string{"Field1", "Field2", "hidden"}

	if fields, err := GetFieldsNames(out, true); err != nil {
		t.Errorf("Unexpected error %+v", err)
	} else if !reflect.DeepEqual(fields, expected) {
		t.Errorf("Unexpected result. Expected: %+v, got: %+v", expected, fields)
	}

	if fields, err := GetFieldsNames(&out, true); err != nil {
		t.Errorf("Unexpected error %+v", err)
	} else if !reflect.DeepEqual(fields, expected) {
		t.Errorf("Unexpected result for pointer. Expected: %+v, got: %+v", expected, fields)
	}

	if fields, err := GetFieldsNames(out, false); err != nil {
		t.Errorf("Unexpected error %+v", err)
	} else if !reflect.DeepEqual(fields, expectedAll) {
		t.Errorf("Unexpected result. Expected: %+v, got: %+v", expectedAll, fields)
	}

	if fields, err := GetFieldsNames(&out, false); err != nil {
		t.Errorf("Unexpected error %+v", err)
	} else if !reflect.DeepEqual(fields, expectedAll) {
		t.Errorf("Unexpected result for pointer. Expected: %+v, got: %+v", expectedAll, fields)
	}

	if fields, err := GetFieldsNames(12, true); err == nil {
		t.Errorf("Unexpected valid result: %+v", fields)
	}

}

func TestConditionalCopy(t *testing.T) {
	// Source struct
	src := struct {
		F1        string
		F2        int
		DoNotCopy string
		WillCopy  string
		F4        string
		f3        int
	}{F1: "f1orig", F2: 12, F4: "f4orig", DoNotCopy: "srcNot", WillCopy: "New Value", f3: 5}

	// Control struct. We will copy only the fields that do not have the same value here and on dst
	control := struct {
		WillCopy  string
		DoNotCopy string
	}{WillCopy: "Control Value", DoNotCopy: "dstNot"}

	// Dst struct
	dst := struct {
		F1        string
		F2        string
		DoNotCopy string
		WillCopy  string
		f3        int
	}{F1: "f1dest", F2: "aa", DoNotCopy: control.DoNotCopy, WillCopy: "Will be changed", f3: 3}

	checkerFn := func(field string, dst, src interface{}) bool {
		cnt, err := GetFieldAsInterface(control, field)
		if err != nil {
			return true
		}
		return cnt != dst
	}

	// We want to copy F1
	t.Logf("Before copy:\nSRC: %+v\nDST: %+v", src, dst)

	if err := ConditionalCopy(&dst, src, checkerFn); err != nil {
		t.Errorf("unexpected error copying: %+v", err)
		return
	}

	// We want to copy F1
	t.Logf("Result after copy:\nSRC: %+v\nDST: %+v", src, dst)

	if dst.F1 != src.F1 {
		t.Errorf("F1 field should be copied and wasn't")
	}

	if dst.DoNotCopy == src.DoNotCopy {
		t.Errorf("DoNotCopy field should not be copied, and was")
	}

	if dst.WillCopy != src.WillCopy {
		t.Errorf("WillCopy should be copied and wasn't: %s - %s", dst.WillCopy, src.WillCopy)
	}
}

func TestConditionalCopy2(t *testing.T) {
	failCases := []interface{}{
		[]string{"1"},
		1,
		"hola",
	}
	validSrcOrDst := &struct {
		A int
	}{}

	for _, f := range failCases {
		if ConditionalCopy(validSrcOrDst, f, func(_ string, _, _ interface{}) bool { return true }) == nil {
			t.Errorf("Unexpected result for conditional copy. Expected error on src: %v", f)
		}
	}
	// Not a pointer
	failCases = append(failCases, struct {
		A int
	}{})

	for _, f := range failCases {
		if ConditionalCopy(f, validSrcOrDst, func(_ string, _, _ interface{}) bool { return true }) == nil {
			t.Errorf("Unexpected result for conditional copy. Expected error on dst: %v", f)
		}
	}
}

func TestGetNewElementForSlice(t *testing.T) {
	st := ""
	testCases := []struct {
		input        interface{}
		expectedType reflect.Type
	}{
		{
			input:        []string{},
			expectedType: reflect.TypeOf(&st),
		},
	}

	for _, tc := range testCases {
		if i := GetNewElementForSlice(tc.input); reflect.TypeOf(i) != tc.expectedType {
			t.Errorf("Unexpected type: %v of %+v. Expected: %v", reflect.TypeOf(i), i, tc.expectedType)
		}
	}
}

func TestAddElementToSlice(t *testing.T) {
	str := "c"
	arr := []string{"a", "b"}
	arrPtr := &arr
	testCases := []struct {
		input          interface{}
		elem           interface{}
		shouldDeref    bool
		expectedOutput interface{}
	}{
		{
			input:          []string{"a", "b"},
			elem:           "c",
			shouldDeref:    false,
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			input:          []string{"a", "b"},
			elem:           &str,
			shouldDeref:    true,
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			input:          arrPtr,
			elem:           "c",
			shouldDeref:    false,
			expectedOutput: []string{"a", "b", "c"},
		},
	}

	for _, tc := range testCases {
		if r := AddElementToSlice(tc.input, tc.elem, tc.shouldDeref); !reflect.DeepEqual(r, tc.expectedOutput) {
			t.Errorf("Unexpected result: %+v. Expected: %+v", r, tc.expectedOutput)
		}
	}
}

func TestAddElementsToSlice(t *testing.T) {
	testCases := []struct {
		input          interface{}
		elems          interface{}
		expectedOutput interface{}
	}{
		{
			input:          []string{"a", "b"},
			elems:          []string{"c", "d"},
			expectedOutput: []string{"a", "b", "c", "d"},
		},
	}

	for _, tc := range testCases {
		data := AddElementsToSlice(tc.input, tc.elems)
		if !reflect.DeepEqual(data, tc.expectedOutput) {
			t.Errorf("Unexpected result: %+v, Expected: %+v", data, tc.expectedOutput)
		}
	}
}

func TestStarSet(t *testing.T) {
	var out []string
	str := []string{"a", "b", "c"}

	StarSet(&out, str)
	if !reflect.DeepEqual(out, str) {
		t.Errorf("Unexpected value for out: %+v. Should be: %+v", out, str)
	}
}

func TestStarSet2(t *testing.T) {
	// Test decoding a set of jsons to a slice...
	testF := func(in []string, out interface{}) error {
		for _, jsonstr := range in {
			objPtr := GetNewElementForSlice(out)
			if err := json.Unmarshal([]byte(jsonstr), objPtr); err != nil {
				return fmt.Errorf("error unmarshaling %s: %v", jsonstr, err)
			}
			StarSet(out, AddElementToSlice(out, objPtr, true))
		}
		return nil
	}

	var out []struct {
		Key1 string
		Key2 int
	}
	input := []string{
		`{"key1": "value1", "key2": 12}`,
		`{"key1": "value2", "key2": 23}`,
		`{"key1": "value3", "key2": 34}`,
	}
	expectedOutput := []struct {
		Key1 string
		Key2 int
	}{
		{
			Key1: "value1",
			Key2: 12,
		},
		{
			Key1: "value2",
			Key2: 23,
		},
		{
			Key1: "value3",
			Key2: 34,
		},
	}

	if err := testF(input, &out); err != nil {
		t.Errorf("Unexpected error on aux function: %v", err)
	} else if !reflect.DeepEqual(out, expectedOutput) {
		t.Errorf("Unexpected value: %+v. Expected: %+v", out, expectedOutput)
	}

}

func TestSliceLen(t *testing.T) {
	st := "12345"
	testCases := []struct {
		input    interface{}
		expected int
	}{
		{
			input:    "hola",
			expected: 4,
		},
		{
			input:    &st,
			expected: 5,
		},
		{
			input:    []string{"a", "b"},
			expected: 2,
		},
		{
			input:    struct{}{},
			expected: 0,
		},
		{
			input:    &([]string{"a"}),
			expected: 1,
		},
	}

	for _, tc := range testCases {
		if sl := SliceLen(tc.input); sl != tc.expected {
			t.Errorf("Unexpected result for %v. Expected %d. Got %d", tc.input, tc.expected, sl)
		}
	}
}

func TestGetMapElem(t *testing.T) {
	testCases := []struct {
		obj   interface{}
		cases []struct {
			key         interface{}
			expected    interface{}
			shouldError bool
		}
	}{
		{
			obj: map[string]string{
				"key": "value",
			},
			cases: []struct {
				key         interface{}
				expected    interface{}
				shouldError bool
			}{
				{
					key:         "key",
					expected:    "value",
					shouldError: false,
				},
				{
					key:         "unknown",
					expected:    nil,
					shouldError: false,
				},
			},
		},
		{
			obj: struct {
				key string
			}{
				key: "value",
			},
			cases: []struct {
				key         interface{}
				expected    interface{}
				shouldError bool
			}{
				{
					key:         "key",
					expected:    "value",
					shouldError: true,
				},
			},
		},
	}

	for _, tc := range testCases {
		for _, c := range tc.cases {
			if elm, err := GetMapElem(tc.obj, c.key); err != nil && !c.shouldError || c.shouldError && err == nil {
				t.Errorf("unexpected error %v (%v)", err, c.shouldError)
			} else if err == nil && !reflect.DeepEqual(elm, c.expected) {
				t.Errorf("unexpected value returned (%v) for key %v. Expected %v", elm, c.key, c.expected)
			}
		}
	}
}

func TestStructToMap(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected map[string]interface{}
	}{
		{
			input: struct {
				A uint   `json:"fieldA"`
				B string `json:"fieldB;whatever"` //nolint:staticcheck
				C string
			}{A: 1, B: "soyB", C: "soyC"},
			expected: map[string]interface{}{"fieldA": uint(1), "fieldB": "soyB", "C": "soyC"},
		},
		{
			input: &struct {
				A uint   `json:"fieldA"`
				B string `json:"fieldB;whatever"` //nolint:staticcheck
				C string
			}{A: 1, B: "soyB", C: "soyC"},
			expected: map[string]interface{}{"fieldA": uint(1), "fieldB": "soyB", "C": "soyC"},
		},
		{
			input: struct {
				A map[string]int `json:"fieldA"`
				B string         `json:"fieldB;whatever"` //nolint:staticcheck
				C string
			}{A: nil, B: "soyB", C: "soyC"},
			expected: map[string]interface{}{"fieldA": nil, "fieldB": "soyB", "C": "soyC"},
		},
		{
			input:    map[string]uint{"A": 0, "B": 1, "C": 2},
			expected: nil,
		},
		{
			input:    map[string]interface{}{"A": 0, "B": 1, "C": 2},
			expected: map[string]interface{}{"A": 0, "B": 1, "C": 2},
		},
	}

	for _, tc := range testCases {
		if asMap := StructToMap(tc.input); !reflect.DeepEqual(asMap, tc.expected) {
			t.Errorf("unexpected returned value %+v for %+v. Expected %+v", asMap, tc.input, tc.expected)
		}
	}
}

func TestStructToMapUsingTag(t *testing.T) {
	testCases := []struct {
		input    interface{}
		tag      string
		expected map[string]interface{}
	}{
		{
			input: struct {
				A uint   `json:"fieldA"`
				B string `json:"fieldB;whatever"` //nolint:staticcheck
				C string
			}{A: 1, B: "soyB", C: "soyC"},
			tag:      "json",
			expected: map[string]interface{}{"fieldA": uint(1), "fieldB": "soyB", "C": "soyC"},
		},
		{
			input: &struct {
				A uint   `bson:"fieldA"`
				B string `bson:"fieldB;whatever"` //nolint:staticcheck
				C string
			}{A: 1, B: "soyB", C: "soyC"},
			tag:      "bson",
			expected: map[string]interface{}{"fieldA": uint(1), "fieldB": "soyB", "C": "soyC"},
		},
		{
			input: struct {
				A map[string]int `whatever:"fieldA"`
				B string         `json:"fieldB;whatever"` //nolint:staticcheck
				C string
			}{A: nil, B: "soyB", C: "soyC"},
			tag:      "whatever",
			expected: map[string]interface{}{"fieldA": nil, "B": "soyB", "C": "soyC"},
		},
		{
			input:    map[string]uint{"A": 0, "B": 1, "C": 2},
			expected: nil,
		},
		{
			input:    map[string]interface{}{"A": 0, "B": 1, "C": 2},
			expected: map[string]interface{}{"A": 0, "B": 1, "C": 2},
		},
	}

	for _, tc := range testCases {
		if asMap := StructToMapUsingTag(tc.input, tc.tag); !reflect.DeepEqual(asMap, tc.expected) {
			t.Errorf("unexpected returned value %+v for %+v. Expected %+v", asMap, tc.input, tc.expected)
		}
	}
}

func TestGetFieldsOfKind(t *testing.T) {
	testObj := struct {
		FieldA string
		FieldB int
		Map1   map[string]string
		Map2   map[string]int
		fieldC string
	}{
		FieldA: "hi",
		FieldB: 1,
		fieldC: "bye",
		Map1: map[string]string{
			"in": "out",
			"hi": "bye",
		},
		Map2: map[string]int{
			"in": 1,
			"hi": 2,
		},
	}
	testCases := []struct {
		in       interface{}
		kind     reflect.Kind
		expected map[string]interface{}
	}{
		{
			in:   testObj,
			kind: reflect.String,
			expected: map[string]interface{}{
				"FieldA": "hi",
			},
		},
		{
			in:   testObj,
			kind: reflect.Int,
			expected: map[string]interface{}{
				"FieldB": 1,
			},
		},
		{
			in:   testObj,
			kind: reflect.Map,
			expected: map[string]interface{}{
				"Map1": map[string]string{
					"in": "out",
					"hi": "bye",
				},
				"Map2": map[string]int{
					"in": 1,
					"hi": 2,
				},
			},
		},
	}

	for _, tc := range testCases {
		if out, err := GetFieldsOfKind(tc.kind, tc.in); !reflect.DeepEqual(out, tc.expected) {
			t.Errorf("unexpected result %+v. Expected %+v. Error: %v", out, tc.expected, err)
		}
	}

}

type aux struct {
	A int
}

func TestGetMapOf(t *testing.T) {

	testCases := []struct {
		key      interface{}
		value    interface{}
		expected interface{}
	}{
		{
			key:      "string",
			value:    "string",
			expected: map[string]string{},
		},
		{
			key:      "string",
			value:    aux{},
			expected: map[string]aux{},
		},
		{
			key:      "string",
			value:    &aux{},
			expected: map[string]*aux{},
		},
	}

	for _, tc := range testCases {
		if typ := reflect.TypeOf(GetMapOf(tc.key, tc.value)); typ != reflect.TypeOf(tc.expected) {
			t.Errorf("Unexpected type for GetMapOf: %v, expected %v", typ, reflect.TypeOf(tc.expected))
		}
	}
}

func TestSetMapElem(t *testing.T) {
	intMap := GetMapOf("", aux{})
	auxValue := aux{1}
	if err := SetMapElem(intMap, "hola", auxValue); err != nil {
		t.Errorf("Unexpected error setting map value: %v", err)
		return
	}
	strMap := intMap.(map[string]aux)
	if !reflect.DeepEqual(strMap["hola"], auxValue) {
		t.Errorf("Value not set correctly")
	}
}

func TestPtrToElem(t *testing.T) {
	a := "hola"
	strPtr := &a
	testCases := []struct {
		in              interface{}
		expectedPtrType interface{}
		expectedType    reflect.Type
	}{
		{
			in:              "hi",
			expectedPtrType: reflect.TypeOf(strPtr),
			expectedType:    reflect.TypeOf("hi"),
		},
		{
			in:              strPtr,
			expectedPtrType: reflect.TypeOf(strPtr),
			expectedType:    reflect.TypeOf("hi"),
		},
		{
			in:              aux{},
			expectedPtrType: reflect.TypeOf(&aux{}),
			expectedType:    reflect.TypeOf(aux{}),
		},
		{
			in:              &aux{},
			expectedPtrType: reflect.TypeOf(&aux{}),
			expectedType:    reflect.TypeOf(aux{}),
		},
	}

	for _, tc := range testCases {
		ptr, baseType := PtrToElem(tc.in)
		if typ := reflect.TypeOf(ptr); typ != tc.expectedPtrType {
			t.Errorf("Unexpected type: %v, expected: %v", typ, tc.expectedPtrType)
		}
		if baseType != tc.expectedType {
			t.Errorf("Unexpected type: %v, expected: %v", baseType, tc.expectedType)
		}

	}

}

func TestGetSliceOf(t *testing.T) {
	testCases := []struct {
		in       interface{}
		expected interface{}
	}{
		{
			in:       "string",
			expected: []string{},
		},
		{
			in:       aux{},
			expected: []aux{},
		},
		{
			in:       &aux{},
			expected: []*aux{},
		},
	}

	for _, tc := range testCases {
		rv := GetSliceOf(tc.in, 0, 5)
		if typ := reflect.TypeOf(rv); typ != reflect.TypeOf(tc.expected) {
			t.Errorf("Unexpected returned type. Expected: %v, got: %v", reflect.TypeOf(tc.expected), typ)
		}
	}
}

func TestGetSliceOfType(t *testing.T) {
	testCases := []struct {
		in       reflect.Type
		expected interface{}
	}{
		{
			in:       reflect.TypeOf("string"),
			expected: []string{},
		},
		{
			in:       reflect.TypeOf(aux{}),
			expected: []aux{},
		},
		{
			in:       reflect.TypeOf(&aux{}),
			expected: []*aux{},
		},
	}

	for _, tc := range testCases {
		rv := GetSliceOfType(tc.in, 0, 5)
		if typ := reflect.TypeOf(rv); typ != reflect.TypeOf(tc.expected) {
			t.Errorf("Unexpected returned type. Expected: %v, got: %v", reflect.TypeOf(tc.expected), typ)
		}
	}
}

func TestGetBaseElem(t *testing.T) {
	a := struct {
		A int
	}{A: 1}
	pA := &a
	ppA := &pA
	testCases := []interface{}{
		&a, pA, ppA,
	}
	for i, tc := range testCases {
		if x := GetBaseElem(tc); x != a {
			t.Errorf("Unexpected error for test %d -- Got: %+v", i, x)
		}
	}
}

func TestCopyElement(t *testing.T) {
	a := struct {
		A int
	}{A: 1}
	pA := &a
	ppA := &pA
	testCases := []interface{}{
		a,
		pA,
		ppA,
	}
	for i, tc := range testCases {
		if y := GetBaseElem(CopyElement(tc)); !reflect.DeepEqual(y, a) {
			t.Errorf("Unexpected error for test %d -- Got: %+v and expected %+v", i, y, a)
		}
	}

}
