package lib

import (
	"fmt"
	"github.com/AntonioMA/go-utils/template"
	"os"
	"regexp"
)

// TemplateData is the data that will be passed to the template evaluator.
type TemplateData map[string]template.ExtendedString

// Filter returns a subset of T where the keys match the passed pattern. It will return an empty
// map and log an error if the pattern is not a valid one
func (t TemplateData) Filter(pattern string) TemplateData {
	exp, err := regexp.Compile(pattern)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Invalid pattern: %s - error: %v", pattern, err)
		return TemplateData{}
	}
	rv := make(TemplateData, len(t))
	for k, v := range t {
		if exp.MatchString(k) {
			rv[k] = v
		}
	}
	return rv
}
