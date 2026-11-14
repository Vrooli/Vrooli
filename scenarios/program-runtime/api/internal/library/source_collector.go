package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/contracts"
	"program-runtime/internal/localworkbridge"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
)

// SourceCollector is an inert, in-process finite adapter. Construction registers
// no route, starts no service and reads no source/credentials. A deployment must
// supply the joined authority/broker invalidation boundary and a confined runner.
// The session manager must be the sole invalidation owner of its repository.
type SourceCollector struct {
	library  *Repository
	sessions *sessions.Manager
	programs *programs.Service
	boundary programs.ServiceAdmissionBoundary
}

// NewSourceCollector wires the canonical one-shot owner directly to the program
// service, rather than trusting arbitrary session validation callbacks. Store
// must be a trusted durable program database, inaccessible to the worker.
func NewSourceCollector(repo *Repository, manager *sessions.Manager, options programs.Options, boundary programs.ServiceAdmissionBoundary) (*SourceCollector, error) {
	if repo == nil || manager == nil || options.Store == nil || options.Runner == nil || boundary == nil {
		return nil, localworkbridge.ErrSourceAuthority
	}
	c := &SourceCollector{library: repo, sessions: manager, boundary: boundary}
	options.AuthorizeSession = manager.AuthorizeSubmission
	options.ValidateSession = func(id string) bool {
		s, err := manager.Get(context.Background(), id)
		return err == nil && s.State == "running"
	}
	options.ExecutionBudget = func(id string) (programs.ExecutionLimits, error) {
		b, err := manager.ExecutionBudget(context.Background(), id)
		return programs.ExecutionLimits{Wall: b.WallBudget - b.WallConsumed, CPU: b.CPUBudget - b.CPUConsumed}, err
	}
	options.ChargeExecution = func(id string, wall, cpu time.Duration) error {
		return manager.ChargeExecution(context.Background(), id, wall, cpu)
	}
	previous := options.OnTerminal
	options.OnTerminal = func(ctx context.Context, id string) error {
		p, err := c.programs.Get(ctx, id)
		if err != nil {
			return err
		}
		_, err = manager.Delete(ctx, p.GetSessionId(), "finite collector complete")
		if previous != nil {
			err = errors.Join(err, previous(ctx, id))
		}
		return err
	}
	c.programs = programs.NewService(options)
	return c, nil
}

type collectorSessionBoundary struct {
	authority programs.ServiceAdmissionBoundary
	manager   *sessions.Manager
	id        string
}

func (b collectorSessionBoundary) Within(ctx context.Context, fn func(context.Context) error) error {
	return b.authority.Within(ctx, func(inner context.Context) error { return b.manager.WithLiveSession(inner, b.id, fn) })
}

func (c *SourceCollector) context(ctx context.Context, p *localworkbridge.SourcePermit, operation string) (context.Context, error) {
	if c == nil || p == nil {
		return nil, localworkbridge.ErrSourceAuthority
	}
	if err := p.Recheck(ctx); err != nil {
		return nil, err
	}
	if p.Call().Operation != operation {
		return nil, localworkbridge.ErrSourceAuthority
	}
	return programs.WithSourcePermit(ctx, p, c.boundary)
}

// prepare uses only the archived, content-addressed canonical declaration and
// policy inputs. No caller source, grant, principal or mutable current alias is
// accepted. Input defaults use the same contract owner as declared execution.
func (c *SourceCollector) prepare(ctx context.Context, p *localworkbridge.SourcePermit) (contracts.Contract, string, programs.Identity, error) {
	name, digest, raw, err := p.CollectorSpec()
	if err != nil {
		return contracts.Contract{}, "", programs.Identity{}, err
	}
	contract, err := c.library.GetDeclaredArtifact(ctx, name, digest)
	if err != nil {
		return contract, "", programs.Identity{}, err
	}
	return prepareSourceCollector(contract, raw, p.Call())
}
func prepareSourceCollector(contract contracts.Contract, raw json.RawMessage, call localworkbridge.SourceCall) (contracts.Contract, string, programs.Identity, error) {
	fail := func() (contracts.Contract, string, programs.Identity, error) {
		return contract, "", programs.Identity{}, localworkbridge.ErrSourceAuthority
	}
	if contract.ID != localworkbridge.SourceCollectorName || contract.ValidationError != "" || contract.SourceMissing || contract.Source == "" || contract.WallMS <= 0 || contract.WallMS > 120000 || (contract.OutputBytes != 4096 && contract.OutputBytes != 65536) || contract.LearningTask != nil || len(contract.Learning.Verbs) > 0 {
		return fail()
	}
	for _, b := range contract.Bindings {
		if b.Effect != "read" {
			return fail()
		}
	}
	var provided map[string]any
	if json.Unmarshal(raw, &provided) != nil || provided == nil {
		return fail()
	}
	resolved, err := contract.ResolveInputs(provided)
	if err != nil {
		return contract, "", programs.Identity{}, err
	}
	encoded, err := json.Marshal(resolved)
	if err != nil {
		return contract, "", programs.Identity{}, err
	}
	var declaration struct {
		Helpers struct {
			Program string `json:"program"`
		} `json:"helpers"`
		Outputs json.RawMessage `json:"outputs"`
	}
	if json.Unmarshal(contract.Declaration, &declaration) != nil {
		return fail()
	}
	identity := programs.Identity{ProgramName: contract.ID, ProgramDigest: contract.Digest, IdempotencyKey: call.RequestKey, AdmissionDeadline: call.AdmissionDeadline, OutputSchema: contract.OutputSchema, HelperContract: &programs.HelperContract{Version: declaration.Helpers.Program, Outputs: declaration.Outputs}}
	source := "import json\ninputs = json.loads(" + strconv.Quote(string(encoded)) + ")\n# declared-program generated input preamble\n" + contract.Source
	return contract, source, identity, nil
}

