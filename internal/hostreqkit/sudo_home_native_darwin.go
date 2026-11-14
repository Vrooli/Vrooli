//go:build darwin

package hostreqkit

import (
	"errors"
	"fmt"
	"os"
	osuser "os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// OpenDirectory users normally have no /etc/passwd entry. Under sudo, HOME
// belongs to root; it must never substitute for the invoking account's home.
func nativeInvokingUserHome() (string, bool, error) {
	if !RunningAsRootFn() || strings.TrimSpace(os.Getenv("SUDO_USER")) == "" {
		return "", false, nil
	}
	uid, _, ok := InvokingUserIDs()
	if !ok {
		return "", true, errors.New("native invoking operator home requires authoritative non-root sudo IDs")
	}
	home, err := resolveDarwinInvokingHome(uid, lookupDarwinInvokingUser)
	return home, true, err
}

func resolveDarwinInvokingHome(uid int, lookup func(string) (*osuser.User, error)) (string, error) {
	if uid <= 0 || lookup == nil {
		return "", errors.New("native invoking operator requires a non-root UID resolver")
	}
	user, err := lookup(strconv.Itoa(uid))
	if err != nil {
		return "", fmt.Errorf("resolve macOS invoking operator uid %d through native account directory: %w", uid, err)
	}
	if user == nil || user.Uid != strconv.Itoa(uid) || !filepath.IsAbs(user.HomeDir) {
		return "", errors.New("native invoking account directory returned a different UID or invalid home")
	}
	home := filepath.Clean(user.HomeDir)
	if home == string(filepath.Separator) || strings.ContainsAny(home, "\x00\r\n") {
		return "", errors.New("native invoking account home is not a qualified absolute user directory")
	}
	return home, nil
}
