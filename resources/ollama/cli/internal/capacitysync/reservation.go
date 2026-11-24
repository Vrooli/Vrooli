package capacitysync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/platform-go"
)

type ModelReservation struct {
	ID                    string    `json:"id"`
	Role                  string    `json:"role"`
	Model                 string    `json:"model"`
	ClaimID               string    `json:"claim_id,omitempty"`
	RequestedBytes        int64     `json:"requested_bytes"`
	CapacityBytes         int64     `json:"capacity_bytes,omitempty"`
	AdmissionBytes        int64     `json:"admission_bytes,omitempty"`
	ReleaseClaimOnStopped bool      `json:"release_claim_on_stopped,omitempty"`
	AdmissionPending      bool      `json:"admission_pending,omitempty"`
	ExpiresAt             time.Time `json:"expires_at"`
}

type ReservationStore struct {
	Path string
	Now  func() time.Time
}

func (s ReservationStore) Create(role, model string, bytes int64, ttl time.Duration) (ModelReservation, error) {
	role, model = strings.TrimSpace(role), strings.TrimSpace(model)
	if role == "" || model == "" || bytes <= 0 || ttl <= 0 {
		return ModelReservation{}, errors.New("model reservation requires role, model, positive VRAM bytes, and TTL")
	}
	unlock, err := s.lock()
	if err != nil {
		return ModelReservation{}, err
	}
	defer unlock()
	if current, err := s.Read(); err != nil {
		return ModelReservation{}, err
	} else if current != nil {
		return ModelReservation{}, fmt.Errorf("model reservation %s for %s is already active", current.ID, current.Role)
	}
	id, err := randomReservationID()
	if err != nil {
		return ModelReservation{}, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	reservation := ModelReservation{ID: id, Role: role, Model: model, RequestedBytes: bytes, ExpiresAt: now().UTC().Add(ttl)}
	if err := s.write(reservation); err != nil {
		return ModelReservation{}, err
	}
	return reservation, nil
}

func (s ReservationStore) Read() (*ModelReservation, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Ollama model capacity reservation: %w", err)
	}
	var reservation ModelReservation
	if err := json.Unmarshal(data, &reservation); err != nil {
		return nil, fmt.Errorf("decode Ollama model capacity reservation: %w", err)
	}
	if reservation.ID == "" || reservation.Role == "" || reservation.Model == "" || reservation.RequestedBytes <= 0 || reservation.ExpiresAt.IsZero() {
		return nil, errors.New("Ollama model capacity reservation is incomplete")
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	if !now().Before(reservation.ExpiresAt) {
		return nil, nil
	}
	return &reservation, nil
}

func (s ReservationStore) Confirm(id, claimID string, capacityBytes, admissionBytes int64, releaseClaimOnStopped bool) (ModelReservation, error) {
	unlock, err := s.lock()
	if err != nil {
		return ModelReservation{}, err
	}
	defer unlock()
	reservation, err := s.Read()
	if err != nil {
		return ModelReservation{}, err
	}
	if reservation == nil || reservation.ID != strings.TrimSpace(id) {
		return ModelReservation{}, errors.New("Ollama model capacity reservation is missing, expired, or has a different ID")
	}
	if capacityBytes < reservation.RequestedBytes {
		return ModelReservation{}, fmt.Errorf("admitted Ollama capacity %d bytes is below the model requirement %d bytes", capacityBytes, reservation.RequestedBytes)
	}
	if admissionBytes < 0 || admissionBytes > capacityBytes {
		return ModelReservation{}, fmt.Errorf("admitted Ollama target increment must be non-negative and no greater than the aggregate capacity")
	}
	if strings.TrimSpace(claimID) == "" {
		return ModelReservation{}, errors.New("admitted Ollama capacity claim ID is required")
	}
	reservation.ClaimID = strings.TrimSpace(claimID)
	reservation.CapacityBytes = capacityBytes
	reservation.AdmissionBytes = admissionBytes
	reservation.ReleaseClaimOnStopped = releaseClaimOnStopped
	reservation.AdmissionPending = false
	if err := s.write(*reservation); err != nil {
		return ModelReservation{}, err
	}
	return *reservation, nil
}

// BeginAdmission durably marks the point before the capacity broker can
// replace the owner claim. If the process stops before its claim ID is saved,
// cleanup knows the handoff was interrupted and will not infer an ID.
func (s ReservationStore) BeginAdmission(id string) (ModelReservation, error) {
	unlock, err := s.lock()
	if err != nil {
		return ModelReservation{}, err
	}
	defer unlock()
	reservation, err := s.Read()
	if err != nil {
		return ModelReservation{}, err
	}
	if reservation == nil || reservation.ID != strings.TrimSpace(id) {
		return ModelReservation{}, errors.New("Ollama model capacity reservation is missing, expired, or has a different ID")
	}
	reservation.AdmissionPending = true
	if err := s.write(*reservation); err != nil {
		return ModelReservation{}, err
	}
	return *reservation, nil
}

func (s ReservationStore) Remove(id string) error {
	unlock, err := s.lock()
	if err != nil {
		return err
	}
	defer unlock()
	reservation, err := s.Read()
	if err != nil {
		return err
	}
	if reservation == nil {
		return nil
	}
	if reservation.ID != strings.TrimSpace(id) {
		return errors.New("Ollama model capacity reservation ID does not match")
	}
	if err := os.Remove(s.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove Ollama model capacity reservation: %w", err)
	}
	return nil
}

func (s ReservationStore) lock() (func(), error) {
	if strings.TrimSpace(s.Path) == "" {
		return nil, errors.New("Ollama model reservation path is required")
	}
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create Ollama capacity state directory: %w", err)
	}
	unlock, err := platform.AcquireFileLockContext(context.Background(), s.Path+".lock")
	if err != nil {
		return nil, fmt.Errorf("lock Ollama model capacity reservation: %w", err)
	}
	return unlock, nil
}

// LockSync serializes the shared companion's full observation and claim
// reconciliation with a reservation handoff.
func (s ReservationStore) LockSync(ctx context.Context) (func(), error) {
	if strings.TrimSpace(s.Path) == "" {
		return nil, errors.New("Ollama model reservation path is required")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return nil, fmt.Errorf("create Ollama capacity state directory: %w", err)
	}
	unlock, err := platform.AcquireFileLockContext(ctx, s.Path+".sync.lock")
	if err != nil {
		return nil, fmt.Errorf("lock Ollama capacity synchronization: %w", err)
	}
	return unlock, nil
}

func (s ReservationStore) write(reservation ModelReservation) error {
	if strings.TrimSpace(s.Path) == "" {
		return errors.New("Ollama model capacity reservation path is required")
	}
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create Ollama capacity state directory: %w", err)
	}
	data, err := json.Marshal(reservation)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".model-capacity-*")
	if err != nil {
		return fmt.Errorf("create Ollama reservation temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, s.Path); err != nil {
		return fmt.Errorf("commit Ollama model capacity reservation: %w", err)
	}
	return nil
}

func randomReservationID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("create Ollama reservation identity: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
