package swiss

type Iterator[K comparable, V any] struct {
	ctrl      []metadata
	groups    []group[K, V]
	g, n, s   int
	nextFound bool
}

// Iterator creates a new Iterator that returns the key-value pairs found in
// the map. See Iter for details on behavior and guarantees.
func (m *Map[K, V]) Iterator() *Iterator[K, V] {
	// take a consistent view of the table in case we rehash during iteration
	ctrl, groups := m.ctrl, m.groups
	// pick a random starting group
	g := int(randIntN(len(groups)))
	return &Iterator[K, V]{
		ctrl:   ctrl,
		groups: groups,
		g:      g,
	}
}

// Next returns true if there is a key-value pair to return, false at the end
// of iteration. It must be called prior to reading the first pair.
func (it *Iterator[K, V]) Next() bool {
	if it.nextFound {
		it.s++
	}

	for ; it.n < len(it.groups); it.n++ {
		for ; it.s < len(it.ctrl[it.g]); it.s++ {
			c := it.ctrl[it.g][it.s]
			if c == empty || c == tombstone {
				continue
			}
			it.nextFound = true
			return true
		}
		it.g++
		it.s = 0
		if it.g >= len(it.groups) {
			it.g = 0
		}
	}
	return false
}

func (it *Iterator[K, V]) Pair() (k K, v V) {
	if !it.nextFound {
		return k, v
	}
	return it.groups[it.g].keys[it.s], it.groups[it.g].values[it.s]
}
