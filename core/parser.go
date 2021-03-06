package core

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/evilsocket/islazy/log"
)

type Tokens map[string]string

var mandatoryTokens = []string{
	"address",
	"datetime",
}

var (
	currYear = time.Now().Year()
)

type Parser struct {
	DatetimeFormat string         `yaml:"datetime_format"`
	Expression     string         `yaml:"expression"`
	Tokens         map[string]int `yaml:"tokens"`
	compiled       *regexp.Regexp
	maxIndex       int
}

func (p *Parser) Compile() (err error) {
	for _, t := range mandatoryTokens {
		if _, found := p.Tokens[t]; !found {
			return fmt.Errorf("mandatory token %s not found in parser", t)
		}
	}

	for _, index := range p.Tokens {
		if index > p.maxIndex {
			p.maxIndex = index
		}
	}

	expr := p.Expression
	if !strings.HasPrefix(expr, "(?i)") {
		expr = "(?i)" + expr
	}

	log.Debug("compiling parser '%s'", expr)

	p.compiled, err = regexp.Compile(expr)
	return
}

func hasYear(value string) bool {
	for year := currYear - 5; year <= currYear; year++ {
		if strings.Contains(value, fmt.Sprintf("%d", year)) {
			return true
		}
	}
	return false
}

func (p *Parser) Parse(line string) (matched bool, tokens Tokens) {
	if m := p.compiled.FindStringSubmatch(line); len(m) >= p.maxIndex {

		matched = true
		tokens = make(map[string]string)
		for token, index := range p.Tokens {
			value := m[index]
			// ugly hack to handle formats withtout the year like sshd
			if token == "datetime" && !hasYear(value) {
				value = fmt.Sprintf("%d %s", currYear, value)
			}

			tokens[token] = value
		}
	}
	return
}
