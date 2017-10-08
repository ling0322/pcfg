package pcfg

import (
	"strings"
	"github.com/pkg/errors"
	"fmt"
	"regexp"
	"strconv"
)

// Symbol represents a symbol in PCFG rule, both terminal and non-terminal
type Symbol string

// InternalSymbol creates an internal non-terminal symbol from name
func InternalSymbol(name string) Symbol {
	return Symbol("<__" + strings.TrimSpace(name) + ">")
}

// The build-in symbol
const EpsilonSymbol = Symbol("<nil>")
const RootSymbol = Symbol("<root>")

// IsValid checks the symbol string is valid
func (s Symbol) IsValid() bool {
	matched, err := regexp.MatchString("^(<\\??[-\\w]+>|[^<>\"?|]+)$", string(s))
	checkAndFatal(err)
	return matched
}

// IsTerminal checks if it is a terminal symbol, assuming s.IsValid() == true
func (s Symbol) IsTerminal() bool {
	return s[0] != '<' || s == "<nil>" || s[: 2] == "<?"
}

// Text return the text in Symbol, the text should be [_A-Za-z0-9] only, like
//     <city-name> -> "city_name"
//     <?time_s0> -> "time_s0"
//     weather -> "weather"
//     上海 -> "_"
func (s Symbol) Text() string {
	text := string(s)
	if len(text) > 1 && text[: 2] == "<?" {
		text = text[2: len(text) - 1]
	} else if text[0] == '<' {
		text = text[1: len(text) - 1]
	}
	return regexp.MustCompile("[^_A-Za-z0-9]+").ReplaceAllString(text, "_")
}

// Rule represents a PCFG rule
type Rule struct {
	Left Symbol
	Right []Symbol
	Weight float64

	// Path is the derive path from right symbols to left symbols
	// It will have values only after some post-processing steps
	// For example, after PCFG to CNF, rule A->B, B->C, C->DE will merged into
	// a single rule A->DE and the path is (B C)
	Path []Symbol
}

// IsBinary returns true if it's a binary rule, like A -> BC
func (r *Rule) IsBinary() bool {
	return len(r.Right) == 2
}

// IsUnary returns true if it's a unary rule, like A -> B
func (r *Rule) IsUnary() bool {
	return len(r.Right) == 1
}

// ParseRule parse rule from string
// The rule would be like:
//     <weather-1> ::= "weather" "in" <city-name>, 0.7 | <city-name> weather, 0.3
// Then returns
//     [{"<weather-1>", ["weather", "in", "<city-name>"], 0.7},
//      {"<weather-1>", ["<city-name>", "weather"], 0.3}]
func ParseRule(ruleText string) (rules []*Rule, err error) {
	rules = make([]*Rule, 0)
	fields := strings.Split(ruleText, "::=")
	if len(fields) != 2 {
		err = errors.New(fmt.Sprintf("ParseRule: unexpected number of ::= token in '%s'", ruleText))
		return
	}

    // Left part
	leftSymbol := Symbol(strings.TrimSpace(fields[0]))
	if leftSymbol.IsTerminal() {
		err = errors.New(fmt.Sprintf("ParseRule: '%s': terminal symbol in the left", ruleText))
		return
	}

    // Right part
	for _, right := range strings.Split(fields[1], "|") {
		rule := new(Rule)
		rule.Left = leftSymbol

		right = strings.TrimSpace(right)
		fields := strings.Split(right, ";")
		if len(fields) == 2 {
			// Has the weight value, parse it
			weightText := strings.TrimSpace(fields[1])
			if rule.Weight, err = strconv.ParseFloat(weightText, 64); err != nil {
				err = errors.New(fmt.Sprintf(
					"ParseRule: float expected but '%s' found in '%s'",
					weightText,
					ruleText))
				return
			}
		} else if len(fields) == 1 {
			rule.Weight = 1.0
		} else {
			err = errors.New(fmt.Sprintf("ParseRule: unexpected ';' token in '%s'", ruleText))
			return
		}

		// Tokens of this rule
		rule.Right = make([]Symbol, 0)
		for _, symbolString := range strings.Fields(fields[0]) {
			symbol := Symbol(symbolString)
			if !symbol.IsValid() {
				err = errors.New(fmt.Sprintf("ParseRule: unexpected '%s' in '%s'", symbolString, ruleText))
				return
			}
			rule.Right = append(rule.Right, Symbol(symbolString))
		}

		rules = append(rules, rule)
	}

	return
}

// String converts rule to string format
func (r *Rule) String() string {
	symbols := []string{}
	for _, symbol := range r.Right {
		symbols = append(symbols, string(symbol))
	}
	s := fmt.Sprintf(
		"%s ::= %s ; %.3f",
		string(r.Left),
		strings.Join(symbols, " "),
		r.Weight)
	if r.Path != nil {
		symbols = []string{}
		for _, symbol := range r.Path {
			symbols = append(symbols, string(symbol))
		}
		s += fmt.Sprintf(" (%s)", strings.Join(symbols, " "))
	}
	return s
}
