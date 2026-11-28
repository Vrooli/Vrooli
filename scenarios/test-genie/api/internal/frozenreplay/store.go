package frozenreplay

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"test-genie/internal/dbexec"
)

var (
	ErrNotFound        = errors.New("frozen replay receipt not found")
	ErrIdempotencyKey  = errors.New("frozen replay idempotency key reused with different input")
	idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)
)

type CreateRequest struct {
	IdempotencyKey string  `json:"idempotency_key"`
	Receipt        Receipt `json:"receipt"`
}

type CreateResult struct {
	Receipt  Receipt `json:"receipt"`
	Replayed bool    `json:"replayed"`
}

type Service struct {
	db       dbexec.Executor
	verifier EvidenceVerifier
	now      func() time.Time
}

func NewService(ctx context.Context, db dbexec.Executor, verifier EvidenceVerifier) (*Service, error) {
	if db == nil || verifier == nil {
		return nil, errors.New("frozen replay database and owner evidence verifier are required")
	}
	if _, err := db.ExecContext(ctx, Schema()); err != nil {
		return nil, fmt.Errorf("initialize frozen replay receipt store: %w", err)
	}
	return &Service{db: db, verifier: verifier, now: time.Now}, nil
}

func (s *Service) Create(ctx context.Context, request CreateRequest) (CreateResult, error) {
	key := strings.TrimSpace(request.IdempotencyKey)
	if !idempotencyPattern.MatchString(key) {
		return CreateResult{}, fmt.Errorf("%w: idempotency key must be 8-128 safe characters", ErrInvalidReceipt)
	}
	receipt, err := deriveArtifactDigests(ctx, request.Receipt, s.verifier)
	if err != nil {
		return CreateResult{}, err
	}
	request.Receipt = receipt
	inputDigest, err := fingerprintInput(request.Receipt)
	if err != nil {
		return CreateResult{}, err
	}
	if existing, existingDigest, err := s.byIdempotencyKey(ctx, key); err == nil {
		if existingDigest != inputDigest {
			return CreateResult{}, ErrIdempotencyKey
		}
		if err := Verify(existing); err != nil {
			return CreateResult{}, fmt.Errorf("stored frozen replay receipt failed integrity verification: %w", err)
		}
		return CreateResult{Receipt: existing, Replayed: true}, nil
	} else if !errors.Is(err, ErrNotFound) {
		return CreateResult{}, err
	}
	receipt, err = Adjudicate(ctx, request.Receipt, s.verifier)
	if err != nil {
		return CreateResult{}, err
	}
	receiptID := strings.TrimPrefix(receipt.ReceiptDigest, "sha256:")
	encodedReceipt, err := json.Marshal(receipt)
	if err != nil {
		return CreateResult{}, fmt.Errorf("encode replay receipt: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CreateResult{}, err
	}
	defer tx.Rollback()
	_, insertErr := tx.ExecContext(ctx, `INSERT INTO frozen_replay_receipts
		(receipt_id, idempotency_key, input_digest, receipt_json, created_at)
		VALUES (?, ?, ?, ?, ?)`, receiptID, key, inputDigest, string(encodedReceipt), s.now().UTC().Format(time.RFC3339Nano))
	if insertErr != nil {
		var existingDigest, existingJSON string
		readErr := tx.QueryRowContext(ctx, `SELECT input_digest, receipt_json FROM frozen_replay_receipts WHERE idempotency_key = ?`, key).Scan(&existingDigest, &existingJSON)
		if errors.Is(readErr, sql.ErrNoRows) {
			readErr = tx.QueryRowContext(ctx, `SELECT input_digest, receipt_json FROM frozen_replay_receipts WHERE receipt_id = ?`, receiptID).Scan(&existingDigest, &existingJSON)
		}
		if errors.Is(readErr, sql.ErrNoRows) {
			return CreateResult{}, fmt.Errorf("insert frozen replay receipt: %w", insertErr)
		}
		if readErr != nil {
			return CreateResult{}, fmt.Errorf("resolve frozen replay idempotency: %w", readErr)
		}
		if existingDigest != inputDigest {
			return CreateResult{}, ErrIdempotencyKey
		}
		var stored Receipt
		if err := json.Unmarshal([]byte(existingJSON), &stored); err != nil || Verify(stored) != nil {
			return CreateResult{}, errors.Join(errors.New("stored frozen replay receipt failed integrity verification"), err)
		}
		if err := tx.Commit(); err != nil {
			return CreateResult{}, err
		}
		return CreateResult{Receipt: stored, Replayed: true}, nil
	}
	if err := tx.Commit(); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{Receipt: receipt, Replayed: false}, nil
}

func deriveArtifactDigests(ctx context.Context, input Receipt, verifier EvidenceVerifier) (Receipt, error) {
	digester, ok := verifier.(EvidenceArtifactDigester)
	for index, evidence := range input.Evidence {
		if evidence.ArtifactDigest != "" {
			continue
		}
		if !ok {
			return Receipt{}, fmt.Errorf("%w: %s outcome lacks an owner-derived artifact digest", ErrInvalidReceipt, evidence.Variant)
		}
		digest, err := digester.ArtifactDigest(ctx, evidence)
		if err != nil {
			return Receipt{}, fmt.Errorf("%w: %s outcome artifact is unavailable from its owner record: %v", ErrInvalidReceipt, evidence.Variant, err)
		}
		input.Evidence[index].ArtifactDigest = digest
	}
	return input, nil
}

func (s *Service) byIdempotencyKey(ctx context.Context, key string) (Receipt, string, error) {
	var digest, encoded string
	err := s.db.QueryRowContext(ctx, `SELECT input_digest, receipt_json FROM frozen_replay_receipts WHERE idempotency_key = ?`, key).Scan(&digest, &encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, "", ErrNotFound
	}
	if err != nil {
		return Receipt{}, "", err
	}
	var receipt Receipt
	if err := json.Unmarshal([]byte(encoded), &receipt); err != nil {
		return Receipt{}, "", errors.Join(errors.New("stored frozen replay receipt failed integrity verification"), err)
	}
	if err := Verify(receipt); err != nil {
		return Receipt{}, "", fmt.Errorf("stored frozen replay receipt failed integrity verification: %w", err)
	}
	storedInputDigest, err := fingerprintInput(receipt)
	if err != nil || storedInputDigest != digest {
		return Receipt{}, "", errors.Join(errors.New("stored frozen replay request digest does not match its receipt"), err)
	}
	return receipt, digest, nil
}

func fingerprintInput(input Receipt) (string, error) {
	input.Version = 1
	input.Disposition = ""
	input.ReceiptDigest = ""
	input.Evidence = append([]RunEvidence(nil), input.Evidence...)
	sort.Slice(input.Evidence, func(i, j int) bool { return input.Evidence[i].Variant < input.Evidence[j].Variant })
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encode replay input: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func (s *Service) Get(ctx context.Context, receiptID string) (Receipt, error) {
	receiptID = strings.TrimPrefix(strings.TrimSpace(receiptID), "sha256:")
	if len(receiptID) != 64 {
		return Receipt{}, ErrNotFound
	}
	var encoded string
	err := s.db.QueryRowContext(ctx, `SELECT receipt_json FROM frozen_replay_receipts WHERE receipt_id = ?`, receiptID).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, ErrNotFound
	}
	if err != nil {
		return Receipt{}, err
	}
	var receipt Receipt
	if err := json.Unmarshal([]byte(encoded), &receipt); err != nil {
		return Receipt{}, errors.Join(errors.New("stored frozen replay receipt cannot be decoded"), err)
	}
	if err := Verify(receipt); err != nil {
		return Receipt{}, fmt.Errorf("stored frozen replay receipt failed integrity verification: %w", err)
	}
	return receipt, nil
}

func Schema() string {
	return `CREATE TABLE IF NOT EXISTS frozen_replay_receipts (
		receipt_id TEXT PRIMARY KEY,
		idempotency_key TEXT NOT NULL UNIQUE,
		input_digest TEXT NOT NULL,
		receipt_json TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TRIGGER IF NOT EXISTS frozen_replay_receipts_no_update
	BEFORE UPDATE ON frozen_replay_receipts BEGIN
		SELECT RAISE(ABORT, 'frozen replay receipts are immutable');
	END;
	CREATE TRIGGER IF NOT EXISTS frozen_replay_receipts_no_delete
	BEFORE DELETE ON frozen_replay_receipts BEGIN
		SELECT RAISE(ABORT, 'frozen replay receipts are immutable');
	END;`
}
