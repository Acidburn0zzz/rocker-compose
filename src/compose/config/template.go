package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"text/template"
)

func ProcessConfigTemplate(name string, reader io.Reader, vars map[string]interface{}, funcs map[string]interface{}) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	// read template
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Error reading template %s, error: %s", name, err)
	}
	funcMap := map[string]interface{}{
		"default": fnDefault,
		"seq":     seq,
	}
	for k, f := range funcs {
		funcMap[k] = f
	}
	tmpl, err := template.New(name).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("Error parsing template %s, error: %s", name, err)
	}
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("Error executing template %s, error: %s", name, err)
	}
	return &buf, nil
}

func fnDefault(defaultVal interface{}, actualValue ...interface{}) interface{} {
	if len(actualValue) > 0 {
		return actualValue[0]
	} else {
		return defaultVal
	}
}

func seq(args ...interface{}) ([]int, error) {
	l := len(args)
	if l == 0 {
		return nil, fmt.Errorf("seq helper expects from 1 to 3 arguments, 0 given")
	} else if l > 3 {
		return nil, fmt.Errorf("seq helper expects from 1 to 3 arguments, %d given", l)
	}
	intArgs := make([]int, l)
	for i, v := range args {
		n, err := interfaceToInt(v)
		if err != nil {
			return nil, err
		}
		intArgs[i] = n
	}
	return doSeq(intArgs[0], intArgs[1:]...)
}

func doSeq(n int, args ...int) ([]int, error) {
	var (
		from, to, step int
	)

	switch len(args) {
	// {{ seq To }}
	case 0:
		// {{ seq 0 }}
		if n == 0 {
			return []int{}, nil
		}
		if n > 0 {
			// {{ seq 15 }}
			from, to, step = 1, n, 1
		} else {
			// {{ seq -15 }}
			from, to, step = -1, n, 1
		}
	// {{ seq From To }}
	case 1:
		from, to, step = n, args[0], 1

	// {{ seq From To Step }}
	case 2:
		from, to, step = n, args[0], args[1]
	}

	if step <= 0 {
		return nil, fmt.Errorf("step should be a positive integer, `%#v` given", step)
	}

	// reverse order
	if from > to {
		res := make([]int, ((from-to)/step)+1)
		i := 0
		for k := from; k >= to; k = k - step {
			res[i] = k
			i++
		}
		return res, nil
	}

	// straight order
	res := make([]int, ((to-from)/step)+1)
	i := 0
	for k := from; k <= to; k = k + step {
		res[i] = k
		i++
	}
	return res, nil
}

func interfaceToInt(v interface{}) (int, error) {
	switch v.(type) {
	case int:
		return v.(int), nil
	case string:
		n, err := strconv.ParseInt(v.(string), 10, 64)
		if err != nil {
			return 0, err
		}
		return (int)(n), nil
	default:
		return 0, fmt.Errorf("Cannot receive %#v, int or string is expected", v)
	}
}
