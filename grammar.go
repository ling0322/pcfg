package pcfg

import (
	"fmt"
	"strings"
	"github.com/pkg/errors"
	"math"
	"log"
)

// Grammar consists a list of PCFG rules
type Grammar struct {
	Rules []*Rule
	Exports map[Symbol]bool
	isDebug bool
}

//
// Here are the functions that used to convert PCFG to CNF
// According to paper: http://www.cs.nyu.edu/courses/fall07/V22.0453-001/cnf.pdf
//

// ParseGrammar parses grammar from string
func ParseGrammar(grammarText string) (grammar *Grammar, err error) {
	grammar = &Grammar{
		Rules: []*Rule{},
		Exports: map[Symbol]bool{},
	}
	lines := strings.Split(grammarText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Exports command
		if strings.Index(line, ";!exports:") == 0 {
			exports := strings.Fields(line[len(";!exports:"):])
			for _, export:= range exports {
				symbol := Symbol(strings.TrimSpace(export))
				if symbol.IsTerminal() || !symbol.IsValid() {
					err = errors.New(fmt.Sprintf(
						"ParseGrammar: unexpected export symbol: %s",
						symbol))
					return nil, err
				}
				grammar.Exports[symbol] = true
			}
		}

		// Comments
		if line == "" || line[0] == ';' {
			continue
		}

		// Parse this rule
		rule, err := ParseRule(line)
		if err != nil {
			return grammar, err
		}
		grammar.Rules = append(grammar.Rules, rule...)
	}
	return
}

// Enable debug in grammar, it will print some debug information
func (g *Grammar) DebugMode() {
	g.isDebug = true
}

// Print grammar
func (g *Grammar) Print() {
	for _, rule := range g.Rules {
		fmt.Println(rule.String())
	}
	fmt.Println("")
}

// ConvertToCNF converts CFG grammar to CNF (Debug mode)
func (g *Grammar) ConvertToCNF() *CNFGrammar {
	if gEnableDebug {
		fmt.Println("======= Original Grammar =======")
	}
	g.normalizeWeight()
	if gEnableDebug {
		g.Print()
		fmt.Println("======= Add Term Variables =======")
	}
	g.addTermVariables()
	if gEnableDebug {
		g.Print()
		fmt.Println("======= Reduce Higher Rules =======")
	}
	g.reduceHigherRules()
	if gEnableDebug {
		g.Print()
		fmt.Println("======= Remove Null Rules =======")
	}
	g.removeNullRules()
	if gEnableDebug {
		g.Print()
		fmt.Println("======= Remove Strong Components =======")
	}
	g.removeStrongComponents()
	if gEnableDebug {
		g.Print()
		fmt.Println("======= Remove Unit Rules =======")
	}
	g.removeUnitRules()
	if gEnableDebug {
		g.Print()
	}

	cnfGrammar := NewCNFGrammar()
	for _, rule := range g.Rules {
		cnfGrammar.AddRule(rule)
	}

	for export := range g.Exports {
		cnfGrammar.AddExportSymbol(export)
	}

	return cnfGrammar
}

// normalizeWeight normalize the weight of rule. Make sure that the sum of weight
// from the same source symbol is 1.0
func (g *Grammar) normalizeWeight() {
	weights := map[Symbol]float64{}
	for _, rule := range g.Rules {
		if _, ok := weights[rule.Left]; !ok {
			weights[rule.Left] = 0.0
		}
		weights[rule.Left] += rule.Weight
	}
	for _, rule := range g.Rules {
		rule.Weight /= weights[rule.Left]
	}
}

// addTermVariables eliminiates terminal symbols except in right hand sides of size 1
func (g *Grammar) addTermVariables() {
	termRulesCount := 0
	terminalSymbols := map[Symbol]Symbol{}
	for _, rule := range g.Rules {
		if rule.IsUnary() {
			// Expect in right hand sides of size 1
			continue
		}
		for i, symbol := range rule.Right {
			if symbol.IsTerminal() {
				nonTerminalSymbol, ok := terminalSymbols[symbol]
				if !ok {
					// Add the corresponded non-terminal symbol if not exist
					nonTerminalSymbol = InternalSymbol(
						fmt.Sprintf("t_%s_%d", symbol.Text(), termRulesCount))
					terminalSymbols[symbol] = nonTerminalSymbol
				}
				rule.Right[i] = nonTerminalSymbol
				termRulesCount++
			}
		}
	}

	// Add each nonTerminalSymbol -> symbol rule
	for symbol, nonTerminalSymbol := range terminalSymbols {
		rule := &Rule{
			Left: nonTerminalSymbol,
			Right: []Symbol{symbol},
			Weight: 1.0}
		g.Rules = append(g.Rules, rule)
	}
}

