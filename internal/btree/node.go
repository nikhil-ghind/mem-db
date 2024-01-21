package btree

// ORDER is the maximum number of keys per node.
// A node splits when it has ORDER keys.
const ORDER = 128

// KeyValue holds a key-value pair returned from queries.
type KeyValue struct {
	Key   string
	Value []byte
}

// Node represents both internal and leaf nodes in the B+ tree.
// Leaf nodes store values and link to the next leaf.
// Internal nodes store child pointers.
type Node struct {
	keys     []string
	values   [][]byte // leaf only
	children []*Node  // internal only
	next     *Node    // leaf only: pointer to next leaf for range scans
	isLeaf   bool
}

// newLeaf creates a new empty leaf node.
func newLeaf() *Node {
	return &Node{
		keys:   make([]string, 0, ORDER),
		values: make([][]byte, 0, ORDER),
		isLeaf: true,
	}
}

// newInternal creates a new empty internal node.
func newInternal() *Node {
	return &Node{
		keys:     make([]string, 0, ORDER),
		children: make([]*Node, 0, ORDER+1),
		isLeaf:   false,
	}
}
