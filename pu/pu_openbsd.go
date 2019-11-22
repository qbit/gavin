// +build openbsd

package pu

import (
	"golang.org/x/sys/unix"
)

func U(path string, perms string) {
	unix.Unveil(path, perms)
}

func UBlock() error {
	return unix.UnveilBlock()
}
