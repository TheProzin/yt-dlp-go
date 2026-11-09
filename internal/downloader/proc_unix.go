//go:build !windows

package downloader

import (
	"syscall"
	"time"
)

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcess(pid int) {
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	go func() {
		time.Sleep(3 * time.Second)
		// Only force-kill if the process group still exists.
		if err := syscall.Kill(-pid, 0); err == nil {
			_ = syscall.Kill(-pid, syscall.SIGKILL)
		}
	}()
}
