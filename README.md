# PCFG

ling0322/pcfg is a simple implementation of PCFG parser in Go. And it is designed and optimized for parsing with hand-written PCFG rules. Didn't like generic purpose PCFG parser such as Stanford Parser, it focus on string/slot matching.

## Grammar

Definition of PCFG grammar for our parser is like

    <source> ::= <target1> <target2> ; probability 
    
Symbols wrap with `<>` are non-terminal symbols, otherwise, it's terminal symbols. "probability" is the probability of this rule. Two rules with the same source symbol could be merged into one rule with "|" like

    <weather> ::= <city> weather ; 0.3
    <weather> ::= weather <city> ; 0.7

Equal to

    <weather> ::= <city> weather ; 0.3 | weather <city> ; 0.7

### Special Symbols

There are also some special symbols in grammar:

- `<root>`: root node of grammar
- `<nil>`: A black symbol, like epsilon in most books

### Comments

Grammar could be commented using ";", for example
    
    <weather> ::= weather <city> ; 0.7
    ; This is a comment

### Export Symbols

Symbols could be exported using `;!exports:` statement. Only exported rules could be seen in parsing tree.

    ;!exports: <export_symbol1> <export_symbol2>


### Example

Here is an example grammar that matches queries like "what's the weather in seattle", "weather in beijing"

```
<city> ::= seattle | beijing
<whats> ::= what's the | <nil>
<root> ::= <whats> weather in <city>

;We only interested in <city>
;!exports: <city>
```

## Programming

The PCFG parser could be created by

```go
func pcfg.NewParser(pcfgGrammar string) (*pcfg.Parser, error)
```

Then parse queries using

```go
func (p *Parser) Parse(query []string) *Tree
```

The Parse function returns parsing tree if successfully matched, otherwise returns nil

For example

```go
package main

import (
	"github.com/ling0322/pcfg"
	"fmt"
	"log"
	"strings"
)

func main() {
	grammarText := `
		<city> ::= seattle | beijing
		<whats> ::= what's the | <nil>
		<root> ::= <whats> weather in <city>

		; We only interested in <city>
		;!exports: <city>`

	parser, err := pcfg.NewParser(grammarText)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(parser.Parse(strings.Fields("what's the weather in seattle")))
	fmt.Println("")
	fmt.Println(parser.Parse(strings.Fields("weather in beijing")))
	fmt.Println("")
	fmt.Println(parser.Parse(strings.Fields("seattle weather")))
}
```

Outputs

```
(<root> 
  what's 
  the 
  weather 
  in 
  (<city> 
    seattle))

(<root> 
  weather 
  in 
  (<city> 
    beijing))

<nil>
```

Another example

```go
package main

import (
	"github.com/ling0322/pcfg"
	"fmt"
	"log"
	"strings"
)

func parseQuery(parser *pcfg.Parser, query string) {
	words := strings.Fields(query)
	fmt.Println(query)
	fmt.Println(parser.Parse(words))
	fmt.Println("")
}

func main() {
	grammarText := `
		<city> ::= seattle | beijing
		<time> ::= today | tomorrow
		<the> ::= the | <nil>
		<in> ::= in | <nil>
		<r1p1> ::= what's | what is | <nil>
		<r1p2> ::= <the> weather
		<r1p3> ::= like | going to be like | <nil>
		<r1p4> ::= <in> <city>
		<r1p5> ::= <time> | <nil>
		<r1> ::= <r1p1> <r1p2> <r1p3> <r1p4> <r1p5>
		<r2p1> ::= <city> weather
		<r2p2> ::= <time> | <nil>
		<r2> ::= <r2p1> <r2p2>
		<root> ::= <r1> | <r2>
		;!exports: <city> <time>`

	parser, err := pcfg.NewParser(grammarText)
	if err != nil {
		log.Fatal(err)
	}

	queries := []string{
		"what's the weather in beijing tomorrow",
		"beijing weather",
		"seattle weather tomorrow",
		"what is the weather going to be like in seattle",
		"weather in seattle",
		"weather seattle tomorrow",
		"the weather in seattle",
		"what's weather seattle",
	}

	for _, query := range queries {
		parseQuery(parser, query)
	}
}
```

Outputs

```
what's the weather in beijing tomorrow
(<root> 
  what's 
  the 
  weather 
  in 
  (<city> 
    beijing) 
  (<time> 
    tomorrow))

beijing weather
(<root> 
  (<city> 
    beijing) 
  weather)

seattle weather tomorrow
(<root> 
  (<city> 
    seattle) 
  weather 
  (<time> 
    tomorrow))

what is the weather going to be like in seattle
(<root> 
  what 
  is 
  the 
  weather 
  going 
  to 
  be 
  like 
  in 
  (<city> 
    seattle))

weather in seattle
(<root> 
  weather 
  in 
  (<city> 
    seattle))

weather seattle tomorrow
(<root> 
  weather 
  (<city> 
    seattle) 
  (<time> 
    tomorrow))

the weather in seattle
(<root> 
  the 
  weather 
  in 
  (<city> 
    seattle))

what's weather seattle
(<root> 
  what's 
  weather 
  (<city> 
    seattle))
```
