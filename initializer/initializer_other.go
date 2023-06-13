//go:build !windows
// +build !windows

package initializer

import (
	"syscall"
)

func (i *Initializer) Init() error {
	oldmask := syscall.Umask(0077)
	defer syscall.Umask(oldmask)
	return i.init()
}