// reduceHigherRules converts rule with right-hand size larger than 2 into a set
// of binary rules
func (g *Grammar) reduceHigherRules() {
	binaryRules := []*Rule{}
	for _, rule := range g.Rules {
		if rule.IsUnary() || rule.IsBinary() {
			// It's already binary rule
			binaryRules = append(binaryRules, rule)
		} else {
			ruleText := rule.Left.Text()
			count := 1

			// Begin rule: U -> W_1 X_0
			// It's the reference to next rule, so didn't increase count here
			x0 := InternalSymbol(fmt.Sprintf("x_%s_%d", ruleText, count))
			r := &Rule{
				Left: rule.Left,
				Right: []Symbol{rule.Right[0], x0},
				Weight: rule.Weight}
			binaryRules = append(binaryRules, r)

			// Middle rules: X_i-1 -> W_i X_i
			for i := 1; i < len(rule.Right) - 2; i++ {
				x := InternalSymbol(fmt.Sprintf("x_%s_%d", ruleText, count))
				nextX := InternalSymbol(fmt.Sprintf("x_%s_%d", ruleText, count + 1))
				count++
				r := &Rule{
					Left: x,
					Right: []Symbol{rule.Right[i], nextX},
					Weight: 1.0}
				binaryRules = append(binaryRules, r)
			}

			// End rule: X_k-1 = W_k-1 W_k
			x := InternalSymbol(fmt.Sprintf("x_%s_%d", ruleText, count))
			count++
			k := len(rule.Right) - 1;
			r = &Rule{
				Left: x,
				Right: []Symbol{rule.Right[k - 1], rule.Right[k]},
				Weight: 1.0}
			binaryRules = append(binaryRules, r)
		}
	}
	g.Rules = binaryRules
}

// Gets occurs-right map, that records which rules does a symbol occurs in the
// right side. assuming all rules are unary or binary
func (g *Grammar) occursRight() map[Symbol][]*Rule {
	occurs := map[Symbol][]*Rule{}
	for _, rule := range g.Rules {
		if rule.IsBinary() {
			// Rule: A -> BC
			B := rule.Right[0]
			C := rule.Right[1]
			if _, ok := occurs[B]; !ok {
				occurs[B] = []*Rule{}
			}
			if _, ok := occurs[C]; !ok {
				occurs[C] = []*Rule{}
			}
			occurs[B] = append(occurs[B], rule)
			occurs[C] = append(occurs[C], rule)
		} else if rule.IsUnary() && !rule.Right[0].IsTerminal() {
			// Rule: A -> B
			B := rule.Right[0]
			if _, ok := occurs[B]; !ok {
				occurs[B] = []*Rule{}
			}
			occurs[B] = append(occurs[B], rule)
		}
	}
	return occurs
}

// Gets occurs-left map. For every rule r: A -> BC, add occursLeft[A] = r
func (g *Grammar) occursLeft() map[Symbol][]*Rule {
	occurs := map[Symbol][]*Rule{}
	for _, rule := range g.Rules {
		if occurs[rule.Left] == nil {
			occurs[rule.Left] = []*Rule{rule}
		} else {
			occurs[rule.Left] = append(occurs[rule.Left], rule)
		}
	}
	return occurs
}

// findNullables finds nullable symbols and its probabilities from grammar
func (g *Grammar) findNullables() map[Symbol]float64 {
	occurs := g.occursRight()
	nullable := map[Symbol]float64{}
	todo := []Symbol{}

	// nullable, todo
	for _, rule := range g.Rules {
		if rule.IsUnary() && rule.Right[0] == EpsilonSymbol {
			// Rule: A -> <nil>
			nullable[rule.Left] = rule.Weight
			todo = append(todo, rule.Left)
		}
	}

	processed := map[*Rule]bool{}
	for len(todo) != 0 {
		var B Symbol
		B, todo = todo[0], todo[1: ]
		for _, rule := range occurs[B] {
			if processed[rule] {
				continue
			}

			nullProb := rule.Weight
			for _, symbol := range rule.Right {
				nullProb *= nullable[symbol]
			}
			if nullProb > 0 {
				// Ok, this rule may be null
				nullable[rule.Left] += nullProb
				processed[rule] = true
				todo = append(todo, rule.Left)
			}
		}
	}

	return nullable
}

