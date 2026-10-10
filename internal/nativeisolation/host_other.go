//go:build !linux

package nativeisolation

func trustedPath(string, uint32) error      { return ErrRefused }
func trustedTree(string, uint32) error      { return ErrRefused }
func checkWorkspace(string, []string) error { return ErrRefused }
func nativeLeader(int, uint32) bool         { return false }
func treeDigest(string) (string, error)     { return "", ErrRefused }
