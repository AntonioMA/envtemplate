package template

import (
	"reflect"
	"testing"
)

func TestExtendedString_Fields(t *testing.T) {
	tests := []struct {
		name string
		es   ExtendedString
		want []ExtendedString
	}{
		{
			name: "Multiple values single space",
			es:   "a b c d",
			want: []ExtendedString{"a", "b", "c", "d"},
		},
		{
			name: "Multiple values multiple space",
			es:   "a   b  c     d",
			want: []ExtendedString{"a", "b", "c", "d"},
		},
		{
			name: "Single",
			es:   "abcd",
			want: []ExtendedString{"abcd"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.es.Fields(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Fields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtendedString_LoadFile(t *testing.T) {
	tests := []struct {
		name string
		es   ExtendedString
		want ExtendedString
	}{
		{
			name: "Text File",
			es:   "./test/sample_file.txt",
			want: "This is a test file. Do not change me",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.es.LoadFile(); got != tt.want {
				t.Errorf("LoadFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtendedString_LoadRelativeFile(t *testing.T) {
	type args struct {
		basePath string
	}
	tests := []struct {
		name string
		es   ExtendedString
		args args
		want ExtendedString
	}{
		{
			name: "Text file",
			es:   "test/sample_file.txt",
			args: args{
				basePath: ".",
			},
			want: "This is a test file. Do not change me",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.es.LoadRelativeFile(tt.args.basePath); got != tt.want {
				t.Errorf("LoadRelativeFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtendedString_LoadRelativeFileES(t *testing.T) {
	type args struct {
		basePath ExtendedString
	}
	tests := []struct {
		name string
		es   ExtendedString
		args args
		want ExtendedString
	}{
		{
			name: "Text file",
			es:   "test/sample_file.txt",
			args: args{
				basePath: ".",
			},
			want: "This is a test file. Do not change me",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.es.LoadRelativeFileES(tt.args.basePath); got != tt.want {
				t.Errorf("LoadRelativeFileES() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtendedString_Split(t *testing.T) {
	type args struct {
		sep string
	}
	tests := []struct {
		name string
		es   ExtendedString
		args args
		want []ExtendedString
	}{
		{
			name: "Single char separator",
			es:   "a,b,c,d",
			args: args{
				sep: ",",
			},
			want: []ExtendedString{"a", "b", "c", "d"},
		},
		{
			name: "Multi char separator",
			es:   "a, b, c, d",
			args: args{
				sep: ", ",
			},
			want: []ExtendedString{"a", "b", "c", "d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.es.Split(tt.args.sep); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtendedString_ToBase64(t *testing.T) {
	tests := []struct {
		name string
		es   ExtendedString
		want ExtendedString
	}{
		{
			name: "Simple string",
			es:   "abcd1234",
			want: "YWJjZDEyMzQ=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.es.ToBase64(); got != tt.want {
				t.Errorf("ToBase64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtendedString_ToJSON(t *testing.T) {
	tests := []struct {
		name string
		es   ExtendedString
		want ExtendedString
	}{
		{
			name: "Simple string",
			es:   "abcd1234",
			want: `"abcd1234"`,
		},
		{
			name: "String with quotes",
			es:   `abcd"1234`,
			want: `"abcd\"1234"`,
		},
		{
			name: "Multi line string",
			es: `abcd
1234`,
			want: `"abcd\n1234"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.es.ToJSON(); got != tt.want {
				t.Errorf("ToJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
