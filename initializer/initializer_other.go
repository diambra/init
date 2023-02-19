//go:build !windows
// +build !windows

package initializer

import (
	"syscall"

	"github.com/go-kit/log"
)

func (i *Initializer) Init(logger log.Logger) error {
	oldmask := syscall.Umask(0077)
	defer syscall.Umask(oldmask)
	return i.init(logger)
}
