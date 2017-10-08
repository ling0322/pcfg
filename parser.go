package pcfg

// Parser is the struct for PCFG parsing
type Parser struct {
	grammar *Grammar
	cnfGrammar *CNFGrammar
}

// If enable debug model when converting grammar or parsing
var gEnableDebug bool

// NewParser creates a new instance of PCFG parser with pcfgGrammar
func NewParser(pcfgGrammar string) (parser *Parser, err error) {
	parser = new(Parser)
	parser.grammar, err = ParseGrammar(pcfgGrammar)
	if err != nil {
		return nil, err
	}

	parser.cnfGrammar = parser.grammar.ConvertToCNF()
	return
}

// Enable debug model
func DebugMode() {
	gEnableDebug = true
}

// Parse parses query using the PCFG grammar. If query matches the grammar,
// returns the parsing tree. Otherwise, return nil
func (p *Parser) Parse(query []string) *Tree {
	return CYK(p.cnfGrammar, query)
}
