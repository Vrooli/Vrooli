package replay_config

import "errors"

var (
	errMissingConfig = errors.New("config is required")
	errInvalidConfig = errors.New("config payload is invalid")
)
