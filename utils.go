package main

import (
	"os/exec"
	"runtime"
	"strings"
)

type Opener struct {
	cmd  string
	args []string
}

func (s *Opener) Open(url Url) {
	args := append(s.args, string(url))
	exec.Command(s.cmd, args...).Start()
}

func NewOpener() Opener {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	case "darwin":
		cmd = "open"
	default: // linux freebsd openbsd netbsd
		cmd = "xdg-open"
	}
	return Opener{cmd, args}
}

type Arr[T any] []T

func (s *Arr[T]) Append(u T) {
	*s = append(*s, u)
}

func (s *Arr[T]) Remove(i int) {
	*s = append((*s)[:i], (*s)[i+1:]...)
}

func (s *Arr[T]) Clear() {
	*s = nil
}

func tokenize[T ~string](cmd T) (T, T, T) {
	r := make([]string, 3)
	copy(r, strings.Split(string(cmd), " "))
	return T(r[0]), T(r[1]), T(r[2])
}
