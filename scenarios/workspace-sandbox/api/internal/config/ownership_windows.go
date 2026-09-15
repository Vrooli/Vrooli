//go:build windows

package config

import "os"

func validateAuthoritativeDirectory(_ string, _ os.FileInfo) error { return nil }
