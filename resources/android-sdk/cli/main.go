package main

import (
	"fmt"
	"os"

	"github.com/vrooli/vrooli/resources/android-sdk/cli/internal/androidsdk"
)

func main() {
	if err := androidsdk.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "android-sdk failed:", err)
		os.Exit(1)
	}
}
