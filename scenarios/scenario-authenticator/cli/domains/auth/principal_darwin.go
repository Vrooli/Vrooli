//go:build darwin

package auth

import (
	"fmt"
	"os"
)

func currentLocalPrincipal() (string, error) { return fmt.Sprintf("unix:%d", os.Getuid()), nil }
