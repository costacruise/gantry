package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type env map[string]string

func (e *env) Set(s string) error {
	if *e == nil {
		*e = make(env)
	}
	entries := strings.SplitN(s, "=", 2)
	if len(entries) != 2 {
		return errors.Errorf("malformed env: key values must be separated with '=': %q", s)
	}
	(*e)[entries[0]] = entries[1]
	return nil
}

func (e env) String() string {
	out := e.ToEnviron()
	sort.Strings(out)
	return strings.Join(out, ",")
}

// ToEnviron returns a copy of strings representing the environment, in the form "key=value".
// It is compatiple to os.Environ() and exec.Cmd.Env.
func (e env) ToEnviron() []string {
	out := []string{}
	for k, v := range e {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}
