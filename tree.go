package pcfg

import (
	"strings"
	"fmt"
)

// Node represents a single node in parsing tree
type Node struct {
	// Children nodes
	Children []*Node

	// Symbol in current node
	Symbol string
}

// Tree represents the parsing tree
type Tree struct {
	*Node
}



// Convert the node to string
func (n *Node) String() string {
	return n.repr(0)
}

// Repr get the string representation of the ndoe recursively
func (n *Node) repr(level int) string {
	// Don't wrap with parentheses when it's a leaf node
	prefix := strings.Repeat(" ", level * 2)
	if level != 0 {
		prefix = "\n" + prefix
	}

	if n.Children == nil {
		return prefix + n.Symbol
	} else {
		childrenReprs := []string{}
		for _, child := range n.Children {
			childrenReprs = append(childrenReprs, child.repr(level + 1))
		}

		return fmt.Sprintf(
			"%s(%s %s)",
			prefix,
			n.Symbol,
			strings.Join(childrenReprs, " "))
	}
}


