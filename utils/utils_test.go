package utils

import (
	"flag"
	"reflect"
	"strings"
	"testing"
	"time"
)

// SomeValue implements flag.Value
type SomeValue struct {
	AField   string
	TwoField string
}

func (s *SomeValue) String() string {
	return s.AField + "::" + s.TwoField
}

func (s *SomeValue) Set(v string) error {
	p := strings.Split(v, "::")
	s.AField = p[0]
	s.TwoField = p[1]
	return nil
}

func TestDefineCommandLineFlags(t *testing.T) {
	testFlags1 := struct {
		StringVar   string        `flag:"string,S;This is a string param"`
		BoolVar     bool          `flag:"bool,B;This is a bool param"`
		DurationVar time.Duration `flag:"duration,D;This is a duration param"`
		IntVar      int           `flag:"int,I;This is an int param"`
		DefaultInt  int           `flag:"defaultI,dI;This is an int param that will be filled from a default"`
	}{"somevalue", false, time.Duration(0), 123, -768}
	testFlags2 := struct {
		UintVar    uint      `flag:"uint,U;This is an uint param"`
		Float64Var float64   `flag:"float64,F;This is a float64 param"`
		Uint64Var  uint64    `flag:"uint64,UI;This is an uint64 param"`
		Int64Var   int64     `flag:"int64,I64;This is an int64 param"`
		Var        SomeValue `flag:"someValue,sV;This is a SomeValue param"`
	}{1234, 1234.12, 123456, -1234567, SomeValue{}}

	testCommandLine := "-S cadena -D 12ns -I -123 -U 1234 -F 1234.12 -UI 123456 -int64 1234567 -sV abcd::1234"
	// Note that the tags here are only needed because DeepEqual croaks otherwise.
	expectedFlags1 := struct {
		StringVar   string        `flag:"string,S;This is a string param"`
		BoolVar     bool          `flag:"bool,B;This is a bool param"`
		DurationVar time.Duration `flag:"duration,D;This is a duration param"`
		IntVar      int           `flag:"int,I;This is an int param"`
		DefaultInt  int           `flag:"defaultI,dI;This is an int param that will be filled from a default"`
	}{StringVar: "cadena", BoolVar: false, DurationVar: time.Duration(12), IntVar: -123, DefaultInt: -768}

	expectedFlags2 := struct {
		UintVar    uint      `flag:"uint,U;This is an uint param"`
		Float64Var float64   `flag:"float64,F;This is a float64 param"`
		Uint64Var  uint64    `flag:"uint64,UI;This is an uint64 param"`
		Int64Var   int64     `flag:"int64,I64;This is an int64 param"`
		Var        SomeValue `flag:"someValue,sV;This is a SomeValue param"`
	}{
		UintVar: 1234, Float64Var: 1234.12, Uint64Var: 123456, Int64Var: 1234567,
		Var: SomeValue{AField: "abcd", TwoField: "1234"},
	}

	testArgs := strings.Split(testCommandLine, " ")
	_ = DefineCommandLineFlags(&testFlags1, nil)
	_ = DefineCommandLineFlags(&testFlags2, nil)
	_ = flag.CommandLine.Parse(testArgs)

	if !reflect.DeepEqual(testFlags1, expectedFlags1) {
		t.Errorf("First set of flags failure. Expected: %+v, Got: %+v", expectedFlags1, testFlags1)
	}
	if !reflect.DeepEqual(testFlags2, expectedFlags2) {
		t.Errorf("Second set of flags failure. Expected: %+v, Got: %+v", expectedFlags2, testFlags2)
	}
}
