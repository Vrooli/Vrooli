package adapters

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed licenses/creativeml-openrail-m-2022-08-22.txt
var openRAILMLicense []byte

const OpenRAILMTermsVersion = "creativeml-openrail-m-2022-08-22"

// LicenseTerms returns the full immutable license text required for an
// adapter's versioned user agreement.
func LicenseTerms(version string) ([]byte, error) {
	if strings.TrimSpace(version) != OpenRAILMTermsVersion {
		return nil, fmt.Errorf("unknown adapter license terms version %q", version)
	}
	return append([]byte(nil), openRAILMLicense...), nil
}
