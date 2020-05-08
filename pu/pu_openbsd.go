// +build openbsd

package pu

import (
	"golang.org/x/sys/unix"
)

func Pledge(promises string) {
	unix.PledgePromises(promises)
}

func Unveil(path string, perms string) {
	unix.Unveil(path, perms)
}

func UnveilBlock() error {
	return unix.UnveilBlock()
}
