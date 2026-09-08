package agentchat_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	agentbrief "github.com/vrooli/agentbrief-go"
	apidb "github.com/vrooli/api-core/database"

	"portal/internal/agentchat"
	internalbrief "portal/internal/brief"
	internalchat "portal/internal/chat"
	localdb "portal/internal/database"
	"portal/internal/integrations/agentmanager"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

type fakeAgentManager struct {
	findCalls      int
	stopCalls      int
	foundAdmission string
	startCalls     int
	startHook      func()
	startInput     agentmanager.StartInput
	uploaded       [][]byte
	events         []agentmanager.ActivityEvent
}

type fakeBriefProvider struct {
	record internalbrief.Record
	input  internalbrief.BuildInput
	uses   []internalbrief.UseInput
}

func (f *fakeBriefProvider) Build(_ context.Context, input internalbrief.BuildInput) (internalbrief.Record, error) {
	f.input = input
	return f.record, nil
}

func (f *fakeBriefProvider) Get(_ context.Context, id string) (internalbrief.Record, error) {
	if id != f.record.ID {
		return internalbrief.Record{}, sql.ErrNoRows
	}
	return f.record, nil
}

func (f *fakeBriefProvider) RecordUse(_ context.Context, input internalbrief.UseInput) (bool, error) {
	for _, use := range f.uses {
		if use == input {
			return false, nil
		}
	}
	f.uses = append(f.uses, input)
	return true, nil
}

func (f *fakeAgentManager) UploadAttachment(_ context.Context, content []byte, _ string, _ string) (string, error) {
	f.uploaded = append(f.uploaded, append([]byte(nil), content...))
	return "agent-attachment-1", nil
}

func (f *fakeAgentManager) Start(_ context.Context, input agentmanager.StartInput) (agentmanager.Session, error) {
	f.startCalls++
	if f.startHook != nil {
		f.startHook()
	}
	f.startInput = input
	return agentmanager.Session{TaskID: "task-1", RunID: "run-1"}, nil
}

func (f *fakeAgentManager) StreamRunEvents(_ context.Context, runID string, emit func(agentmanager.ActivityEvent) error) error {
	for _, ev := range f.events {
		ev.RunID = runID
		if err := emit(ev); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeAgentManager) FindAdmission(_ context.Context, admission string) (agentmanager.Session, error) {
	f.findCalls++
	f.foundAdmission = admission
	return agentmanager.Session{TaskID: "task-1", RunID: "run-1"}, nil
}

func (f *fakeAgentManager) Run(context.Context, string) (agentmanager.RunState, error) {
	return agentmanager.RunState{RunID: "run-1", Status: "running"}, nil
}

func (f *fakeAgentManager) Stop(context.Context, string) (agentmanager.RunState, error) {
	f.stopCalls++
	return agentmanager.RunState{RunID: "run-1", Status: "cancelled", Terminal: true}, nil
}

func newChatService(t *testing.T) (*internalchat.Service, *sql.DB) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalchat.Schema),
		apidb.SchemaProviderFunc(agentchat.Schema),
	))
	repo := internalchat.NewSQLiteRepository(d, scheduletest.New(time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)))
	return internalchat.NewService(repo), d
}

