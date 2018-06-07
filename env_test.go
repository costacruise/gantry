package main

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_EnvSet(t *testing.T) {
	tests := []struct {
		name  string
		input string
		env   env
		err   string
	}{
		{
			name:  "happy path",
			input: "FOO=bar",
			env:   env{"FOO": "bar"},
			err:   "<nil>",
		},
		{
			name:  "malformed env missing =",
			input: "FOO bar",
			env:   env{},
			err:   "malformed env: key values must be separated with '=': \"FOO bar\"",
		},
		{
			name:  "malformed env no value",
			input: "FOO",
			env:   env{},
			err:   "malformed env: key values must be separated with '=': \"FOO\"",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var environ env

			err := environ.Set(testCase.input)

			errStr := fmt.Sprintf("%v", err)
			if errStr != testCase.err {
				t.Fatalf("expected error to equal %q, got %v", testCase.err, err)
			}

			if !reflect.DeepEqual(testCase.env, environ) {
				t.Fatalf("expected env to equal '%v', got '%v", testCase.env, environ)
			}
		})

	}

	t.Run("multiple Sets", func(t *testing.T) {
		inputs := []string{"FOO=bar", "QUX=42"}

		var environ env

		for _, input := range inputs {
			err := environ.Set(input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		}

		expected := env{
			"FOO": "bar",
			"QUX": "42",
		}

		if !reflect.DeepEqual(expected, environ) {
			t.Fatalf("expected env to equal '%v', got '%v", expected, environ)
		}
	})
}

func Test_Env_String(t *testing.T) {
	e := env{
		"foo":  "42",
		"bar":  "false",
		"zicl": "1",
	}

	actual := e.String()

	expected := "bar=false,foo=42,zicl=1"

	if expected != actual {
		t.Fatalf("expected env.String to return %q, got %q", expected, actual)
	}
}
