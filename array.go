package main

type Arr[T any] []T

func (s Arr[T]) Length() int {
	return len(s)
}

func (s *Arr[T]) Append(u T) {
	*s = append(*s, u)
}

func (s *Arr[T]) Remove(i int) {
	*s = append((*s)[:i], (*s)[i+1:]...)
}

func (s *Arr[T]) Clear() {
	*s = nil
}
