package deps

import (
	"errors"
	"strings"

	"connectrpc.com/connect"
)

// ToConnectError maps deps-domain sentinels into Connect codes. Unknown
// errors collapse to Internal so callers never depend on raw storage
// details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var notIndexed ErrComponentNotIndexed
	if errors.As(err, &notIndexed) {
		return connect.NewError(connect.CodeFailedPrecondition, notIndexed)
	}
	var pkgMissing ErrScenarioPackageJSONMissing
	if errors.As(err, &pkgMissing) {
		return connect.NewError(connect.CodeNotFound, pkgMissing)
	}
	if strings.Contains(err.Error(), "required") {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
