package runtimesupervisor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// Resolve the original operator from native account metadata, never from the
// machine broker's HOME. Darwin sudo identity is validated again by platform.
func nativeSupervisorInstallOperator(target, home, image string) (platform.NativeSystemServiceBrokerOptions, error) {
	var zero platform.NativeSystemServiceBrokerOptions
	current, err := user.Current()
	if err != nil {
		return zero, err
	}
	if target == "darwin" && current.Uid == "0" {
		uid, _, ok := hostreqkit.InvokingUserIDs()
		if !ok {
			return zero, fmt.Errorf("native system supervisor requires the original nonroot operator identity")
		}
		current, err = user.LookupId(strconv.Itoa(uid))
		if err != nil {
			return zero, err
		}
	}
	canonicalHome, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return zero, err
	}
	if filepath.Clean(canonicalHome) != home {
		return zero, fmt.Errorf("native supervisor operator home differs from invoking account")
	}
	info, err := os.Lstat(image)
	if err != nil {
		return zero, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 512<<20 {
		return zero, fmt.Errorf("native supervisor requires a bounded regular CLI image")
	}
	f, err := os.Open(image)
	if err != nil {
		return zero, err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil {
		return zero, err
	}
	if !os.SameFile(info, actual) {
		return zero, fmt.Errorf("native supervisor CLI image changed while opening")
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, io.LimitReader(f, 512<<20+1)); err != nil {
		return zero, err
	}
	options, err := nativeSupervisorOperatorMetadata(target, home, hex.EncodeToString(hash.Sum(nil)), current)
	if err != nil {
		return zero, err
	}
	options.GlobalResourceStorageRoot = os.Getenv("VROOLI_RESOURCE_STORAGE_ROOT")
	return options, nil
}

func nativeSupervisorOperatorMetadata(target, home, digest string, u *user.User) (platform.NativeSystemServiceBrokerOptions, error) {
	var zero platform.NativeSystemServiceBrokerOptions
	if u == nil || home == "" || !filepath.IsAbs(home) || filepath.Clean(home) != home || strings.ContainsAny(home+u.Username, "\x00\r\n") || u.HomeDir != home || u.Username == "" {
		return zero, fmt.Errorf("native supervisor requires exact canonical operator account metadata")
	}
	o := platform.NativeSystemServiceBrokerOptions{ExecutableSHA256: digest, OperatorUsername: u.Username, OperatorHome: home}
	switch target {
	case "darwin":
		uid, e := strconv.ParseUint(u.Uid, 10, 32)
		if e != nil || uid == 0 || strconv.FormatUint(uid, 10) != u.Uid {
			return zero, fmt.Errorf("native supervisor requires nonroot operator UID")
		}
		gid, e := strconv.ParseUint(u.Gid, 10, 32)
		if e != nil || gid == 0 || strconv.FormatUint(gid, 10) != u.Gid {
			return zero, fmt.Errorf("native supervisor requires nonroot operator GID")
		}
		o.OperatorUID = uint32(uid)
		o.OperatorGID = uint32(gid)
		o.OperatorPrincipal = "unix:" + u.Uid
	case "windows":
		if !strings.HasPrefix(u.Uid, "S-1-5-21-") && !strings.HasPrefix(u.Uid, "S-1-12-1-") {
			return zero, fmt.Errorf("native supervisor requires a concrete Windows operator SID")
		}
		o.OperatorPrincipal = "windows:" + u.Uid
	default:
		return zero, platform.ErrUnsupported
	}
	return o, nil
}
