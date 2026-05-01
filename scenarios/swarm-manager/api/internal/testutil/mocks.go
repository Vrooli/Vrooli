package testutil

import "swarm-manager/internal/testutil/mocks"

// NoopInvalidator is a no-op implementation of dispatch.Invalidator for tests.
type NoopInvalidator = mocks.NoopInvalidator

// NoopNodeDispatcher is a no-op implementation of dispatch.NodeDispatcher for tests.
type NoopNodeDispatcher = mocks.NoopNodeDispatcher

// RecordingInvalidator captures DispatchInvalidate calls for assertions.
type RecordingInvalidator = mocks.RecordingInvalidator

// ErrorWriter is an http.ResponseWriter that always fails on Write,
// for testing JSON encoding error paths.
type ErrorWriter = mocks.ErrorWriter
