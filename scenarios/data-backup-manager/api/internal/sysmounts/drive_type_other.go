//go:build !windows

package sysmounts

func platformDriveType(string) uint32 { return driveTypeUnknown }
