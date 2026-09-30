package main

import (
	"net/http"

	"github.com/vrooli/api-core/health"
)

var healthHandler http.HandlerFunc = health.New("api-library").
	Version("1.0.0").
	Handler()
