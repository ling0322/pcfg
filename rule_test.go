package pcfg

import (
	"testing"
)

func TestParseRule(t *testing.T) {
	// TestCase-1
	r, err := ParseRule("<weather-1> ::= weather in <city-name>")
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 {
		t.Fatal("len(r) == 1")
	}

	expected := "<weather-1> ::= weather in <city-name> ; 1.000"
	if r[0].String() != expected {
		t.Fatalf("'%s' != '%s'", r[0].String(), expected)
	}

	// TestCase-2
	r, err = ParseRule("<weather-2> ::= weather in <city-name>|<city-name> weather;0.3")
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 2 {
		t.Fatal("len(r) == 2")
	}

	expected = "<weather-2> ::= weather in <city-name> ; 1.000"
	if r[0].String() != expected {
		t.Fatalf("'%s' != '%s'", r[0].String(), expected)
	}
	expected = "<weather-2> ::= <city-name> weather ; 0.300"
	if r[1].String() != expected {
		t.Fatalf("'%s' != '%s'", r[0].String(), expected)
	}

	// TestCase-3: failed case
	_, err = ParseRule("<weather-2> ::= <city-name weather;0.3")
	if err == nil {
		t.Fatal("err != nil expected")
	}

	// TestCase-4: failed case
	_, err = ParseRule("<weather-2> ::= city-name weather;0.3")
	if err == nil {
		t.Fatal("err != nil expected")
	}

	// TestCase-5: failed case
	_, err = ParseRule("weather_f ::= <city-name> weather;0.3")
	if err == nil {
		t.Fatal("err != nil expected")
	}
}
