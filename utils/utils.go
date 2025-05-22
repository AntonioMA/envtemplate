package utils

import (
	"envtemplate/reflection"
	"flag"
	"fmt"
	"strings"
	"time"
)

// DefineCommandLineFlags sets the command line flags from an annotated object. This method will process
// only the attributes of options that have the following annotations:
//
//	flag: The attribute can be filled from a CLI flag. The format for this annotation is
//	      name[,name]+;Usage
//
// options *must* be a pointer to an struct or this will fail
// The default value for each param will be the current value of the corresponding field on the
// defaults object (which should be of the same type as options). If nil is passed as defaults
// then the default values will be captured from options
func DefineCommandLineFlags(options any, defaults any) (err error) {
	if defaults == nil {
		defaults = options
	}
	err = nil
	for fieldName, tag := range reflection.GetTagMap(options) {
		clFlag := tag.Get("flag")
		if len(clFlag) == 0 {
			continue
		}
		parts := strings.Split(clFlag, ";")
		var names []string = nil
		usage := ""
		if len(parts) >= 1 {
			names = strings.Split(parts[0], ",")
		}
		if len(parts) >= 2 {
			usage = parts[1]
		}

		ptr, err := reflection.GetFieldPointer(options, fieldName)
		if err != nil {
			return fmt.Errorf("cannot get pointer of %s: %+v", fieldName, err)
		}

		def, err := reflection.GetFieldAsInterface(defaults, fieldName)
		if err != nil {
			return fmt.Errorf("cannot get default value of %s: %+v", fieldName, err)
		}

		// Known types:
		// StringVar
		// BoolVar
		// DurationVar
		// IntVar
		// UintVar
		// Float64Var
		// Uint64Var
		// Int64Var
		// Var (implements flag.Value)
		// I predict lots of C&P in my near future
		switch typedVal := ptr.(type) {
		case *string:
			for _, name := range names {
				flag.StringVar(typedVal, name, def.(string), usage)
			}
		case *bool:
			for _, name := range names {
				flag.BoolVar(typedVal, name, def.(bool), usage)
			}
		case *time.Duration:
			for _, name := range names {
				flag.DurationVar(typedVal, name, def.(time.Duration), usage)
			}
		case *int:
			for _, name := range names {
				flag.IntVar(typedVal, name, def.(int), usage)
			}
		case *uint:
			for _, name := range names {
				flag.UintVar(typedVal, name, def.(uint), usage)
			}
		case *float64:
			for _, name := range names {
				flag.Float64Var(typedVal, name, def.(float64), usage)
			}
		case *uint64:
			for _, name := range names {
				flag.Uint64Var(typedVal, name, def.(uint64), usage)
			}
		case *int64:
			for _, name := range names {
				flag.Int64Var(typedVal, name, def.(int64), usage)
			}

		default:
			asValue := ptr.(flag.Value)
			for _, name := range names {
				flag.Var(asValue, name, usage)
			}
		}

	}
	return

}
