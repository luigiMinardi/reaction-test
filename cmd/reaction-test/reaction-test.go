package main

import (
	"fmt"
	"os"
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

func main() {
	winsize, err := GetWinsize()
	if err != nil {
		fmt.Println(err)
		panic("failed to get winsize")
	}
	fmt.Println(winsize.Col, winsize.Row, winsize.Xpixel, winsize.Ypixel, winsize)
}
