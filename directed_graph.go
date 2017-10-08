package pcfg

import (
	"math"
)

// Vertex in graoh
type Vertex string

// DirectedGraph represents a weighted directed graph
type DirectedGraph struct {
	Arcs map[Vertex]map[Vertex]float64
	Vertices map[Vertex]bool
}

// NewDirectedGraph creates a new DirectedGraph
func NewDirectedGraph() *DirectedGraph {
	g := new(DirectedGraph)
	g.Arcs = make(map[Vertex]map[Vertex]float64)
	g.Vertices = make(map[Vertex]bool)
	return g
}

// Add adds an arc into graph
func (g *DirectedGraph) Add(s, t Vertex, weight float64) {
	if g.Arcs[s] == nil {
		g.Arcs[s] = map[Vertex]float64{}
	}
	g.Arcs[s][t] = weight
	g.Vertices[s] = true
	g.Vertices[t] = true
}

// HasArc returns whether arc (s, t) exists in this graph
func (g *DirectedGraph) HasArc(s, t Vertex) bool {
	if _, ok := g.Arcs[s]; !ok {
		return false
	}
	if _, ok := g.Arcs[s][t]; !ok {
		return false
	}
	return true
}

// DFS runs depth-first search on graph and returns the vertices visited by
// deep-first order.
// It will not visit the vertices where visited[V] == true.
// After finished, it will update the visited map
func (g *DirectedGraph) DFS(s Vertex, visited map[Vertex]bool) []Vertex {
	if visited[s] || !g.Vertices[s] {
		return []Vertex{}
	}
	visited[s] = true

	order := []Vertex{s}
	outgoingArcs, ok := g.Arcs[s]
	if ok {
		for nextV, _ := range outgoingArcs {
			order = append(order, g.DFS(nextV, visited)...)
		}
	}
	return order
}

// TopologicalSort sorts the graph by topological order
func (g *DirectedGraph) TopologicalSort() []Vertex {
	visited := map[Vertex]bool{}
	topologicalOrder := []Vertex{}
	for v := range g.Vertices {
		if !visited[v] {
			topologicalOrder = append(g.DFS(v, visited), topologicalOrder...)
		}
	}
	return topologicalOrder
}


// Transpose returns the reversed graph of g
func (g *DirectedGraph) Transpose() *DirectedGraph {
	reversed := NewDirectedGraph()
	for s, targets := range g.Arcs {
		for t, weight := range targets {
			reversed.Add(t, s, weight)
		}
	}

	return reversed
}

// StrongComponents find strong connected components from graph
func (g *DirectedGraph) StrongComponents() [][]Vertex {
	visited := map[Vertex]bool{}
	components := [][]Vertex{}
	topologicalOrder := g.TopologicalSort()
	gt := g.Transpose()
	for _, v := range topologicalOrder {
		if visited[v] {
			continue
		}

		component := gt.DFS(v, visited)
		if len(component) <= 1 {
			continue
		}
		components = append(components, component)
	}
	return components
}

// Floyd finds the weight of shortest path between each vertices using
// Floyd–Warshall algorithm
func (g *DirectedGraph) Floyd() map[Vertex]map[Vertex]float64 {
	distance := map[Vertex]map[Vertex]float64{}
	for s, _ := range g.Vertices {
		distance[s] = map[Vertex]float64{}
		for t, _ := range g.Vertices {
			if s == t {
				distance[s][t] = 0
			} else {
				distance[s][t] = math.Inf(1)
			}
		}
	}

	for s, ts := range g.Arcs {
		for t, w := range ts {
			distance[s][t] = w
		}
	}

	// According to https://en.wikipedia.org/wiki/Floyd%E2%80%93Warshall_algorithm
	for k, _ := range g.Vertices {
		for i, _ := range g.Vertices {
			for j, _ := range g.Vertices {
				d := distance[i][k] + distance[k][j]
				if distance[i][j] > d {
					distance[i][j] = d
				}
			}
		}
	}

	return distance
}
