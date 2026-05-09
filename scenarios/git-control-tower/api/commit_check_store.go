package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type CommitCheckRecorder interface {
	Save(ctx context.Context, repoPath, commitHash string, run CommitCheckRun) error
}

type CommitCheckReader interface {
	ListForCommits(ctx context.Context, repoPath string, hashes []string) (map[string][]CommitCheckRun, error)
}

type CommitCheckStore struct {
	db *sql.DB
}

func NewCommitCheckStore(db *sql.DB) *CommitCheckStore {
	return &CommitCheckStore{db: db}
}

func (s *CommitCheckStore) Save(ctx context.Context, repoPath, commitHash string, run CommitCheckRun) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("commit check store not configured")
	}
	repoPath = strings.TrimSpace(repoPath)
	commitHash = strings.TrimSpace(commitHash)
	if repoPath == "" {
		return fmt.Errorf("repo path is required")
	}
	if commitHash == "" {
		return fmt.Errorf("commit hash is required")
	}
	if run.Timestamp.IsZero() {
		run.Timestamp = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO git_commit_check_runs (
			repo_path, commit_hash, kind, status, command, exit_code, summary,
			stdout, stderr, duration_ms, timestamp
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, repoPath, commitHash, run.Kind, run.Status, run.Command, run.ExitCode, run.Summary,
		run.Stdout, run.Stderr, run.DurationMs, run.Timestamp.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save commit check run: %w", err)
	}
	return nil
}

func (s *CommitCheckStore) ListForCommits(ctx context.Context, repoPath string, hashes []string) (map[string][]CommitCheckRun, error) {
	result := make(map[string][]CommitCheckRun)
	if s == nil || s.db == nil || len(hashes) == 0 {
		return result, nil
	}
	cleanHashes := make([]string, 0, len(hashes))
	seen := map[string]struct{}{}
	for _, hash := range hashes {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}
		cleanHashes = append(cleanHashes, hash)
	}
	if len(cleanHashes) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(cleanHashes))
	args := make([]any, 0, len(cleanHashes)+1)
	args = append(args, strings.TrimSpace(repoPath))
	for i, hash := range cleanHashes {
		placeholders[i] = "?"
		args = append(args, hash)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT commit_hash, kind, status, command, exit_code, summary, stdout, stderr, duration_ms, timestamp
		FROM git_commit_check_runs
		WHERE repo_path = ? AND commit_hash IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY id ASC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list commit check runs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			hash         string
			run          CommitCheckRun
			timestampRaw string
			stdout       sql.NullString
			stderr       sql.NullString
		)
		if err := rows.Scan(&hash, &run.Kind, &run.Status, &run.Command, &run.ExitCode, &run.Summary, &stdout, &stderr, &run.DurationMs, &timestampRaw); err != nil {
			return nil, fmt.Errorf("scan commit check run: %w", err)
		}
		if stdout.Valid {
			run.Stdout = stdout.String
		}
		if stderr.Valid {
			run.Stderr = stderr.String
		}
		if parsed, err := time.Parse(time.RFC3339Nano, timestampRaw); err == nil {
			run.Timestamp = parsed
		}
		result[hash] = append(result[hash], run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate commit check runs: %w", err)
	}
	return result, nil
}
