# envtemplate
# What it this
This is a simple application that evaluates a Go Template file. The data passed
to the template evaluation has the following type:
```
type TemplateData map[string]ExtendedString
```

Where `TemplateData` is a map with the values from all the environment variables, using the variable name
as key and the variable value as value. Variable values are expanded, and if a variable value has
an expression like %WHATEVER%, that will be replaced with ${WHATEVER} before expanding.

The %VARIABLE% syntax allows late expansion. So for example you can have something like:

```
export VARIABLE="Early"
export MY_VAR="Early value: ${VARIABLE}. Late Value: %VARIABLE%
# some code here...
# much later
VARIABLE="Later gator"
```

And the value of `MY_VARIABLE` as passed to the template will be:

`Early value: Early. Late Value: Later gator`

ExtendedString is a string extended with the following functions:

* Split(separator string) []ExtendedString => Splits the string by the passed in separator and
  returns an array of strings. This allows using the result on a range, such as:
```
{[range $index, $elem := .Env.TESTVAR.Split ","]}
  Index: {[$index]}
  Elem: {[$elem]}
{[end]}
```
* Fields []ExtendedString => Splits the string by any whitespace
* LoadFile ExtendedString => Treats the string as a filename and tries loading it and
  returning the content as a single string
* ToJSON ExtendedString: JSONifies the string and returns it
* ToBase64 ExtendedString: Converts the string to base64 (standard encoding) and returns it
