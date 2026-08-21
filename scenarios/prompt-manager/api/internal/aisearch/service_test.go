package aisearch

// sentinel error for testing
var errForTesting = &testError{"test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
