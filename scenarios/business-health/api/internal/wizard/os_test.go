package wizard

import "os"

func osReadDir(dir string) ([]os.DirEntry, error) { return os.ReadDir(dir) }
