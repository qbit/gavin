// +build !openbsd

package pu

import "fmt"

func Pledge(promisess string) {
	return nil
}

func Unveil(path string, perms string) {
	return nil
}

func UnveilBlock() error {
	return nil
}
