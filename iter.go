package swiss

type Iterator[K comparable, V any] struct {
	ctrl   []metadata
	groups []group[K, V]
	g, n   int
}

// Iterator creates a new Iterator that returns the key-value pairs found in
// the map. See Iter for details on behavior and guarantees.
func (m *Map[K, V]) Iterator() *Iterator[K, V] {
	// take a consistent view of the table in case
	// we rehash during iteration
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

}

func (it *Iterator[K, V]) Pair() (K, V) {

}

/*
	for n := 0; n < len(groups); n++ {
		for s, c := range ctrl[g] {
			if c == empty || c == tombstone {
				continue
			}
			k, v := groups[g].keys[s], groups[g].values[s]
			if stop := cb(k, v); stop {
				return
			}
		}
		g++
		if g >= uint32(len(groups)) {
			g = 0
		}
	}
*/
