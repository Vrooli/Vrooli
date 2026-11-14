package variant

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"landing-page-business-suite-api/internal/experimentation"
)

func TestPublicBrandingSuppliesCanonicalAuthorityWithoutPrivateFields(t *testing.T) { // [REQ:LP-PRES-011] [REQ:LP-PRES-012]
	store := experimentation.NewConfigStore("", filepath.Join(t.TempDir(), "branding.json"), nil)
	branding := store.GetBranding()
	canonical, smtpHost := "https://suite.example/marketing", "private-mail.internal"
	branding.CanonicalBaseURL, branding.SMTPHost = &canonical, &smtpHost
	if err := store.SaveBranding(branding); err != nil {
		t.Fatal(err)
	}
	response, err := NewBrandingConnectHandler(store).GetPublicBranding(context.Background(), connect.NewRequest(&lpbsv1.GetPublicBrandingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetBranding().GetCanonicalBaseUrl() != canonical {
		t.Fatal("configured canonical authority missing")
	}
	data, err := protojson.Marshal(response.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "smtp") || strings.Contains(string(data), smtpHost) {
		t.Fatal("private branding crossed public projection")
	}
}
