# Historical validation issue

The immutable baseline captured a fixture mismatch in
`TestRunDevelopRunsSetupWhenNeededAndStartsNativeServices`: the test received
an API launch spec with port 8092 while asserting 18095. The implementation
resolved the environment-precedence defect in `resolveAPIPort`; the current
`go test ./internal/setup/...` suite passes. This file is retained to preserve
the historical baseline observation and is not a current red test.
