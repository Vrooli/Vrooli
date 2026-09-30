//go:build windows

package lifecycle

// Windows has no SIGPIPE: a write to a closed pipe already fails with an
// error rather than ending the process.
func subscribeBrokenPipe() {}

func unsubscribeBrokenPipe() {}
