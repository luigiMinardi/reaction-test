package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Do the interface allocations only once for common
// Errno values.
var (
	errEAGAIN error = syscall.EAGAIN
	errEINVAL error = syscall.EINVAL
	errENOENT error = syscall.ENOENT
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case unix.EAGAIN:
		return errEAGAIN
	case unix.EINVAL:
		return errEINVAL
	case unix.ENOENT:
		return errENOENT
	}
	return e
}

// calls ioctl for file fd with request code req and the result will be in the arguments arg
// if an error happens it returns an error and the arg will be nil
func ioctlPtr(fd int, req uint, arg unsafe.Pointer) (err error) {
	_, _, e1 := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(arg))
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}

// Usually you will use term.GetSize passing int(os.Stdout.Fd()) to it
// https://pkg.go.dev/golang.org/x/term#GetSize
// For the sake of understanding how things works I've rewritten the functions
// that Go uses underneath focusing only on linux (Go supports a lot of other kernels)
// this means that this will work on my computer but may not work on yours, try using
// a linux VM in that case. If you are on linux my base is zsyscall_linux.go from
// https://cs.opensource.google/go/x/sys/+/master:unix/zsyscall_linux.go?q=ioctlPtr&ss=go%2Fx%2Fsys
// term_unix.go from
// https://cs.opensource.google/go/x/term/+/master:term_unix.go?q=getSize&ss=go%2Fx%2Fterm
// ioctl_unsigned.go from
// https://cs.opensource.google/go/x/sys/+/refs/tags/v0.37.0:unix/ioctl_unsigned.go;l=59
// zerrors_linux.go from
// https://cs.opensource.google/go/x/sys/+/master:unix/zerrors_linux.go;drc=a4199c0bfe68a7d7de6e44cead3c91e7bd1e328d;l=4092
// all of those links are from the go generated with "go:build linux" on the go version of the package.
func GetWinsize() (*unix.Winsize, error) {
	var winsize unix.Winsize
	// unix.TIOCGWINSZ is the ioctl request code that is used to get or set the current winsize
	// syscall.SIGWINCH is the signal number sent to the foreground process group
	// when the window size changes
	err := ioctlPtr(int(os.Stdout.Fd()), uint(unix.TIOCGWINSZ), unsafe.Pointer(&winsize))
	return &winsize, err
}

// fd = File Descriptor, for unix it will be os.Stdout.Fd() for example
// b is the byte slice to be written
// nb is the number of bytes that were written
func write(fd int, b []byte) (nb int, err error) {

	n, _, errno := unix.Syscall(
		unix.SYS_WRITE,
		uintptr(fd),
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)),
	)
	if errno != 0 {
		return int(n), errnoErr(errno)
	}
	return int(n), nil
}

// Mutates the byteArray inserting the character char
func fill(bArr []byte, char byte) {
	for i := range bArr {
		bArr[i] = char
	}
}

// return the index of an element in a 2d grid represented in a slice
func index(row, col, rows, cols int) int {
	if row >= 0 && row < rows && col >= 0 && col < cols {
		return row*cols + col
	}
	panic(
		fmt.Sprintf(
			"grid index out of bounds: row=%d col=%d index=%d size=(%dx%d) length=%d\n%s",
			row, col, row*cols+col, rows, cols, rows*cols,
			"remember that index start's at 0, if index==length you're actually at length +1",
		),
	)
}

// return the index of an element in a unix winsize represented in a slice
func windex(winsize unix.Winsize, row, col int) int {
	return index(row, col, int(winsize.Row), int(winsize.Col))
}

// Mutates a byte array drawing borders in it
func drawBorders(buf []byte, winsize unix.Winsize) {
	i := windex(winsize, 0, int(winsize.Col)-1)
	fill(buf[0:i+1], '=')
	buf[winsize.Col] = '\n'

	for row := 1; row < int(winsize.Row)-1; row++ {
		i = windex(winsize, int(row), int(winsize.Col)-1)
		j := windex(winsize, int(row), 1)
		buf[j-1] = '|'
		buf[i] = '|'
		buf[i+1] = '\n'
	}

	i = windex(winsize, int(winsize.Row)-1, int(winsize.Col)-1)
	j := windex(winsize, int(winsize.Row)-1, 0)
	fill(buf[j:i+1], '=')
}

// make the inner side of the buffer filled with spaces
func clearBoard(buf []byte, winsize unix.Winsize) {
	for row := 1; row < int(winsize.Row)-1; row++ {
		i := windex(winsize, int(row), int(winsize.Col)-1)
		j := windex(winsize, int(row), 1)
		fill(buf[j:i], ' ')
	}
}

type Coords struct {
	Row int
	Col int
}

func drawBox(buf []byte, winsize unix.Winsize, start, end Coords) {
	for row := start.Row; row < end.Row; row++ {
		i := windex(winsize, int(row), start.Col)
		j := windex(winsize, int(row), end.Col)
		fill(buf[i:j], 'o')
	}
}

func main() {
	var previousFrameSize unix.Winsize

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGWINCH)
	/*
		write(int(os.Stdout.Fd()), []byte("\033[?1049h"))
		write(int(os.Stdout.Fd()), []byte("\033[?1006h"))
		write(int(os.Stdout.Fd()), []byte("\033[?1000h"))
	*/

	winsize, err := GetWinsize()
	if err != nil {
		fmt.Println(err)
		panic("failed to get winsize")
	}
	if previousFrameSize.Col != winsize.Col ||
		previousFrameSize.Row != winsize.Row {
		write(int(os.Stdout.Fd()), []byte("\033[3J\033[H"))
		fmt.Println(winsize.Col, winsize.Row, winsize.Xpixel, winsize.Ypixel, winsize)
		buf := make([]byte, winsize.Row*winsize.Col)
		drawBorders(buf, *winsize)
		clearBoard(buf, *winsize)
		drawBox(buf, *winsize, Coords{Row: 10, Col: 10}, Coords{Row: 15, Col: 20})
		write(int(os.Stdout.Fd()), buf)
	}

	previousFrameSize.Col = winsize.Col
	previousFrameSize.Row = winsize.Row
	previousFrameSize.Xpixel = winsize.Xpixel
	previousFrameSize.Ypixel = winsize.Ypixel
	for {
		// read from channel
		sig := <-sigChan
		switch sig {
		case os.Interrupt:
			/*
				write(int(os.Stdout.Fd()), []byte("\033[?1049l"))
				write(int(os.Stdout.Fd()), []byte("\033[?1006l"))
				write(int(os.Stdout.Fd()), []byte("\033[?1000l"))
			*/
			return
		case syscall.SIGWINCH:
			winsize, err := GetWinsize()
			if err != nil {
				fmt.Println(err)
				panic("failed to get winsize")
			}
			if previousFrameSize.Col != winsize.Col ||
				previousFrameSize.Row != winsize.Row {
				write(int(os.Stdout.Fd()), []byte("\033[3J\033[H"))
				fmt.Println(winsize.Col, winsize.Row, winsize.Xpixel, winsize.Ypixel, winsize)
				buf := make([]byte, winsize.Row*winsize.Col)
				drawBorders(buf, *winsize)
				clearBoard(buf, *winsize)
				drawBox(buf, *winsize, Coords{Row: 10, Col: 10}, Coords{Row: 15, Col: 20})
				write(int(os.Stdout.Fd()), buf)
			}

			previousFrameSize.Col = winsize.Col
			previousFrameSize.Row = winsize.Row
			previousFrameSize.Xpixel = winsize.Xpixel
			previousFrameSize.Ypixel = winsize.Ypixel
		}
	}
}
