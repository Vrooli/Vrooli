package mocks

import "landing-page-business-suite-api/internal/envx"

// FakeEnvironment is an in-memory process-configuration substitute.
type FakeEnvironment map[string]string

func (f FakeEnvironment) Get(key string) string { return f[key] }

var _ envx.Reader = FakeEnvironment{}
