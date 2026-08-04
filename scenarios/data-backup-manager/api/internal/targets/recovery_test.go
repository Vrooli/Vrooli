package targets

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"data-backup-manager/internal/sources"
)

func TestRegisterRefusesRecordedRecoveryBundle(t *testing.T) {
	original := recoveryReceiptPath
	recoveryReceiptPath = func() (string, error) { return "/state/recovery-receipt.json", nil }
	t.Cleanup(func() { recoveryReceiptPath = original })
	// The receipt reader is filesystem-backed; point the seam at a temporary
	// receipt and verify the exact recorded bundle path is not accepted.
	receipt := t.TempDir() + "/recovery-receipt.json"
	data, err := json.Marshal(struct {
		Path string `json:"path"`
	}{Path: "/media/drive/vrooli-credentials.bundle"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receipt, data, 0o600); err != nil {
		t.Fatal(err)
	}
	recoveryReceiptPath = func() (string, error) { return receipt, nil }
	svc := NewService(recoveryTestRepository{})
	_, registerErr := svc.Register(context.Background(), RegisterInput{Owner: "vrooli", Name: "credentials-recovery", SourceKind: sources.KindFilesystem, Locator: "/media/drive/vrooli-credentials.bundle"})
	var invalid ErrInvalidTarget
	if !errors.As(registerErr, &invalid) || invalid.Field != "locator" {
		t.Fatalf("register error = %v, want locator validation", registerErr)
	}
}

type recoveryTestRepository struct{}

func (recoveryTestRepository) Create(context.Context, Target) (Target, error) {
	return Target{}, errors.New("unexpected create")
}
func (recoveryTestRepository) Update(context.Context, Target) (Target, error) {
	return Target{}, errors.New("unexpected update")
}
func (recoveryTestRepository) GetByOwnerName(context.Context, string, string) (Target, error) {
	return Target{}, ErrTargetNotFound{}
}
func (recoveryTestRepository) GetByID(context.Context, string) (Target, error) {
	return Target{}, ErrTargetNotFound{}
}
func (recoveryTestRepository) List(context.Context, string, int) ([]Target, error) { return nil, nil }
func (recoveryTestRepository) DeleteByOwnerName(context.Context, string, string) (bool, error) {
	return false, nil
}
