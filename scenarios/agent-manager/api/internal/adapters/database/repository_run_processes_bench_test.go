package database

import (
	"context"
	"fmt"
	"io"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func BenchmarkManagedRunProcessCandidates(b *testing.B) {
	sqlDB, err := sqlx.Connect("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(ON)", uuid.NewString()))
	if err != nil {
		b.Fatal(err)
	}
	defer sqlDB.Close()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	db := &DB{DB: sqlDB, log: logger}
	if err := db.initSchema(); err != nil {
		b.Fatal(err)
	}
	repos := NewRepositories(db, logger)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "candidate benchmark", ScopePath: "/fixture", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 1131; i++ {
		status, mode, pid := domain.RunStatusComplete, domain.ExecutionModeCodecPipe, 0
		if i < 100 {
			status, mode, pid = domain.RunStatusRunning, domain.ExecutionModeAttached, 1234
		}
		if i >= 100 && i < 131 {
			status, pid = domain.RunStatusNeedsReview, 5678
		}
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: status, ExecutionMode: mode, RunnerPID: pid, Phase: domain.RunPhaseQueued, ApprovalState: domain.ApprovalStateNone}
		if err := repos.Runs.Create(ctx, run); err != nil {
			b.Fatal(err)
		}
	}
	for _, projection := range []bool{false, true} {
		name := "rich-list"
		if projection {
			name = "process-projection"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				count := 0
				if projection {
					rows, err := repos.Runs.(repository.ManagedRunProcessReader).ListManagedRunProcesses(ctx)
					if err != nil {
						b.Fatal(err)
					}
					count = len(rows)
				} else {
					for offset := 0; ; offset += 100 {
						rows, err := repos.Runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 100, Offset: offset}, NonTerminalOnly: true})
						if err != nil {
							b.Fatal(err)
						}
						for _, row := range rows {
							if !row.Status.IsTerminal() && row.RunnerPID > 1 && row.ExecutionMode.Normalized() != domain.ExecutionModeAttached {
								count++
							}
						}
						if len(rows) < 100 {
							break
						}
					}
				}
				if count != 31 {
					b.Fatalf("candidate population changed: %d", count)
				}
			}
		})
	}
}
