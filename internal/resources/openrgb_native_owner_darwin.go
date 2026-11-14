//go:build darwin

package resources

import (
	"errors"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"os"
	"strconv"
)

func openRGBNativeOperatorPrincipal() (string, error) {
	uid, _, sudoed := hostreqkit.InvokingUserIDs()
	if !sudoed {
		uid = os.Geteuid()
	}
	if uid <= 0 {
		return "", errors.New("native OpenRGB owner requires an original non-root operator identity")
	}
	return "uid:" + strconv.Itoa(uid), nil
}
