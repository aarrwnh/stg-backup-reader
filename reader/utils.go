package reader

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Opener struct {
	cmd  string
	args []string
}

func (s *Opener) Open(url string) {
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

func commandParse[T ~string](input T) (T, T, T) {
	r := make([]string, 3)
	copy(r, strings.Split(string(input), " "))
	return T(r[0]), T(r[1]), T(r[2])
}

func highlightWord(pattern, line string) string {
	start := strings.Index(strings.ToLower(line), strings.ToLower(pattern))
	if start < 0 {
		return line
	}
	pattern = line[start : start+len(pattern)]
	parts := strings.Split(line, pattern)
	return strings.Join(parts, "\033[34m"+pattern+"\033[0m")
}

func setTitle(t string) {
	fmt.Fprintf(os.Stdout, "\033]0;%s\007", t)
}

func printInfo(format string, a ...any) {
	fmt.Fprintf(os.Stdout, "\033[38;2;100;100;100m# %s\033[0m\n", fmt.Sprintf(format, a...))
}

func timeTrack(start time.Time) {
	elapsed := time.Since(start)
	printInfo("...%s", round(elapsed, 2))
}

var divs = []time.Duration{
	time.Duration(1), time.Duration(10), time.Duration(100), time.Duration(1000),
}

func round(d time.Duration, digits int) time.Duration {
	if digits < 0 && digits > len(divs) {
		panic("wrong length provided")
	}
	switch {
	case d > time.Second:
		d = d.Round(time.Second / divs[digits])
	case d > time.Millisecond:
		d = d.Round(time.Millisecond / divs[digits])
	case d > time.Microsecond:
		d = d.Round(time.Microsecond / divs[digits])
	}
	return d
}