func (c *SourceCollector) AdmitCollector(ctx context.Context, p *localworkbridge.SourcePermit) (localworkbridge.SourceBrokerProgram, error) {
	owned, err := c.context(ctx, p, "local-work.collector.admit")
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	contract, source, identity, err := c.prepare(owned, p)
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	const provenance = programsv1.Provenance_PROVENANCE_AGENT
	replay, err := c.programs.ReplayDeclared(owned, source, provenance, contract.OutputBytes == 65536, identity, programs.Caller{})
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	if replay != nil {
		return c.programs.SourceProgram(owned, replay.GetId())
	}
	session, admission, err := c.sessions.CreateOneShot(owned, "finite-collector:"+contract.ID, "", nil, 0, 0, contract.WallMS, 0)
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	admission, err = programs.WithSourcePermit(admission, p, collectorSessionBoundary{c.boundary, c.sessions, session.ID})
	if err != nil {
		_, cleanup := c.sessions.Delete(context.Background(), session.ID, "finite collector admission denied")
		return localworkbridge.SourceBrokerProgram{}, errors.Join(err, cleanup)
	}
	program, _, err := c.programs.SubmitDeclared(admission, session.ID, source, provenance, contract.OutputBytes == 65536, false, identity, programs.Caller{}, true)
	if err != nil || program.GetSessionId() != session.ID || programs.IsTerminal(program.GetStatus()) {
		_, cleanup := c.sessions.Delete(context.Background(), session.ID, "finite collector submission finished")
		if err != nil {
			return localworkbridge.SourceBrokerProgram{}, errors.Join(err, cleanup)
		}
		if cleanup != nil {
			return localworkbridge.SourceBrokerProgram{}, cleanup
		}
	}
	return c.programs.SourceProgram(owned, program.GetId())
}

func (c *SourceCollector) ObserveCollector(ctx context.Context, p *localworkbridge.SourcePermit) (localworkbridge.SourceBrokerProgram, error) {
	owned, err := c.context(ctx, p, "local-work.collector.observe")
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	return c.programs.SourceProgram(owned, programs.SourceAdmissionID(p.Call().RequestKey))
}
func (c *SourceCollector) WaitCollector(ctx context.Context, p *localworkbridge.SourcePermit) (localworkbridge.SourceBrokerProgram, error) {
	owned, err := c.context(ctx, p, "local-work.collector.wait")
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	call := p.Call()
	if call.TimeoutMillis < 1 || call.TimeoutMillis > 90000 {
		return localworkbridge.SourceBrokerProgram{}, localworkbridge.ErrSourceAuthority
	}
	initial, err := c.programs.SourceProgram(owned, call.ProgramID)
	if err != nil || !initial.Found {
		return initial, err
	}
	_, _, err = c.programs.Wait(owned, call.ProgramID, time.Duration(call.TimeoutMillis)*time.Millisecond)
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	return c.programs.SourceProgram(owned, call.ProgramID)
}

// WithTerminal holds the actual supplied authority/broker boundary through the
// native receipt owner's completion commit. Terminal rows outlive their reclaimed
// one-shot session. Callers must acquire native authority before this boundary.
func (c *SourceCollector) WithTerminal(ctx context.Context, p *localworkbridge.SourcePermit, fn func(context.Context, localworkbridge.SourceBrokerProgram) error) error {
	owned, err := c.context(ctx, p, "local-work.collector.observe")
	if err != nil {
		return err
	}
	if fn == nil {
		return localworkbridge.ErrSourceAuthority
	}
	return c.programs.WithSourceProgram(owned, programs.SourceAdmissionID(p.Call().RequestKey), func(inner context.Context, result localworkbridge.SourceBrokerProgram) error {
		switch result.Status {
		case "PROGRAM_STATUS_SUCCEEDED", "PROGRAM_STATUS_FAILED", "PROGRAM_STATUS_CANCELLED":
		default:
			return fmt.Errorf("collector is not terminal")
		}
		return fn(inner, result)
	})
}

// RecoverCollectorAfterStop is explicit startup recovery for one retained
// selection. The required lifecycle boundary must stop/join the prior executor
// and exclude replacement through cleanup. No live boundary is installed here.
func (c *SourceCollector) RecoverCollectorAfterStop(ctx context.Context, p *localworkbridge.SourcePermit, stopped programs.ServiceRecoveryBoundary) (localworkbridge.SourceBrokerProgram, error) {
	owned, err := c.context(ctx, p, "local-work.collector.observe")
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	id := programs.SourceAdmissionID(p.Call().RequestKey)
	err = c.programs.RecoverSourceAfterStop(owned, id, stopped, func(inner context.Context, sessionID string) error {
		_, err := c.sessions.Delete(inner, sessionID, "finite collector recovered after stopped owner")
		if errors.Is(err, sessions.ErrNotFound) {
			return nil
		}
		return err
	})
	if errors.Is(err, programs.ErrProgramNotFound) {
		return localworkbridge.SourceBrokerProgram{}, nil
	}
	if err != nil {
		return localworkbridge.SourceBrokerProgram{}, err
	}
	return c.programs.SourceProgram(owned, id)
}