// removeNullables remove null rules (A -> <nil>) from grammar
func (g *Grammar) removeNullRules() {
	nullables := g.findNullables()

	// Unary rules
	singleRules := map[[2]Symbol]*Rule{}
	for _, rule := range g.Rules {
		if rule.IsUnary() {
			singleRules[[2]Symbol{rule.Left, rule.Right[0]}] = rule
		}
	}

	// For rule A -> BC, if B is nullable, add new rule A -> C
	type ruleToAdd struct {
		A, B Symbol
		Probability float64
	}
	rulesToAdd := []ruleToAdd{}
	for _, rule := range g.Rules {
		if !rule.IsBinary() {
			continue
		}

		A := rule.Left
		B := rule.Right[0]
		C := rule.Right[1]
		probability := rule.Weight
		if nullables[B] > 0 {
			ruleProb := probability * nullables[B]
			rulesToAdd = append(rulesToAdd, ruleToAdd{A, C, ruleProb})
			rule.Weight -= ruleProb
		}
		if nullables[C] > 0 {
			ruleProb := probability * nullables[C]
			rulesToAdd = append(rulesToAdd, ruleToAdd{A, B, ruleProb})
			rule.Weight -= ruleProb
		}
	}

	// Add rules in rulesToAdd
	for _, rule := range rulesToAdd {
		if targetRule, ok := singleRules[[2]Symbol{rule.A, rule.B}]; ok {
			// If A -> B already exists
			targetRule.Weight += rule.Probability
		} else {
			g.Rules = append(g.Rules, &Rule{
				Left: rule.A,
				Right: []Symbol{rule.B},
				Weight: rule.Probability})
		}
	}

	// Remove empty rules like A -> <nil>
	rules := []*Rule{}
	for _, rule := range g.Rules {
		if !(rule.IsUnary() && rule.Right[0] == EpsilonSymbol) {
			rules = append(rules, rule)
		}
	}
	g.Rules = rules

	// Normalize probabilities after empty rules removed
	// Only influences directly nullables symbols like A with A -> <nil>
	g.normalizeWeight();
}

// replaceStrongComponents replaces strong component with a single symbol/vertex.
// Then stores such replacement into g.Alternatives
func (g *Grammar) findStrongComponents() [][]Symbol {
	// Find each strong component with Kosaraju's algorithm
	// Here strong component will only occur in unary rules like A -> B
	graph := NewDirectedGraph()
	for _, rule := range g.Rules {
		if rule.IsUnary() && !rule.Right[0].IsTerminal() {
			graph.Add(Vertex(rule.Left), Vertex(rule.Right[0]), rule.Weight)
		}
	}

	components := graph.StrongComponents()
	symbolComps := [][]Symbol{}
	for _, c := range components {
		symbolComp := []Symbol{}
		for _, v := range c {
			symbolComp = append(symbolComp, Symbol(v))
		}
		symbolComps = append(symbolComps, symbolComp)
	}
	return symbolComps
}

