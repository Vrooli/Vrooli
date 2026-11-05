package main

import (
	"fmt"
	"os"
	"os/exec"

	"audio-tools/internal/buildidentity"
)

func main() {
	identity, err := buildidentity.SourceFingerprint(".")
	if err != nil {
		fail("compute source identity", err)
	}

	command := exec.Command("go", "build", "-ldflags", fmt.Sprintf("-X audio-tools/internal/buildidentity.SourceIdentity=%s", identity), "-o", "audio-tools-api", ".")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fail("build API", err)
	}
	fmt.Println(identity)
}

func fail(action string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", action, err)
	os.Exit(1)
}
