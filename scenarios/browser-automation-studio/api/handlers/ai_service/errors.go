package ai_service

import "errors"

var (
	errInvalidURL    = errors.New("invalid url")
	errMissingURL    = errors.New("url is required")
	errMissingIntent = errors.New("intent is required")
	errDOMNotObject  = errors.New("dom tree payload was not a JSON object")
)
