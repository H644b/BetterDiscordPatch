//go:build darwin
// +build darwin

package main

import (
	"os/exec"
)

func notify(title, message string) error {
	return exec.Command("osascript", "-e", `display notification "`+message+`" with title "`+title+`"`).Run()
}
