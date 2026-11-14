package coreactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
)

const enrollmentWitness = "vrooli-root-update-enrollment-v1\n"

// Enrollment is operator configuration, not an authorization grant or proof
// of update safety. Every update still qualifies its retained native inputs.
type Enrollment struct {
	SchemaVersion   int    `json:"schema_version"`
	Home            string `json:"home"`
	Executable      string `json:"executable"`
	UnitName        string `json:"unit_name"`
	Enabled         bool   `json:"enabled"`
	UpdatedUnixNano int64  `json:"updated_unix_nano"`
}

func (s *Store) Enrollment() (Enrollment, bool, error) {
	var enrollment Enrollment
	witness, err := readFrozen(filepath.Join(s.Directory(), "enrollment-initialized"), definitionLimit)
	initialized := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return enrollment, false, err
	}
	if initialized && !bytes.Equal(witness, []byte(enrollmentWitness)) {
		return enrollment, false, ErrConflict
	}
	data, err := readFrozen(filepath.Join(s.Directory(), "enrollment.json"), maxRecordBytes)
	if errors.Is(err, os.ErrNotExist) {
		if initialized {
			return enrollment, false, fmt.Errorf("root update enrollment is missing from its initialized store")
		}
		return enrollment, false, nil
	}
	if err != nil {
		return enrollment, false, err
	}
	if !initialized {
		return enrollment, false, fmt.Errorf("root update enrollment witness is missing")
	}
	if err := canonicalMetadata(data, &enrollment); err != nil {
		return enrollment, false, err
	}
	bin, err := repocontract.RuntimeHomeEntryPath(s.home, repocontract.HomeKeyBin)
	if err != nil {
		return enrollment, false, err
	}
	unit, _ := platform.CoreUnitByID(platform.CoreUnitRuntimeSupervisor)
	if enrollment.SchemaVersion != enrollmentSchemaVersion || enrollment.Home != s.home || enrollment.Executable != filepath.Join(bin, "vrooli") || enrollment.UnitName != unit.Systemd || enrollment.UpdatedUnixNano <= 0 {
		return enrollment, false, ErrConflict
	}
	return enrollment, true, nil
}

// ConfigureEnrollment takes the operation lock before the caller's install
// lock. qualify must retain its install lock until the returned release runs;
// enabling must prove current native recovery readiness under that lock.
func (s *Store) ConfigureEnrollment(ctx context.Context, enabled bool, qualify func(context.Context) (func(), error)) (Enrollment, error) {
	if qualify == nil {
		return Enrollment{}, ErrConflict
	}
	release, err := s.lock(ctx)
	if err != nil {
		return Enrollment{}, err
	}
	defer release()
	if _, _, err := s.Enrollment(); err != nil {
		return Enrollment{}, err
	}
	if err := s.GuardOrdinaryMutation(ctx); err != nil {
		return Enrollment{}, err
	}
	releaseInstall, err := qualify(ctx)
	if err != nil {
		return Enrollment{}, err
	}
	defer releaseInstall()
	if err := ctx.Err(); err != nil {
		return Enrollment{}, err
	}
	bin, err := repocontract.RuntimeHomeEntryPath(s.home, repocontract.HomeKeyBin)
	if err != nil {
		return Enrollment{}, err
	}
	unit, _ := platform.CoreUnitByID(platform.CoreUnitRuntimeSupervisor)
	e := Enrollment{SchemaVersion: enrollmentSchemaVersion, Home: s.home, Executable: filepath.Join(bin, "vrooli"), UnitName: unit.Systemd, Enabled: enabled, UpdatedUnixNano: time.Now().UnixNano()}
	data, err := json.Marshal(e)
	if err != nil {
		return Enrollment{}, err
	}
	if err := s.write(filepath.Join(s.Directory(), "enrollment-initialized"), []byte(enrollmentWitness), 0o600); err != nil {
		return Enrollment{}, err
	}
	if err := s.write(filepath.Join(s.Directory(), "enrollment.json"), data, 0o600); err != nil {
		return Enrollment{}, err
	}
	return e, nil
}
