//go:build !linux

package desktophelper

func readPrivate(string, int64) ([]byte, error) { return nil, ErrBootstrap }
func privateDirectory(string) error             { return ErrBootstrap }
func lockDirectory(string) (func(), error)      { return nil, ErrBootstrap }
func checkPrivateDatabase(string) error         { return ErrBootstrap }
func lockPrivateFile(string) (func(), error)    { return nil, ErrBootstrap }
