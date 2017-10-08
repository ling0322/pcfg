package pcfg

// CNFRuleBase is the base struct for CNFRule and CNFTerminalRule
type CNFRuleBase struct {
	// SymbolId in the left of rule
	Source int

	// Probability of this rule
	Probability float64

	// Path of symbolIds from source to target
	Path []int
}

// CNFRule stores a non-terminal rule in CNF grammar. All of the symbols in this
// rule arerepresented by symbol-id
type CNFRule struct {
	CNFRuleBase

	// SymbolIds in the right of rule
	FirstTarget int
	SecondTarget int
}

// CNFTerminalRule stores the terminal rule in the grammar
type CNFTerminalRule struct {
	CNFRuleBase

	// Terminal symbol in this rule
	TerminalTarget string
}

// CNFGrammar stores the grammar in Chomsky normal form
type CNFGrammar struct {
	// Map from symbol name to its id
	SymbolIds map[string]int

	// Map from symbolId to symbol name
	Symbols []string

	// Map from terminal string to symbolId
	TerminalRules map[string][]*CNFTerminalRule

	// Map from targets to rule. For example, rule: A -> BC. It maps (B, C) to
	// the rule itself
	Rules map[int]map[int][]*CNFRule

	// Nonterminal symbols that exports to parsing tree
	Exports map[int]bool
}

// NewCNFGrammar creates a new instance of CNFGrammar
func NewCNFGrammar() *CNFGrammar {
	return &CNFGrammar{
		SymbolIds: map[string]int{},
		Symbols: []string{},
		Rules: map[int]map[int][]*CNFRule{},
		TerminalRules: map[string][]*CNFTerminalRule{},
		Exports: map[int]bool{},
	}
}

// getSymbolId get the id of given symbol. If the symbol not exist in grammar
// insert a new one
func (g *CNFGrammar) getSymbolId(s Symbol) int {
	if symbolId, ok := g.SymbolIds[string(s)]; ok {
		return symbolId
	}
	symbolId := len(g.Symbols)
	g.SymbolIds[string(s)] = symbolId
	g.Symbols = append(g.Symbols, string(s))
	return symbolId
}

// AddRule adds an export symbol to grammar
func (g *CNFGrammar) AddExportSymbol(s Symbol) {
	symbolId := g.getSymbolId(s)
	g.Exports[symbolId] = true
}

// Add a new rule into grammar
func (g *CNFGrammar) AddRule(rule *Rule) {
	assert(
		rule.IsBinary() || (rule.IsUnary() && rule.Right[0].IsTerminal()),
		"CNFGrammar::AddRule: invalid rule")
	assert(
		rule.IsUnary() || !rule.Right[0].IsTerminal() && !rule.Right[1].IsTerminal(),
		"CNFGrammar::AddRule: invalid rule")

	// convertPath converts a symbol-based path slice to int-based
	convertPath := func (path []Symbol) []int {
		intPath := []int{}
		for _, s := range path {
			intPath = append(intPath, g.getSymbolId(s))
		}
		return intPath
	}

	if rule.IsUnary() {
		// It's a terminal rule, like <weather> ::= weather
		sourceId := g.getSymbolId(rule.Left)
		terminalSymbol := string(rule.Right[0])
		if _, ok := g.TerminalRules[terminalSymbol]; !ok {
			g.TerminalRules[terminalSymbol] = []*CNFTerminalRule{}
		}
		cnfRule := &CNFTerminalRule{
			CNFRuleBase: CNFRuleBase{
				Source: sourceId,
				Probability: rule.Weight,
				Path: convertPath(rule.Path),
			},
			TerminalTarget: terminalSymbol,
		}
		g.TerminalRules[terminalSymbol] = append(
			g.TerminalRules[terminalSymbol],
			cnfRule)
	} else {
		sourceId := g.getSymbolId(rule.Left)
		firstTargetId := g.getSymbolId(rule.Right[0])
		secondTargetId := g.getSymbolId(rule.Right[1])

		cnfRule := &CNFRule{
			CNFRuleBase: CNFRuleBase{
				Source: sourceId,
				Probability: rule.Weight,
				Path: convertPath(rule.Path),
			},
			FirstTarget: firstTargetId,
			SecondTarget: secondTargetId,
		}

		if _, ok := g.Rules[firstTargetId]; !ok {
			g.Rules[firstTargetId] = map[int][]*CNFRule{}
		}
		if _, ok := g.Rules[firstTargetId][secondTargetId]; !ok {
			g.Rules[firstTargetId][secondTargetId] = []*CNFRule{}
		}
		g.Rules[firstTargetId][secondTargetId] = append(
			g.Rules[firstTargetId][secondTargetId],
			cnfRule)
	}
}

