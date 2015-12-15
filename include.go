package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"encoding/json"
	"strings"
	"path/filepath"
	"io/ioutil"
)

const (
	kindRaw = "raw"
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

var key = flag.String("key", "", "constant name")
var file = flag.String("path", "", "absolute path to file")
var output = flag.String("out", "generated_include.go", "output filename")
var kind = &Kind{[]string{kindRaw, kindJson}, kindRaw}

func main() {
	flag.Var(kind, "type", fmt.Sprintf("how to parse input file (%s)", strings.Join(kind.values, ", ")))
	flag.Parse()

	if *file == "" {
		flag.Usage()
		os.Exit(1)
	}

	fd, err := os.Open(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open file: %s", err)
		os.Exit(1)
	}

	out, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create file: %s", err)
		os.Exit(1)
	}

	out.WriteString("package main\n\n")

	switch kind.String() {
	case kindRaw:
		var name string
		if *key == "" {
			name = filepath.Base(*file)
			name = strings.TrimSuffix(name, filepath.Ext(name))
		} else {
			name = *key
		}

		fmt.Fprintf(out, "const %s = `", name)
		io.Copy(out, fd)
		out.WriteString("`\n")
	case kindJson:
		var obj map[string]interface{}
		b, err := ioutil.ReadAll(fd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not read file: %s", err)
			os.Exit(1)
		}

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
				fmt.Fprintf(os.Stderr, "nested structs is not supported")
				os.Exit(1)
			}
		}
	}

	fd.Close()
	os.Exit(1)
}
