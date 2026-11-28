//go:build !darwin

package hostreqkit

func nativeInvokingUserHome() (string, bool, error) { return "", false, nil }