// removeStrongComponent removes a strong component from graph
func (g *Grammar) removeStrongComponent(strongComponent []Symbol) {
	graph := NewDirectedGraph()
	occursLeft := g.occursLeft()
	occursRight := g.occursRight()

	component := map[Symbol]bool{}
	for _, s := range strongComponent {
		component[s] = true
	}

	// Construct the strong connected graph to compute shortest path
	for _, rule := range g.Rules {
		if component[rule.Left] && rule.IsUnary() {
			if component[rule.Right[0]] {
				// -math.Log(): Some tricks to apply shortPath in probability
				graph.Add(Vertex(rule.Left), Vertex(rule.Right[0]), -math.Log(rule.Weight))
			}
		}
	}
	distance := graph.Floyd()
	transProbs := map[Symbol]map[Symbol]float64{}
	for s, ts := range distance {
		for t, negativeLogP := range ts {
			if _, ok := transProbs[Symbol(s)]; !ok {
				transProbs[Symbol(s)] = map[Symbol]float64{}
			}
			transProbs[Symbol(s)][Symbol(t)] = math.Exp(-negativeLogP)
		}
	}

	// Symbols only referenced inside the component
	internals := map[Symbol]bool{}

	// For symbols S, T in components. if P(S->T) = 0.2 after floyd algorithm,
	// and "T -> BC; 0.4". Then add rule "S -> BC; innerProb*0.2*0.4"
	for symbol, _ := range component {
		// Ignore this symbol if it is only referenced inside the strong
		// connected component
		isExternal := false
		for _, rule := range occursRight[symbol] {
			if rule.IsBinary() || !component[rule.Left] {
				isExternal = true
				break
			}
		}
		if !isExternal {
			internals[symbol] = true
			continue
		}

		// innerProb is the probability that symbol transfer into its strong
		// connected components
		innerProb := 0.0
		for _, rule := range occursLeft[symbol] {
			if rule.IsUnary() && component[rule.Right[0]] {
				innerProb += rule.Weight
			}
		}
		for targetSymbol, _ := range component {
			if symbol == targetSymbol {
				// Don't replace anything with the symbol itself
				continue
			}
			for _, targetRule := range occursLeft[targetSymbol] {
				if targetRule.IsUnary() && component[targetRule.Right[0]] {
					// Ignore the rules of this component
					continue
				}
				transProb := transProbs[symbol][targetSymbol]
				g.Rules = append(g.Rules, &Rule{
					Left: symbol,
					Right: targetRule.Right,
					Weight: innerProb * transProb * targetRule.Weight})
			}
		}
	}

	// Remove useless rules in this strong component, including
	//   - Strong connected rules, like A -> C in strong component [A, B, C]
	//   - Unreferenced rules outside the component
	rules := []*Rule{}
	for _, rule := range g.Rules {
		if rule.IsUnary() && component[rule.Left] && component[rule.Right[0]] {
			continue
		}
		if internals[rule.Left] {
			continue
		}
		rules = append(rules, rule)
	}
	g.Rules = rules
}

// removeStrongComponents removes all strong components from graph
func (g *Grammar) removeStrongComponents() {
	components := g.findStrongComponents()
	for _, component := range components {
		g.removeStrongComponent(component)
	}

	// Remove rules like X -> X
	rules := []*Rule{}
	for _, rule := range g.Rules {
		if rule.IsUnary() && rule.Left == rule.Right[0] {
			continue
		}
		rules = append(rules, rule)
	}
	g.Rules = rules
	g.normalizeWeight()
}

// Remove one unit rule (left -> right) from grammar
func (g *Grammar) removeUnitRule(left, right Symbol) {
	occursLeft := g.occursLeft()
	occursRight := g.occursRight()

	// Find rule: left -> right
	weight := 0.0
	for _, rule := range occursLeft[left] {
		if rule.IsUnary() && rule.Right[0] == right {
			weight = rule.Weight
			break
		}
	}

	// For any rule like "right -> BC; pr", add rule "left -> BC; weight * pr"
	for _, rule := range occursLeft[right] {
		path := []Symbol{right}
		if rule.Path != nil {
			path = append(path, rule.Path...)
		}
		g.Rules = append(g.Rules, &Rule{
			Left: left,
			Right: rule.Right,
			Weight: rule.Weight * weight,
			Path: path})
	}

	// Checks if right is only referenced by left
	isRightUseless := len(occursRight[right]) == 1

	// Remove rule left -> right. If isRightUseless == true, remove rules like
	// right -> ..
	rules := []*Rule{}
	for _, rule := range g.Rules {
		// Remove the rule: left -> right
		if rule.IsUnary() && rule.Left == left && rule.Right[0] == right {
			continue
		}

		// Remove rules: right -> ... when needed
		if isRightUseless && rule.Left == right {
			continue
		}
		rules = append(rules, rule)
	}
	g.Rules = rules
}

// removeUnitRules removes unit rules like A -> B, B -> C
func (g *Grammar) removeUnitRules() {
	// Get unit rules in reversed topological order
	for {
		graph := NewDirectedGraph()
		hasUnaryRule := false
		for _, rule := range g.Rules {
			if rule.IsUnary() && !rule.Right[0].IsTerminal() {
				graph.Add(Vertex(rule.Left), Vertex(rule.Right[0]), rule.Weight)
				hasUnaryRule = true
			}
		}
		if !hasUnaryRule {
			break
		}

		// Find a leaf rule
		graphT := graph.Transpose()
		leafVertex := graphT.TopologicalSort()[0]
		visited := map[Vertex]bool{}
		leafRules := graphT.DFS(leafVertex, visited)

		left := leafRules[1]
		right := leafRules[0]
		if graph.HasArc(left, right) {
			// Rule: left -> right exists
			if g.isDebug {
				log.Printf("removeUnitRule: %s ::= %s\n", left, right)
			}
			g.removeUnitRule(Symbol(left), Symbol(right))
		}
	}
}
