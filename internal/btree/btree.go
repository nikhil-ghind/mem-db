package btree

import (
	"sort"
	"sync"
)

// TreeStats holds statistics about the B+ tree.
type TreeStats struct {
	Size   int
	Height int
	Order  int
}

// BPlusTree is a concurrent-safe B+ tree with string keys and byte-slice values.
type BPlusTree struct {
	mu    sync.RWMutex
	root  *Node
	order int
	size  int
}

// New creates a new B+ tree with the given order.
// If order <= 0, the default ORDER constant is used.
func New(order int) *BPlusTree {
	if order <= 0 {
		order = ORDER
	}
	return &BPlusTree{
		root:  newLeaf(),
		order: order,
	}
}

// Size returns the number of key-value pairs in the tree.
func (t *BPlusTree) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.size
}

// Height returns the height of the tree (1 = just a leaf).
func (t *BPlusTree) Height() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	h := 0
	n := t.root
	for n != nil {
		h++
		if n.isLeaf {
			break
		}
		n = n.children[0]
	}
	return h
}

// Stats returns tree statistics.
func (t *BPlusTree) Stats() TreeStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	h := 0
	n := t.root
	for n != nil {
		h++
		if n.isLeaf {
			break
		}
		n = n.children[0]
	}
	return TreeStats{
		Size:   t.size,
		Height: h,
		Order:  t.order,
	}
}

// Get retrieves the value for a key. Returns (value, true) if found.
func (t *BPlusTree) Get(key string) ([]byte, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	leaf := t.findLeaf(key)
	idx := sort.SearchStrings(leaf.keys, key)
	if idx < len(leaf.keys) && leaf.keys[idx] == key {
		return leaf.values[idx], true
	}
	return nil, false
}

// Insert inserts or updates a key-value pair.
func (t *BPlusTree) Insert(key string, value []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	leaf := t.findLeaf(key)

	// Check if key already exists (update in place).
	idx := sort.SearchStrings(leaf.keys, key)
	if idx < len(leaf.keys) && leaf.keys[idx] == key {
		leaf.values[idx] = value
		return
	}

	// Insert into leaf at sorted position.
	leaf.keys = append(leaf.keys, "")
	copy(leaf.keys[idx+1:], leaf.keys[idx:])
	leaf.keys[idx] = key

	leaf.values = append(leaf.values, nil)
	copy(leaf.values[idx+1:], leaf.values[idx:])
	leaf.values[idx] = value

	t.size++

	// Split if overflow.
	if len(leaf.keys) >= t.order {
		t.splitLeaf(leaf)
	}
}

// Delete removes a key. Returns true if the key existed.
func (t *BPlusTree) Delete(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	leaf := t.findLeaf(key)
	idx := sort.SearchStrings(leaf.keys, key)
	if idx >= len(leaf.keys) || leaf.keys[idx] != key {
		return false
	}

	// Remove key and value.
	leaf.keys = append(leaf.keys[:idx], leaf.keys[idx+1:]...)
	leaf.values = append(leaf.values[:idx], leaf.values[idx+1:]...)
	t.size--
	return true
}

// RangeScan returns all key-value pairs with startKey <= key <= endKey.
func (t *BPlusTree) RangeScan(startKey, endKey string) []KeyValue {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []KeyValue
	leaf := t.findLeaf(startKey)

	for leaf != nil {
		for i, k := range leaf.keys {
			if k > endKey {
				return results
			}
			if k >= startKey {
				results = append(results, KeyValue{Key: k, Value: leaf.values[i]})
			}
		}
		leaf = leaf.next
	}
	return results
}

// findLeaf traverses from root to the leaf that should contain the given key.
// Caller must hold at least a read lock.
func (t *BPlusTree) findLeaf(key string) *Node {
	n := t.root
	for !n.isLeaf {
		idx := sort.SearchStrings(n.keys, key)
		// In a B+ tree internal node with k keys, there are k+1 children.
		// Child at index idx covers keys >= keys[idx-1] and < keys[idx].
		// sort.SearchStrings returns the index where key would be inserted,
		// which corresponds to the correct child pointer.
		if idx < len(n.children) {
			n = n.children[idx]
		} else {
			n = n.children[len(n.children)-1]
		}
	}
	return n
}

// splitLeaf splits a leaf node that has overflowed.
// It creates a new right leaf and pushes the split key up to the parent.
func (t *BPlusTree) splitLeaf(leaf *Node) {
	mid := len(leaf.keys) / 2

	right := newLeaf()
	right.keys = append(right.keys, leaf.keys[mid:]...)
	right.values = append(right.values, leaf.values[mid:]...)
	right.next = leaf.next
	leaf.next = right

	leaf.keys = leaf.keys[:mid]
	leaf.values = leaf.values[:mid]

	splitKey := right.keys[0]
	t.insertIntoParent(leaf, splitKey, right)
}

// insertIntoParent inserts a new key and right child into the parent of left.
// If left is the root, a new root is created.
func (t *BPlusTree) insertIntoParent(left *Node, key string, right *Node) {
	parent := t.findParent(t.root, left)

	if parent == nil {
		// left is the root — create a new root.
		newRoot := newInternal()
		newRoot.keys = append(newRoot.keys, key)
		newRoot.children = append(newRoot.children, left, right)
		t.root = newRoot
		return
	}

	// Find where left sits among parent's children.
	idx := -1
	for i, c := range parent.children {
		if c == left {
			idx = i
			break
		}
	}

	// Insert key at position idx in parent.keys and right at idx+1 in parent.children.
	parent.keys = append(parent.keys, "")
	copy(parent.keys[idx+1:], parent.keys[idx:])
	parent.keys[idx] = key

	parent.children = append(parent.children, nil)
	copy(parent.children[idx+2:], parent.children[idx+1:])
	parent.children[idx+1] = right

	// Split parent if it overflows.
	if len(parent.keys) >= t.order {
		t.splitInternal(parent)
	}
}

// splitInternal splits an internal node.
func (t *BPlusTree) splitInternal(node *Node) {
	mid := len(node.keys) / 2
	pushUpKey := node.keys[mid]

	right := newInternal()
	right.keys = append(right.keys, node.keys[mid+1:]...)
	right.children = append(right.children, node.children[mid+1:]...)

	node.keys = node.keys[:mid]
	node.children = node.children[:mid+1]

	t.insertIntoParent(node, pushUpKey, right)
}

// findParent finds the parent of target starting from the given node.
// Returns nil if target is the root.
func (t *BPlusTree) findParent(current, target *Node) *Node {
	if current.isLeaf || current == target {
		return nil
	}
	for _, child := range current.children {
		if child == target {
			return current
		}
		if !child.isLeaf {
			if p := t.findParent(child, target); p != nil {
				return p
			}
		}
	}
	return nil
}
