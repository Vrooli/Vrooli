package main

import (
	"fmt"
	"os"
	"os/exec"
)

// The resource lifecycle is owned by the control plane. This small wrapper
// keeps the resource's generated CLI contract available without introducing a
// second cloudflared installer or a second secret path.
func main() {
	args := append([]string{"resource", "cloudflared"}, os.Args[1:]...)
	command := exec.Command("vrooli", args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
