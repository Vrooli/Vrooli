//go:build !unix && !windows

package platform

import "context"

func processGroupHasLiveMembers(context.Context, int) (bool, error) {
	return false, ErrUnsupported
}