func TestStreamStartsAgentRunAndPersistsAgentMessage(t *testing.T) {
	ctx := context.Background()
	chatSvc, d := newChatService(t)
	t.Cleanup(func() { require.NoError(t, d.Close()) })
	chat, err := chatSvc.CreateChat(ctx, internalchat.CreateChatInput{
		Title:        "Agent chat",
		Mode:         internalchat.ChatModeAgent,
		AgentHarness: internalchat.AgentHarnessCodex,
	})
	require.NoError(t, err)
	user, err := chatSvc.SendUserMessage(ctx, internalchat.SendMessageInput{
		ChatID:  chat.ID,
		Content: "Update the portal tests",
	})
	require.NoError(t, err)
	fake := &fakeAgentManager{events: []agentmanager.ActivityEvent{
		{Kind: agentmanager.EventKindProgress, Text: "Agent progress 50%"},
		{Kind: agentmanager.EventKindDone, Text: "Agent run complete", Done: true},
	}}
	svc := agentchat.NewService(agentchat.Config{Chat: chatSvc, AgentManager: fake, Runs: agentchat.NewSQLiteRepository(d)})
	var streamed []agentmanager.ActivityEvent

	result, err := svc.Stream(ctx, agentchat.StreamInput{
		ChatID:        chat.ID,
		FromMessageID: user.ID,
	}, func(ev agentmanager.ActivityEvent) error {
		streamed = append(streamed, ev)
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, "Update the portal tests", fake.startInput.Prompt)
	binding, bindErr := agentchat.NewSQLiteRepository(d).Get(ctx, chat.ID, user.ID)
	require.NoError(t, bindErr)
	require.Equal(t, binding.ID, fake.startInput.AdmissionID)
	require.Equal(t, internalchat.RoleAgent, result.Message.Role)
	require.Equal(t, user.ID, result.Message.ParentMessageID)
	require.Contains(t, result.Message.Content, "Agent progress 50%")
	require.Len(t, streamed, 3)

	messages, leaf, err := chatSvc.GetTree(ctx, chat.ID)
	require.NoError(t, err)
	require.Equal(t, result.Message.ID, leaf)
	require.Len(t, messages, 2)
	require.Equal(t, internalchat.RoleAgent, messages[1].Role)
}

func TestStreamBuildsAgentBriefFromTheSelectedUserMessage(t *testing.T) {
	ctx := context.Background()
	chatSvc, d := newChatService(t)
	chat, err := chatSvc.CreateChat(ctx, internalchat.CreateChatInput{
		Title:        "Agent brief",
		Mode:         internalchat.ChatModeAgent,
		AgentHarness: internalchat.AgentHarnessClaudeCode,
	})
	require.NoError(t, err)
	user, err := chatSvc.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Inspect the selected implementation path"})
	require.NoError(t, err)
	fakeAgent := &fakeAgentManager{events: []agentmanager.ActivityEvent{{Kind: agentmanager.EventKindDone, Text: "done", Done: true}}}
	briefs := &fakeBriefProvider{record: internalbrief.Record{ID: "agent-brief-1", Rendered: "<vrooli-context-brief>agent data</vrooli-context-brief>"}}
	svc := agentchat.NewService(agentchat.Config{Chat: chatSvc, AgentManager: fakeAgent, Runs: agentchat.NewSQLiteRepository(d), Briefs: briefs})

	result, err := svc.Stream(ctx, agentchat.StreamInput{ChatID: chat.ID, FromMessageID: user.ID}, func(agentmanager.ActivityEvent) error { return nil })

	require.NoError(t, err)
	require.Equal(t, "agent-brief-1", result.BriefID)
	require.Equal(t, "Inspect the selected implementation path", briefs.input.Prompt)
	require.Equal(t, string(internalchat.AgentHarnessClaudeCode), briefs.input.Harness)
	require.Contains(t, fakeAgent.startInput.Prompt, "agent data")
	require.Contains(t, fakeAgent.startInput.Prompt, "Inspect the selected implementation path")
}

func TestStreamRecordsAgentBriefReferenceOnlyForExactPath(t *testing.T) {
	ctx := context.Background()
	chatSvc, d := newChatService(t)
	chat, err := chatSvc.CreateChat(ctx, internalchat.CreateChatInput{Title: "Agent references", Mode: internalchat.ChatModeAgent})
	require.NoError(t, err)
	user, err := chatSvc.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Inspect the docs"})
	require.NoError(t, err)
	briefs := &fakeBriefProvider{record: internalbrief.Record{ID: "agent-reference", Items: []agentbrief.Item{{Title: "Runbook", Path: "docs/portal/runbook.md"}}}}
	fake := &fakeAgentManager{events: []agentmanager.ActivityEvent{{Kind: agentmanager.EventKindDone, Text: "See docs/portal/runbook.md", Done: true}}}
	svc := agentchat.NewService(agentchat.Config{Chat: chatSvc, AgentManager: fake, Runs: agentchat.NewSQLiteRepository(d), Briefs: briefs, BriefStore: briefs})
	_, err = svc.Stream(ctx, agentchat.StreamInput{ChatID: chat.ID, FromMessageID: user.ID}, func(agentmanager.ActivityEvent) error { return nil })
	require.NoError(t, err)
	require.Equal(t, []internalbrief.UseInput{{BriefID: "agent-reference", ItemIndex: 0, Kind: "REFERENCED"}}, briefs.uses)
}

type fakeImages struct{ pixels [][]byte }

func (f fakeImages) ResolveMessageImages(context.Context, string, string) ([][]byte, error) {
	return f.pixels, nil
}

func TestStreamUploadsOwnedRenderedImagesBeforeAgentStart(t *testing.T) {
	ctx := context.Background()
	chatSvc, d := newChatService(t)
	chat, err := chatSvc.CreateChat(ctx, internalchat.CreateChatInput{Title: "Agent image", Mode: internalchat.ChatModeAgent})
	require.NoError(t, err)
	message, err := chatSvc.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Describe this"})
	require.NoError(t, err)
	fake := &fakeAgentManager{events: []agentmanager.ActivityEvent{{Kind: agentmanager.EventKindDone, Text: "done"}}}
	svc := agentchat.NewService(agentchat.Config{Chat: chatSvc, AgentManager: fake, Runs: agentchat.NewSQLiteRepository(d), Images: fakeImages{pixels: [][]byte{[]byte("png")}}})
	_, err = svc.Stream(ctx, agentchat.StreamInput{ChatID: chat.ID, FromMessageID: message.ID}, func(agentmanager.ActivityEvent) error { return nil })
	require.NoError(t, err)
	require.Equal(t, []string{"agent-attachment-1"}, fake.startInput.AttachmentIDs)
	require.Equal(t, [][]byte{[]byte("png")}, fake.uploaded)
}

func TestAdmissionSurvivesCancelledRendererAndRejectsDuplicateLaunch(t *testing.T) {
	chatSvc, d := newChatService(t)
	ctx := context.Background()
	chat, err := chatSvc.CreateChat(ctx, internalchat.CreateChatInput{Title: "durable run", Mode: internalchat.ChatModeAgent})
	require.NoError(t, err)
	message, err := chatSvc.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "bounded fixture"})
	require.NoError(t, err)
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	fake := &fakeAgentManager{startHook: cancel}
	repo := agentchat.NewSQLiteRepository(d)
	svc := agentchat.NewService(agentchat.Config{Chat: chatSvc, AgentManager: fake, Runs: repo})
	_, err = svc.Stream(streamCtx, agentchat.StreamInput{ChatID: chat.ID, FromMessageID: message.ID}, func(agentmanager.ActivityEvent) error { return errors.New("renderer disconnected") })
	require.Error(t, err)
	// Reconstruct repositories and services: identities live in SQL, not a process map.
	recovered := agentchat.NewSQLiteRepository(d)
	binding, err := recovered.Get(ctx, chat.ID, message.ID)
	require.NoError(t, err)
	require.Equal(t, "run-1", binding.RunID)
	require.Equal(t, "task-1", binding.TaskID)
	second := agentchat.NewService(agentchat.Config{Chat: chatSvc, AgentManager: fake, Runs: recovered})
	_, err = second.Stream(ctx, agentchat.StreamInput{ChatID: chat.ID, FromMessageID: message.ID}, func(agentmanager.ActivityEvent) error { return nil })
	require.ErrorIs(t, err, agentchat.ErrAlreadyAdmitted)
	require.Equal(t, 1, fake.startCalls)
	require.NoError(t, recovered.Bind(ctx, binding.ID, agentmanager.Session{RunID: "run-1", TaskID: "task-1"}))
	require.ErrorIs(t, recovered.Bind(ctx, binding.ID, agentmanager.Session{RunID: "other", TaskID: "task-1"}), agentchat.ErrBindingConflict)
	_, err = recovered.Get(ctx, "different-chat", message.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUnboundAdmissionCannotBeSilentlyRelaunched(t *testing.T) {
	_, d := newChatService(t)
	repo := agentchat.NewSQLiteRepository(d)
	b, err := repo.Reserve(context.Background(), "chat", "message")
	require.NoError(t, err)
	recovered := agentchat.NewSQLiteRepository(d)
	_, err = recovered.Reserve(context.Background(), "chat", "message")
	require.ErrorIs(t, err, agentchat.ErrAlreadyAdmitted)
	got, err := recovered.Get(context.Background(), "chat", "message")
	require.NoError(t, err)
	require.Equal(t, b.ID, got.ID)
	require.Empty(t, got.RunID)
}

func TestResolveBindsExistingAdmissionWithoutLaunchingAndScopesStop(t *testing.T) {
	ctx := context.Background()
	chatSvc, d := newChatService(t)
	chat, err := chatSvc.CreateChat(ctx, internalchat.CreateChatInput{Title: "recover"})
	require.NoError(t, err)
	msg, err := chatSvc.SendUserMessage(ctx, internalchat.SendMessageInput{ChatID: chat.ID, Content: "recover exact run"})
	require.NoError(t, err)
	repo := agentchat.NewSQLiteRepository(d)
	binding, err := repo.Reserve(ctx, chat.ID, msg.ID)
	require.NoError(t, err)
	fake := &fakeAgentManager{}
	svc := agentchat.NewService(agentchat.Config{Chat: chatSvc, AgentManager: fake, Runs: repo})
	input := agentchat.StreamInput{ChatID: chat.ID, FromMessageID: msg.ID}
	resolved, err := svc.Resolve(ctx, input)
	require.NoError(t, err)
	require.Equal(t, "run-1", resolved.RunID)
	require.Equal(t, binding.ID, fake.foundAdmission)
	require.Zero(t, fake.startCalls)
	_, err = svc.Resolve(ctx, input)
	require.NoError(t, err)
	require.Equal(t, 1, fake.findCalls)
	state, err := svc.Stop(ctx, input)
	require.NoError(t, err)
	require.True(t, state.Terminal)
	require.Equal(t, 1, fake.stopCalls)
	_, err = svc.Stop(ctx, agentchat.StreamInput{ChatID: chat.ID, FromMessageID: "other-message"})
	require.Error(t, err)
	require.Equal(t, 1, fake.stopCalls)
}

func TestAdmissionRecoveryPagesIncludeUnknownLaunchesAcrossReconstruction(t *testing.T) {
	ctx := context.Background()
	_, db := newChatService(t)
	repo := agentchat.NewSQLiteRepository(db)
	first, err := repo.Reserve(ctx, "chat-a", "message-a")
	require.NoError(t, err)
	second, err := repo.Reserve(ctx, "chat-b", "message-b")
	require.NoError(t, err)
	require.NoError(t, repo.Bind(ctx, second.ID, agentmanager.Session{TaskID: "task", RunID: "run"}))
	third, err := repo.Reserve(ctx, "chat-c", "message-c")
	require.NoError(t, err)
	fake := &fakeAgentManager{}
	service := agentchat.NewService(agentchat.Config{Runs: agentchat.NewSQLiteRepository(db), AgentManager: fake})
	page, err := service.ListAdmissions(ctx, "", 2)
	require.NoError(t, err)
	require.Len(t, page.Bindings, 2)
	require.Equal(t, first.ID, page.Bindings[0].ID)
	require.Empty(t, page.Bindings[0].RunID)
	require.Equal(t, "run", page.Bindings[1].RunID)
	require.NotEmpty(t, page.NextPageToken)
	// A new admission inserted between pages must not displace the earlier ones.
	fourth, err := repo.Reserve(ctx, "chat-d", "message-d")
	require.NoError(t, err)
	recovered := agentchat.NewService(agentchat.Config{Runs: agentchat.NewSQLiteRepository(db), AgentManager: fake})
	last, err := recovered.ListAdmissions(ctx, page.NextPageToken, 2)
	require.NoError(t, err)
	require.Len(t, last.Bindings, 2)
	require.Equal(t, third.ID, last.Bindings[0].ID)
	require.Equal(t, fourth.ID, last.Bindings[1].ID)
	require.Empty(t, last.NextPageToken)
	require.Zero(t, fake.startCalls)
	require.Zero(t, fake.findCalls)
	for _, token := range []string{"-1", "0", "01", "1x", "9223372036854775808"} {
		_, err := recovered.ListAdmissions(ctx, token, 2)
		require.ErrorIs(t, err, agentchat.ErrInvalidPage)
	}
	for _, size := range []int{-1, 101} {
		_, err := recovered.ListAdmissions(ctx, "", size)
		require.ErrorIs(t, err, agentchat.ErrInvalidPage)
	}
}

func TestAdmissionOwnershipSurvivesConversationDeletionAndScopesRecovery(t *testing.T) {
	chats, d := newChatService(t)
	legacy := context.Background()
	alice, err := internalchat.WithOwner(legacy, "alice")
	require.NoError(t, err)
	bob, err := internalchat.WithOwner(legacy, "bob")
	require.NoError(t, err)
	chat, err := chats.CreateChat(alice, internalchat.CreateChatInput{Title: "Private task"})
	require.NoError(t, err)
	message, err := chats.SendUserMessage(alice, internalchat.SendMessageInput{ChatID: chat.ID, Content: "Private work"})
	require.NoError(t, err)
	repo := agentchat.NewSQLiteRepository(d)
	binding, err := repo.Reserve(alice, chat.ID, message.ID)
	require.NoError(t, err)
	_, err = repo.Reserve(legacy, "legacy-chat", "legacy-message")
	require.NoError(t, err)
	for _, ctx := range []context.Context{legacy, bob} {
		_, err = repo.Get(ctx, chat.ID, message.ID)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.ErrorIs(t, repo.Bind(ctx, binding.ID, agentmanager.Session{TaskID: "foreign-task", RunID: "foreign-run"}), agentchat.ErrBindingConflict)
		page, err := repo.List(ctx, "", 1)
		require.NoError(t, err)
		for _, row := range page.Bindings {
			require.NotEqual(t, binding.ID, row.ID)
		}
	}
	deleted, err := chats.DeleteChat(alice, chat.ID)
	require.NoError(t, err)
	require.True(t, deleted)
	page, err := repo.List(alice, "", 1)
	require.NoError(t, err)
	require.Len(t, page.Bindings, 1)
	require.Equal(t, binding.ID, page.Bindings[0].ID)
	agent := &fakeAgentManager{}
	service := agentchat.NewService(agentchat.Config{Chat: chats, Runs: repo, AgentManager: agent})
	_, err = service.Stop(bob, agentchat.StreamInput{ChatID: chat.ID, FromMessageID: message.ID})
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Zero(t, agent.stopCalls)
	require.Zero(t, agent.findCalls)
	_, err = service.Stop(alice, agentchat.StreamInput{ChatID: chat.ID, FromMessageID: message.ID})
	require.NoError(t, err)
	require.Equal(t, 1, agent.findCalls)
	require.Equal(t, 1, agent.stopCalls)
	require.Zero(t, agent.startCalls)
}

func TestAdmissionOwnerFailureRollsBackReservation(t *testing.T) {
	_, d := newChatService(t)
	ctx, err := internalchat.WithOwner(context.Background(), "alice")
	require.NoError(t, err)
	_, err = d.Exec(`CREATE TRIGGER fail_admission_owner BEFORE INSERT ON agent_chat_run_owners BEGIN SELECT RAISE(ABORT,'injected owner failure'); END`)
	require.NoError(t, err)
	repo := agentchat.NewSQLiteRepository(d)
	_, err = repo.Reserve(ctx, "chat", "message")
	require.Error(t, err)
	var count int
	require.NoError(t, d.QueryRow("SELECT COUNT(*) FROM agent_chat_runs").Scan(&count))
	require.Zero(t, count)
}
