//go:build !linux

package hostwireless

import (
	"errors"
	"os"
)

func openSignedBundleParent(string) (*os.Root, error) {
	return nil, errors.New("signed kernel bundle publication requires Linux")
}

func chownSignedBundleFile(*os.File, string) error {
	return errors.New("signed kernel bundles require Linux")
}
