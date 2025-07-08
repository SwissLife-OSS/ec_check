package main

import (
	"cmp"
	"iter"
	"sort"
)

// mapOrderedByKey returns an iterator allowing to iterate over a given
// map ordered by key.
func mapOrderedByKey[K cmp.Ordered, E any](m map[K]E) iter.Seq2[K, E] {
	return func(yield func(K, E) bool) {
		keys := make([]K, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		for _, k := range keys {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
