package config

import "os"

func allowed(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

