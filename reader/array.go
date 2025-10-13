package reader

import (
	"slices"
)

type Arr[T any] []T

func (s Arr[T]) Length() int {
	return len(s)
}

func (s *Arr[T]) Append(u ...T) {
	*s = append(*s, u...)
}

func (s *Arr[T]) Remove(i int) {
	*s = append((*s)[:i], (*s)[i+1:]...)
}

func (s *Arr[T]) Filter(check func(item T) bool) {
	for i := len(*s) - 1; i >= 0; i-- {
		if check((*s)[i]) {
			s.Remove(i)
		}
	}
}

func (s *Arr[T]) Clear() {
	*s = nil
}

func (s *Arr[T]) Sort(cmp func(a, b T) int) {
	slices.SortFunc(*s, cmp)
}
