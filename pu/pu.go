// +build !openbsd

package pu

import "fmt"

func U(path string, perms string) {
	fmt.Printf("WARNING: no unveil (%s, %s)\n", path, perms)
}

func UBlock() error {
	return nil
}
