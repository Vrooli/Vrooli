//go:build !linux

package maintenance

import "fmt"

func readProcessOpenFiles(int) ([]string, error) {
	return nil, fmt.Errorf("open-transcript evidence is unsupported on this host")
}
