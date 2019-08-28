// +build !windows

package main

import (
	"fmt"
	"syscall"
)

func fdRaise(nn int) error {
	n := uint64(nn)

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}

	if uint64(rLimit.Cur) >= n {
		fmt.Printf("already at %d >= %d fds\n", rLimit.Cur, n)
		return nil // all good.
	}
	var i interface{} = &rLimit.Cur
	switch i := i.(type) {
	case *uint64:
		*i = uint64(n)
	case *int64:
		*i = int64(n)
	default:
		return fmt.Errorf("error message")
	}

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}

	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}

	if uint64(rLimit.Cur) < n {
		return fmt.Errorf("failed to raise fd limit to %d (still %d)", n, rLimit.Cur)
	}

	fmt.Printf("raised fds to %d >= %d fds\n", rLimit.Cur, n)
	return nil
}
