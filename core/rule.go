package core

import (
	"regexp"

	"github.com/evilsocket/islazy/log"
)

type Rule struct {
	Name        string `yaml:"name"`
	Token       string `yaml:"token"`
	Description string `yaml:"description"`
	Expression  string `yaml:"expression"`
	compiled    *regexp.Regexp
}

func (r *Rule) Compile() (err error) {
	log.Debug("compiling rule '%s'", r.Expression)
	r.compiled, err = regexp.Compile(r.Expression)
	return
}

func (r *Rule) Match(tokens Tokens) (matched bool, value string) {
	if token, found := tokens[r.Token]; found {
		if r.compiled.MatchString(token) {
			matched = true
			value = token
		}
	}
	return
}
