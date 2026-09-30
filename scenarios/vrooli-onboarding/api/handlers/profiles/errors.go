package profiles

import (
	"connectrpc.com/connect"
	"strings"
)

func profilesError(err error) error {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "not found") || strings.Contains(message, "required") || strings.Contains(message, "unsupported") || strings.Contains(message, "profile") || strings.Contains(message, "question") {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
