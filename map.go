package swiss

import (
	"math"
	"math/bits"

	"github.com/dolthub/maphash"
)

const (
	width      = 16
	loadFactor = 14.0 / 16.0
)

// Map is an open-addressing hash map
// based on Abseil's flat_hash_map.
type Map[K comparable, V any] struct {
	ctrl     []metadata
	groups   []group[K, V]
	hash     maphash.Hasher[K]
	resident uint32
	dead     uint32
	limit    uint32
}

// group is a group of 16 key-value pairs
type group[K comparable, V any] struct {
	keys   [width]K
	values [width]V
}

// NewMap constructs a Map.
func NewMap[K comparable, V any](sz uint32) (m *Map[K, V]) {
	groups := numGroups(sz)
	m = &Map[K, V]{
		ctrl:   make([]metadata, groups),
		groups: make([]group[K, V], groups),
		hash:   maphash.NewHasher[K](),
		limit:  groups * 14,
	}
	for i := range m.ctrl {
		m.ctrl[i] = newEmptyMetadata()
	}
	return
}

// Has returns true if |key| is present in |m|.
func (m *Map[K, V]) Has(key K) (ok bool) {

	hi, lo := hashKey(m.hash, key)
	_, _, ok = m.find(key, hi, lo)
	return
}

// Get returns the |value| mapped by |key| if one exists.
func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	hi, lo := hashKey(m.hash, key)
	var g, s uint32
	g, s, ok = m.find(key, hi, lo)
	if ok {
		value = m.groups[g].values[s]
	}
	return
}

// Put attempts to insert |key| and |value|
func (m *Map[K, V]) Put(key K, value V) {
	if m.resident >= m.limit {
		m.rehash(m.nextSize())
	}
	hi, lo := hashKey(m.hash, key)
	g, s, ok := m.find(key, hi, lo)
	if !ok {
		m.resident++
	}
	m.ctrl[g][s] = int8(lo)
	m.groups[g].keys[s] = key
	m.groups[g].values[s] = value
}

// Delete attempts to remove |key|, returns true successful.
func (m *Map[K, V]) Delete(key K) bool {
	hi, lo := hashKey(m.hash, key)
	g, s, ok := m.find(key, hi, lo)
	if !ok {
		// |key| is absent, delete failed
		return false
	}
	// optimization: if |m.ctrl[g]| contains any empty
	// metadata bytes, we can physically delete |key|
	// rather than placing a tombstone.
	// The observation is that any probes into group |g|
	// would already be terminated by the existing empty
	// slot, and therefore reclaiming slot |s| will not
	// cause premature termination of probes into |g|.
	if metaMatchEmpty(&m.ctrl[g]) != 0 {
		m.ctrl[g][s] = empty
		m.resident--
	} else {
		m.ctrl[g][s] = tombstone
		m.dead++
	}
	var zerok K
	var zerov V
	m.groups[g].keys[s] = zerok
	m.groups[g].values[s] = zerov
	return true
}

// Count returns the number of elements in the Map.
func (m *Map[K, V]) Count() int {
	return int(m.resident - m.dead)
}

// find returns the location of |key| if present, or its insertion location if absent.
func (m *Map[K, V]) find(key K, hi h1, lo h2) (g, s uint32, ok bool) {
	g = probeStart(hi, len(m.groups))
	for {
		set := metaMatchH2(&m.ctrl[g], lo)
		for set != 0 {
			s = uint32(bits.TrailingZeros16(uint16(set)))
			if key == m.groups[g].keys[s] {
				return g, s, true
			}
			set &= ^(1 << s) // clear bit |s|
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		set = metaMatchEmpty(&m.ctrl[g])
		if set != 0 {
			s = uint32(bits.TrailingZeros16(uint16(set)))
			return g, s, false
		}
		g += 1 // linear probing
		if g >= uint32(len(m.groups)) {
			g = 0
		}
	}
}

func (m *Map[K, V]) nextSize() (n uint32) {
	n = uint32(len(m.groups)) * 2
	if m.dead >= (m.resident / 2) {
		n = uint32(len(m.groups))
	}
	return
}

func (m *Map[K, V]) rehash(n uint32) {
	groups, ctrl := m.groups, m.ctrl
	m.groups = make([]group[K, V], n)
	m.ctrl = make([]metadata, n)
	for i := range m.ctrl {
		m.ctrl[i] = newEmptyMetadata()
	}
	m.hash = maphash.NewHasher[K]()
	m.limit = n * 14
	m.resident, m.dead = 0, 0
	for g := range ctrl {
		for s := range ctrl[g] {
			c := ctrl[g][s]
			if c == empty || c == tombstone {
				continue
			}
			m.Put(groups[g].keys[s], groups[g].values[s])
		}
	}
}

func (m *Map[K, V]) loadFactor() float32 {
	slots := float32(len(m.groups) * width)
	return float32(m.resident-m.dead) / slots
}

func probeStart(hi h1, groups int) uint32 {
	return fastModN(uint32(hi), uint32(groups))

}

// numGroups returns the minimum number of groups needed to store |n|
// elements, accounting for load factor and power of 2 alignment
func numGroups(n uint32) (groups uint32) {
	groups = uint32(math.Ceil(float64(n) / 14.0))
	if groups == 0 {
		groups = 1
	}
	return
}