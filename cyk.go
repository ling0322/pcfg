package pcfg

import (
	"math"
	"fmt"
	"strings"
)

// cykNode is the node used in CKY table
type _CYKNode struct {
	symbol int
	rule *CNFRuleBase
	logp float64

	left *_CYKNode
	right *_CYKNode
	next *_CYKNode
}

// nodePool is the pool that allocatesand stores _CYKNode
const _PoolBatchSize = 4096
type _NodePool struct {
	nodes [][]_CYKNode
	row int
	column int
}

// newNodePool create a new instance of _NodePool
func newNodePool() *_NodePool {
	pool := &_NodePool{
		nodes: [][]_CYKNode{make([]_CYKNode, _PoolBatchSize)},
		row: 0,
		column: 0,
	}
	return pool
}

// Get allocates a new _CYKNode from pool
func (pool *_NodePool) Get() *_CYKNode {
	var node *_CYKNode
	node = &pool.nodes[pool.row][pool.column]

	pool.column++
	if pool.column >= _PoolBatchSize {
		pool.nodes = append(pool.nodes, make([]_CYKNode, _PoolBatchSize))
		pool.row++
		pool.column = 0
	}
	return node
}


func constructParsingTree(grammar *CNFGrammar, node *_CYKNode, query []string) []*Node {
	// When it's a leaf node (terminal node, row = 0)
	if node.symbol < 0 {
		treeNode := &Node{Symbol: query[-node.symbol - 1]}
		return []*Node{treeNode}
	}

	// Get nodes of its children
	leftNodes := constructParsingTree(grammar, node.left, query)

	// For some nodes node.right may be nil
	rightNodes := []*Node{}
	if node.right != nil {
		rightNodes = constructParsingTree(grammar, node.right, query)
	}

	treeNodes := append(leftNodes, rightNodes...)

	// Handle the path from target to source
	if node.rule.Path != nil {
		// We are constructing the tree bottom-up, the path should be process
		// in reversed order
		for i := len(node.rule.Path) - 1; i >= 0; i-- {
			symbol := node.rule.Path[i]
			if grammar.Exports[symbol] {
				treeNode := &Node{
					Children: treeNodes,
					Symbol: grammar.Symbols[symbol],
				}
				treeNodes = []*Node{treeNode}
			}
		}
	}

	// Handle the node itself
	if grammar.Exports[node.symbol] ||
		grammar.Symbols[node.symbol] == string(RootSymbol) {
		treeNode := &Node{
			Children: treeNodes,
			Symbol: grammar.Symbols[node.symbol],
		}
		treeNodes = []*Node{treeNode}
	}

	return treeNodes
}

// printRow prints a row in CYK table for debugging
func printRow(grammar *CNFGrammar, row []*_CYKNode) {
	for i, node := range row {
		nodeReprs := []string{}
		for node != nil {
			nodeReprs = append(nodeReprs, grammar.Symbols[node.symbol])
			node = node.next
		}
		fmt.Printf("[%d: %s] ", i, strings.Join(nodeReprs, " "))
	}
	fmt.Println("")
}

// CYK parses query using CKY algorithm. When query matches grammae, returns the
// parsing tree. Otherwise returns nil
func CYK(grammar *CNFGrammar, query []string) *Tree {
	if gEnableDebug {
		fmt.Println("======= CYK algorithm =======")
	}
	table := [][]*_CYKNode{}
	pool := newNodePool()

	// Row 0: dummy node for terminal symbols
	table = append(table, make([]*_CYKNode, len(query)))
	for i := range query {
		// For leaf nodes, symbol stores the in query with negative number
		table[0][i] = &_CYKNode{symbol: -i - 1}
	}

	// Row 1: apply all terminla rules
	table = append(table, make([]*_CYKNode, len(query)))
	for i, tok := range query {
		if rules, ok := grammar.TerminalRules[tok]; ok {
			var nodes *_CYKNode
			for _, rule := range rules {
				node := pool.Get()
				node.symbol = rule.Source
				node.rule = &rule.CNFRuleBase
				node.logp = math.Log(rule.Probability)
				node.left = table[0][i]
				node.next = nodes

				// Insert into the head of linklist
				nodes = node
			}
			table[1][i] = nodes
		}
	}
	if gEnableDebug {
		printRow(grammar, table[1])
	}


	// Row 2 to row n: apply non-terminal rules
	// TODO: early stop
	// Length of span
	for length := 2; length <= len(query); length++ {
		columns := len(query) - length + 1
		table = append(table, make([]*_CYKNode, columns))
		// Start of span
		for start := 0; start < columns; start++ {
			// Partition of span
			for partition := 1; partition < length; partition++ {
				left := table[partition][start]
				for left != nil {
					rightRules, ok := grammar.Rules[left.symbol]
					right := table[length - partition][start + partition]
					for ok && right != nil {
						if rules, ok := rightRules[right.symbol]; ok {
							// Ok, there are some rules A -> BC that B == first
							// and C == second
							nodes := table[length][start]
							for _, rule := range rules {
								logp := math.Log(rule.Probability) + left.logp + right.logp
								node := pool.Get()
								node.symbol = rule.Source
								node.left = left
								node.right = right
								node.next = nodes
								node.rule = &rule.CNFRuleBase
								node.logp = logp

								nodes = node
							}
							table[length][start] = nodes
						}
						right = right.next
					}

					left = left.next
				}
			}
		}
		if gEnableDebug {
			printRow(grammar, table[len(table) - 1])
		}
	}

	// Find the best root node and construct the parsing tree
	rootSymbol := grammar.SymbolIds[string(RootSymbol)]
	node := table[len(query)][0]
	maxLogProb := math.Inf(-1)
	var root *_CYKNode
	for node != nil {
		if node.symbol == rootSymbol && node.logp > maxLogProb {
			maxLogProb = node.logp
			root = node
		}
		node = node.next
	}
	if root == nil {
		// root == nil means query didn't match grammar
		return nil
	}


	nodes := constructParsingTree(grammar, root, query)
	return &Tree{
		Node: nodes[0],
	}
}
