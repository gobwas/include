package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"bytes"
	"go/build"
)

const (
	kindRaw  = "raw"
	kindToml = "toml"
	kindJson = "json"
)

type Kind struct {
	values []string
	value  string
}

func (e *Kind) Set(s string) error {
	for _, v := range e.values {
		if v == s {
			e.value = s
			return nil
		}
	}

	return fmt.Errorf("unsupported value: %s", s)
}

func (e *Kind) String() string {
	return e.value
}

var key = flag.String("name", "", "constant name")
var file = flag.String("file", "", "absolute path to file")
var output = flag.String("out", "generated_include.go", "output filename")
var trim = flag.Bool("trim", true, "trim new line characters")
var kind = &Kind{[]string{kindRaw, kindJson}, kindRaw}

const doNotEdit = `
// DO NOT EDIT!
// Code generated by https://github.com/gobwas/include

`

func main() {
	flag.Var(kind, "parse", fmt.Sprintf("how to parse input file (%s)", strings.Join(kind.values, ", ")))
	flag.Parse()

	if *file == "" {
		flag.Usage()
		os.Exit(1)
	}

	pack, err := findPackageName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not determ package of generated code: %s", err)
		os.Exit(1)
	}

	fd, err := os.Open(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open file: %s", err)
		os.Exit(1)
	}

	b, err := ioutil.ReadAll(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read file: %s", err)
		os.Exit(1)
	}

	out, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create file: %s", err)
		os.Exit(1)
	}

	out.WriteString(doNotEdit)
	fmt.Fprintf(out, "package %s\n\n", pack)

	switch kind.String() {
	case kindRaw:
		var name string
		if *key == "" {
			name = filepath.Base(*file)
			name = strings.TrimSuffix(name, filepath.Ext(name))
		} else {
			name = *key
		}

		if *trim {
			fmt.Fprintf(out, "const %s = `%s`\n", name, string(bytes.Trim(b, "\n")))
		} else {
			fmt.Fprintf(out, "const %s = `%s`\n", name, string(b))
		}

	case kindJson:
		var obj map[string]interface{}
		err = json.Unmarshal(b, &obj)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not parse json: %s", err)
			os.Exit(1)
		}

		for key, value := range obj {
			switch v := value.(type) {
			case string:
				fmt.Fprintf(out, "const %s = `%s`\n", key, v)

			case float64:
				fmt.Fprintf(out, "const %s = %f\n", key, v)

			case bool:
				fmt.Fprintf(out, "const %s = %t", key, v)

			default:
				fmt.Fprintf(os.Stderr, "nested structs is not supported in json")
				os.Exit(1)
			}
		}
	}

	fd.Close()
	os.Exit(0)
}

func findPackageName() (string, error) {
	p, err := build.Default.Import(".", ".", build.ImportMode(0))
	if err != nil {
		return "", err
	}

	return p.Name, nil
}
