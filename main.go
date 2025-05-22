package main

import (
	"envtemplate/lib"
	templateUtils "envtemplate/template"
	"envtemplate/utils"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type commandlineFlags struct {
	OutputFile string `flag:"o,out;File to write the result to"`
	InputFile  string `flag:"i,in;File to read the template from"`
}

func checkOptions(cf commandlineFlags) (writer io.Writer, tmplt *template.Template, err error) {
	reader := os.Stdin
	writer = os.Stdout
	err = nil

	if len(cf.OutputFile) > 0 {
		if writer, err = os.Create(cf.OutputFile); err != nil {
			err = fmt.Errorf("cannot open output file %s. Error: %+v\n", cf.OutputFile, err)
			return
		}
	}

	if len(cf.InputFile) > 0 {
		if reader, err = os.Open(cf.InputFile); err != nil {
			err = fmt.Errorf("cannot open input file %s. Error: %+v\n", cf.OutputFile, err)
			return
		}
	}
	var tmplData []byte

	if tmplData, err = ioutil.ReadAll(reader); err != nil {
		err = fmt.Errorf("error parsing input template (%s): %v", cf.InputFile, err)
		return
	}

	tmplt = template.
		New("root").
		Delims("{[", "]}").
		Option("missingkey=zero")
	if tmplt, err = tmplt.Funcs(sprig.FuncMap()).Parse(string(tmplData)); err != nil {
		err = fmt.Errorf("error parsing template: %v\n", err)
		return
	}
	return
}

// Can't believe something like this doesn't exist already...
func getEnvMap() lib.TemplateData {
	envAssignments := os.Environ()
	envMap := make(map[string]templateUtils.ExtendedString, len(envAssignments))
	rexp, _ := regexp.Compile(`%(?P<VARNAME>[\w-]+)%`)
	for _, envAssignment := range os.Environ() {
		envVar := strings.SplitN(envAssignment, "=", 2)
		envVar[1] = os.ExpandEnv(string(rexp.ReplaceAll([]byte(envVar[1]), []byte(`${$VARNAME}`))))
		envMap[envVar[0]] = templateUtils.ExtendedString(envVar[1])
	}
	return envMap
}

func main() {
	defaultFlags := commandlineFlags{
		InputFile:  "",
		OutputFile: "",
	}
	outputFlags := commandlineFlags{}
	if err := utils.DefineCommandLineFlags(&outputFlags, defaultFlags); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err)
	}
	flag.Parse()

	outputFile, tmplt, err := checkOptions(outputFlags)

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error in options: %v\n", err)
		os.Exit(1)
	}

	if err := tmplt.Execute(outputFile, getEnvMap()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error generating file: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)

}

/*
{{range $num, $info := .}}{{$num}} {{$info}}{{end}}

{{range $num, $info := .}}{{$num}} {{$info.String}} {{end}}

{{range $label, $objectAtts := .}}
  {{$label}}:
   {{range $att, $val := $objectAtts}}
     {{$att}} => {{.String}} -- {{len $val.Value}} bytes
   {{end}}
{{end}}

*/
