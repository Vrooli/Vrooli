package resources

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

// ValidateOpenRGBNativeResourceStorage rejects a changed ambient namespace
// before intent, quiescence or cleanup. Initial preparation may choose a root
// before boot enrollment; later changes require explicit reconfiguration.
func (c *Controller) ValidateOpenRGBNativeResourceStorage(ctx context.Context) error {
	if runtime.GOOS != openRGBPlatformDarwin && runtime.GOOS != openRGBPlatformWindows {
		return nil
	}
	if c == nil || ctx == nil {
		return errors.New("native OpenRGB storage validation requires its operator controller")
	}
	enrolled, configured, err := platform.EnrolledNativeOperatorResourceStorageRoot(ctx, c.Home)
	if err != nil {
		return err
	}
	if !configured {
		control, err := lightingruntime.NewController(c.Home, lightingruntime.Owners{})
		if err != nil {
			return err
		}
		intent, err := control.Current()
		if err != nil {
			return err
		}
		if err := validateNativeOpenRGBUnknownStorageIntent(intent); err != nil {
			if proofErr := c.verifyOpenRGBRetainedStorageRoot(ctx, intent); proofErr != nil {
				return errors.Join(err, proofErr)
			}
		}
		return nil
	}
	current, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return err
	}
	defaultRoot := ""
	if enrolled == "" {
		defaultRoot, err = nativeOpenRGBMachineRoot()
		if err != nil {
			return err
		}
	}
	return validateNativeOpenRGBStorageEnrollment(runtime.GOOS, current, enrolled, configured, defaultRoot)
}

func validateNativeOpenRGBStorageEnrollment(goos, current, enrolled string, configured bool, defaultRoot string) error {
	if goos != openRGBPlatformDarwin && goos != openRGBPlatformWindows || !configured {
		return nil
	}
	expected := defaultRoot
	if enrolled != "" {
		if !filepath.IsAbs(enrolled) || filepath.Clean(enrolled) != enrolled || strings.ContainsAny(enrolled, "\x00\r\n") {
			return errors.New("enrolled native resource root is not canonical")
		}
		expected = filepath.Join(enrolled, "native", "openrgb")
	}
	if !filepath.IsAbs(current) || filepath.Clean(current) != current || !filepath.IsAbs(expected) || filepath.Clean(expected) != expected {
		return errors.New("native resource roots must be canonical before lifecycle actions")
	}
	same := current == expected
	if goos == openRGBPlatformWindows {
		same = strings.EqualFold(current, expected)
	}
	if same {
		return nil
	}
	restore := "unset VROOLI_RESOURCE_STORAGE_ROOT to restore its enrolled default"
	if enrolled != "" {
		restore = "restore VROOLI_RESOURCE_STORAGE_ROOT=" + strconv.Quote(enrolled)
	}
	return fmt.Errorf("native OpenRGB storage differs from its enrolled operator namespace; %s before normal disable or cleanup, then use explicit setup to reconfigure", restore)
}

func validateNativeOpenRGBUnknownStorageIntent(intent lightingruntime.Intent) error {
	if intent.Phase != lightingruntime.Disabled {
		return errors.New("native OpenRGB storage enrollment is unavailable while lighting intent is active; restore its retained operator enrollment before lifecycle or cleanup")
	}
	return nil
}
