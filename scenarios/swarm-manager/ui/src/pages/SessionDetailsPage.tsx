/**
 * SessionDetailsPage — routed agent-session detail page.
 *
 * Owns route/data orchestration and assembles session-specific components.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { AlertCircle, RefreshCw, Square, Trash2 } from "lucide-react";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { Button } from "../components/ui/button";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { SessionArtifactList } from "../components/session/SessionArtifactList";
import { SessionConversation } from "../components/session/SessionConversation";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { useDeleteConfirm } from "../hooks/useDeleteConfirm";
import { SessionEventTimeline } from "../components/session/SessionEventTimeline";
import { SessionInspector } from "../components/session/SessionInspector";
import { SessionMetadata } from "../components/session/SessionMetadata";
import { SessionSectionTabs, type SessionSectionValue } from "../components/session/SessionSectionTabs";
import { useComposerImageAttachments } from "../components/composer/useComposerImageAttachments";
import { optionsToRefs } from "../components/session/context/session-context-options";
import { sessionKindAllowsContextType } from "../components/session/context/session-context-config";
import { sessionOption, startupBriefOption, type SessionContextOption } from "../components/session/context/session-context-refs";
import { clearStagedContextForSession, mergeContextOptions, peekStagedContextForSession } from "../components/session/context/pending-session-context";
import { readSessionDraft, writeSessionDraft } from "../components/session/session-draft-storage";
import { isUnfilledOpener } from "../components/session/session-starter-suggestions";
import { useAttachToSessionAction } from "../components/session/context/useAttachToSessionAction";
import { type ActionMenuItem } from "../components/ui/action-menu";
import { nodeIdForSessionArtifact } from "../components/session/session-artifact-routing";
import {
  defaultSessionInspectorSection,
  isSessionWaitingForAgent,
  SESSION_KIND_LABELS,
  TERMINAL_SESSION_STATUSES,
} from "../components/session/session-view-model";
import { formatDisplayText, formatRelativeTime } from "../lib/format-utils";
import { readChatDensity, rememberChatDensity, type ChatDensity } from "../lib/chat-density";
import type { ChatMessageView } from "../components/chat/chat-types";
import { useAgentSessionStore } from "../stores";
import { useAgentSessionEvents } from "../hooks/useAgentSessionEvents";
import { useAgentSessionPolling } from "../hooks/useAgentSessionPolling";
import { useAppBack } from "../app/routes/useAppBack";
import { backlogDetailPath, detailPathFromNodeId, goalDetailPath } from "../app/routes/route-paths";
import { useIsMobile } from "../hooks/useMediaQuery";
import type { AgentSession, AgentSessionArtifact, AgentSessionProposal, CreatableAgentSessionKind } from "../types";

/**
 * A message the operator has composed and the server has not yet confirmed.
 *
 * Without this the composer emptied on submit and nothing appeared until the
 * round trip finished, so a slow send read as a dropped one — and a failed
 * send left the operator with no record of what they had written.
 */
interface PendingSend {
  id: string;
  content: string;
  createdAt: string;
  context: SessionContextOption[];
  delivery: "sending" | "failed";
  baselineUserMessages: number;
}

export function SessionDetailsPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const closeDetail = useAppBack();
  const isMobile = useIsMobile();
  const desktopLayoutRef = useRef<HTMLDivElement>(null);
	const pendingContextOwnerRef = useRef<string | undefined>(undefined);

  const storeSession = useAgentSessionStore((s) =>
    s.sessions.find((session) => session.id === sessionId),
  );
  const loadSession = useAgentSessionStore((s) => s.loadSession);
  const loadStartupBrief = useAgentSessionStore((s) => s.loadStartupBrief);
	const changeSessionKind = useAgentSessionStore((s) => s.changeSessionKind);
  const startSession = useAgentSessionStore((s) => s.startSession);
  const continueSession = useAgentSessionStore((s) => s.continueSession);
  const uploadSessionAttachments = useAgentSessionStore((s) => s.uploadSessionAttachments ?? (() => Promise.resolve([])));
  const refreshSession = useAgentSessionStore((s) => s.refreshSession);
  const cancelSession = useAgentSessionStore((s) => s.cancelSession);
  const deleteSession = useAgentSessionStore((s) => s.deleteSession);
  const applyProposal = useAgentSessionStore((s) => s.applyProposal);
  const isMutating = useAgentSessionStore((s) => s.isMutating);
  const isRefreshing = useAgentSessionStore((s) => s.isRefreshing);
  const error = useAgentSessionStore((s) => s.error);

  useAgentSessionPolling(sessionId);

  const { data: fetchedSession, isLoading, error: queryError } = useQuery({
    queryKey: ["session", sessionId],
    queryFn: () => loadSession(sessionId ?? ""),
    enabled: !!sessionId && !storeSession,
  });

  const session: AgentSession | undefined = storeSession ?? fetchedSession;
  const attachToSession = useAttachToSessionAction(
    session ? sessionOption(session) : null,
    { currentSessionId: session?.id },
  );
  const startupBriefQuery = useQuery({
    queryKey: ["session-startup-brief", session?.id, session?.kind],
    queryFn: () => loadStartupBrief(session?.id ?? ""),
    enabled: Boolean(session?.id && session.status === "draft"),
    staleTime: 60_000,
  });
  const { events, isLoading: eventsLoading, error: eventsError, refreshEvents } = useAgentSessionEvents(session);

  const [draft, setDraft] = useState("");
  const [starterJobId, setStarterJobId] = useState<string | undefined>();
  const [pendingContext, setPendingContext] = useState<SessionContextOption[]>([]);
  const [localError, setLocalError] = useState<string | null>(null);
  const [pendingSend, setPendingSend] = useState<PendingSend | null>(null);
  const [density, setDensity] = useState<ChatDensity>(readChatDensity);
  const [inspectorCollapsed, setInspectorCollapsed] = useState(false);
  const { requestDelete: requestDeleteSession, dialogProps: deleteDialogProps } =
    useDeleteConfirm("session");
  const [mobileSection, setMobileSection] = useState<SessionSectionValue>("conversation");
  const draftSessionId = sessionId ?? "unknown";
  const sessionAttachments = useComposerImageAttachments(sessionId ? `swarm-session-attachments:${sessionId}` : "swarm-session-attachments:unknown");

  useEffect(() => {
    setDraft(readSessionDraft(draftSessionId));
    setStarterJobId(session?.starterJobId);
    setPendingContext((current) => {
      if (!session || session.status !== "draft") return [];
      const sameSession = pendingContextOwnerRef.current === session.id;
      pendingContextOwnerRef.current = session.id;
      const retained = sameSession
        ? current.filter(
            (item) => item.type !== "startup_brief" && sessionKindAllowsContextType(session.kind, item.type),
          )
        : [];
      return [startupBriefOption(session.kind), ...retained];
    });
    setPendingSend(null);
  }, [session, draftSessionId]);

  useEffect(() => {
    if (!session) return;
    const staged = peekStagedContextForSession(session.id);
    if (staged.length === 0) return;
    setPendingContext((current) => {
      const merged = mergeContextOptions(current, staged, session.kind);
      if (merged.rejected.length > 0) {
        setLocalError(merged.rejected.map(({ option, reason }) => `${option.title}: ${reason}`).join(" "));
      }
      clearStagedContextForSession(session.id, merged.applied);
      return merged.items;
    });
  }, [session]);

  useEffect(() => {
    if (!session || session.status !== "draft" || !startupBriefQuery.data) return;
    const brief = startupBriefQuery.data;
    setPendingContext((current) => {
      const withoutStartupBrief = current.filter((item) => item.type !== "startup_brief");
      return [
        // The resolved summary rides along so the chip can show what the agent
        // will actually receive. Dropping it here is what made the brief
        // unreadable from the UI even though it was already in memory.
        startupBriefOption(session.kind, brief.title, brief.selectedAt, brief.summary),
        ...withoutStartupBrief,
      ];
    });
  }, [session, startupBriefQuery.data]);

  // While a send is outstanding the composer is empty but the text is not
  // lost — it lives in pendingSend, and the failure branch persists it under
  // the same draft key. Writing the (empty) composer value here would clobber
  // that, so the debounce stands down until the send resolves.
  useEffect(() => {
    if (pendingSend) return undefined;
    const timer = window.setTimeout(() => {
      writeSessionDraft(draftSessionId, draft);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [draft, draftSessionId, pendingSend]);

  const handleDensityChange = useCallback((next: ChatDensity) => {
    setDensity(next);
    rememberChatDensity(next);
  }, []);

  const deliverMessage = useCallback(
    async (send: PendingSend) => {
      if (!session) return;
      setLocalError(null);
      try {
        const uploaded = await uploadSessionAttachments(session.id, sessionAttachments.getFiles());
        const attachmentIds = uploaded.map((attachment) => attachment.id);
        const contextRefs = optionsToRefs(send.context);
        const hasAutoStartupContext = contextRefs.some((ref) => ref.type === "startup_brief" || ref.type === "operations_briefing");
        const sendArgs = {
          sessionId: session.id,
          message: send.content,
          ...(attachmentIds.length > 0 ? { attachmentIds } : {}),
          ...(contextRefs.length > 0 ? { contextRefs } : {}),
          ...(session.status === "draft" && hasAutoStartupContext ? { autoContextPolicy: "none" as const } : {}),
          ...(session.status === "draft" && starterJobId ? { starterJobId } : {}),
        };
        if (session.status === "draft") {
          await startSession(sendArgs);
        } else {
          await continueSession(sendArgs);
        }
        sessionAttachments.clearAll();
        setPendingSend(null);
        writeSessionDraft(draftSessionId, "");
      } catch (err) {
        // Keep the message visible and recoverable rather than silently
        // dropping it, and persist the text so a refresh cannot lose it.
        setPendingSend((current) => (current && current.id === send.id ? { ...current, delivery: "failed" } : current));
        writeSessionDraft(draftSessionId, send.content);
        setLocalError(err instanceof Error ? err.message : "Unable to send session message.");
      }
    },
    [continueSession, draftSessionId, session, sessionAttachments, starterJobId, startSession, uploadSessionAttachments],
  );

  const handleSend = useCallback(() => {
    if (!session || pendingSend) return;
    // An invitation the operator never answered is worse than no message at
    // all: "Here is the idea:" with nothing after it reads as truncation. The
    // selected job already carries the intent, so send empty instead.
    const message = isUnfilledOpener(session.kind, draft) ? "" : draft.trim();
    if (!message && sessionAttachments.attachments.length === 0 && pendingContext.length === 0 && !starterJobId) return;
    const send: PendingSend = {
      id: `pending-${session.id}-${session.messages.length}`,
      content: message,
      createdAt: new Date().toISOString(),
      context: pendingContext,
      delivery: "sending",
      // Snapshot of what the server had confirmed before this send. The
      // optimistic message retires the moment that count grows, which is a
      // fact about the transcript rather than a race with our own setState.
      baselineUserMessages: session.messages.filter((entry) => entry.role === "user").length,
    };
    setDraft("");
    setPendingContext([]);
    setPendingSend(send);
    void deliverMessage(send);
  }, [deliverMessage, draft, pendingContext, pendingSend, session, sessionAttachments, starterJobId]);

  const handleRetrySend = useCallback(() => {
    if (!pendingSend) return;
    const retry: PendingSend = { ...pendingSend, delivery: "sending" };
    setPendingSend(retry);
    void deliverMessage(retry);
  }, [deliverMessage, pendingSend]);

  const handleEditPendingSend = useCallback(() => {
    if (!pendingSend) return;
    setDraft(pendingSend.content);
    setPendingContext(pendingSend.context);
    setPendingSend(null);
  }, [pendingSend]);

  const handleRefresh = useCallback(async () => {
    if (!session) return;
    setLocalError(null);
    try {
      await refreshSession(session.id);
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to refresh session.");
    }
  }, [refreshSession, session]);

  const handleRefreshStartupBrief = useCallback(async () => {
    if (!session || session.status !== "draft") return;
    setLocalError(null);
    try {
      const brief = await loadStartupBrief(session.id, true);
      setPendingContext((current) => [
        startupBriefOption(session.kind, brief.title, brief.selectedAt),
        ...current.filter((item) => item.type !== "startup_brief"),
      ]);
      await startupBriefQuery.refetch();
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to refresh startup brief.");
    }
  }, [loadStartupBrief, session, startupBriefQuery]);

	const handleKindChange = useCallback(async (kind: CreatableAgentSessionKind) => {
		if (!session || session.status !== "draft" || kind === session.kind) return;
		setLocalError(null);
		const staged = pendingContext;
		try {
			const result = await changeSessionKind({ sessionId: session.id, kind, contextRefs: optionsToRefs(staged) });
			const dropped = new Set(result.droppedContextRefs.map((ref) => `${ref.type}\u0000${ref.ref}`));
			const droppedTitles = staged
				.filter((item) => dropped.has(`${item.type}\u0000${item.ref}`))
				.map((item) => item.title);
			setPendingContext((current) => current.filter((item) => !dropped.has(`${item.type}\u0000${item.ref}`)));
			setStarterJobId(result.session.starterJobId);
			const notices: string[] = [];
			if (droppedTitles.length > 0) notices.push(`Removed context this session kind does not allow: ${droppedTitles.join(", ")}.`);
			if (result.starterJobCleared) notices.push("Cleared the starter job because it is not offered for this session kind.");
			if (notices.length > 0) setLocalError(notices.join(" "));
		} catch (err) {
			setLocalError(err instanceof Error ? err.message : "Unable to change session kind.");
		}
	}, [changeSessionKind, pendingContext, session]);

  const handleCancel = useCallback(async () => {
    if (!session) return;
    setLocalError(null);
    try {
      await cancelSession(session.id);
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to cancel session.");
    }
  }, [cancelSession, session]);

  const handleDelete = useCallback(() => {
    if (!session) return;
    requestDeleteSession({
      entityName: session.title.trim() || session.id,
      description:
        "This removes the conversation, session details, proposal drafts, and session artifact links. Created backlog items, milestones, captures, files, and agent activity records stay in Swarm Manager.",
      confirmLabel: "Delete Session",
      testIds: {
        dialog: "session-delete-dialog",
        confirmButton: "session-delete-confirm",
        cancelButton: "session-delete-cancel",
        copyButton: "session-delete-copy",
      },
      onConfirm: async () => {
        setLocalError(null);
        try {
          await deleteSession(session.id);
          closeDetail();
        } catch (err) {
          setLocalError(err instanceof Error ? err.message : "Unable to delete session.");
          throw err; // keep the confirm dialog open so the user can retry
        }
      },
    });
  }, [closeDetail, deleteSession, requestDeleteSession, session]);

  const handleOpenArtifact = useCallback(
    (artifact: AgentSessionArtifact) => {
      const nodeId = nodeIdForSessionArtifact(artifact);
      if (!nodeId) return;
      const path = detailPathFromNodeId(nodeId);
      if (path) navigate(path);
    },
    [navigate],
  );

  const defaultInspectorSection = useMemo(
    () => (session ? defaultSessionInspectorSection(session.proposals, session.artifacts, session.status) : "details"),
    [session],
  );

  // The inspector's three panes have nothing to do with the composer draft, but
  // they were rebuilt from inline arrows on every render — so every keystroke
  // re-rendered the events timeline, the artifact list, and the details pane.
  // Stabilising these lets the memoized panes below actually skip.
  const proposalTarget = session?.proposalTarget;
  const handleOpenProposal = useCallback(() => {
    if (!proposalTarget) return;
    if (proposalTarget.type === "goal") {
      navigate(goalDetailPath(proposalTarget.ref, { tab: "proposals" }));
      return;
    }
    const slashIndex = proposalTarget.ref.indexOf("/");
    if (slashIndex > 0 && slashIndex < proposalTarget.ref.length - 1) {
      navigate(backlogDetailPath(proposalTarget.ref.slice(0, slashIndex), proposalTarget.ref.slice(slashIndex + 1), { tab: "proposals" }));
    }
  }, [navigate, proposalTarget]);

  const handleApplyProposal = useCallback(async (proposal: AgentSessionProposal) => {
    if (!session) return;
    setLocalError(null);
    try {
      await applyProposal(session.id, proposal.id);
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to apply proposal.");
    }
  }, [applyProposal, session]);

  const handleRefreshEvents = useCallback(() => { void refreshEvents(); }, [refreshEvents]);

  if (!sessionId) {
    return (
      <DetailPageLayout header={<DetailPageHeader entityType="Session" title="Not found" nodeId={null} lenses={[]} />}>
        <ErrorState message="No session selected." onRetry={closeDetail} />
      </DetailPageLayout>
    );
  }

  if (isLoading && !session) {
    return <PageLoadingState label="Loading session..." />;
  }

  if ((queryError && !session) || (!session && !isLoading)) {
    return (
      <DetailPageLayout header={<DetailPageHeader entityType="Session" title="Not found" nodeId={null} lenses={[]} />}>
        <ErrorState message="Session not found." onRetry={closeDetail} />
      </DetailPageLayout>
    );
  }

  if (!session) return <PageLoadingState label="Loading session..." />;

  const cancelDisabled = isMutating || TERMINAL_SESSION_STATUSES.has(session.status);
  const deleteDisabled = isMutating;
  const isWaitingForAgent = isSessionWaitingForAgent(session);

  // Retire the optimistic message as soon as the server's own copy lands, so
  // the transcript never shows the same turn twice even for one frame.
  const confirmedUserMessages = session.messages.filter((entry) => entry.role === "user").length;
  const pendingMessageView: ChatMessageView | undefined =
    pendingSend && !(pendingSend.delivery === "sending" && confirmedUserMessages > pendingSend.baselineUserMessages)
      ? {
          id: pendingSend.id,
          role: "user",
          content: pendingSend.content,
          createdAt: pendingSend.createdAt,
          delivery: pendingSend.delivery,
        }
      : undefined;

  const conversationProps = {
    messages: session.messages,
    draft,
    onDraftChange: setDraft,
    onSend: handleSend,
    isMutating,
    isWaitingForAgent,
    sessionKind: session.kind,
    sessionStatus: session.status,
    sessionId: session.id,
    attachments: session.attachments ?? [],
    pendingAttachments: sessionAttachments.attachments,
    onAttachFiles: (files: File[]) => files.forEach(sessionAttachments.addFile),
    onRemovePendingAttachment: sessionAttachments.removeFile,
    pendingContext,
    onPendingContextChange: setPendingContext,
    density,
    onDensityChange: handleDensityChange,
    profileKey: session.profileKey,
    runId: session.runId,
    pendingMessage: pendingMessageView,
    onRetryPendingMessage: handleRetrySend,
    onEditPendingMessage: handleEditPendingSend,
    starterJobId,
    onStarterJobChange: setStarterJobId,
  };

  const artifactContent = (variant: "panel" | "plain") => (
    <SessionArtifactList
      artifacts={session.artifacts}
      proposals={session.proposals}
      proposalTarget={session.proposalTarget}
      onOpenArtifact={handleOpenArtifact}
      onOpenProposal={handleOpenProposal}
      onApplyProposal={(proposal) => void handleApplyProposal(proposal)}
      applyingProposalId={isMutating ? session.proposals.find((proposal) => proposal.status === "ready")?.id : undefined}
      variant={variant}
    />
  );
  const detailContent = (variant: "panel" | "plain") => <SessionMetadata session={session} variant={variant} />;
  const eventsContent = (variant: "panel" | "plain") => (
    <SessionEventTimeline
      events={events}
      isLoading={eventsLoading}
      error={eventsError}
      onRefresh={handleRefreshEvents}
      variant={variant}
    />
  );

  const inspectorSections = [
    { value: "events" as const, label: "Events", count: events.length, content: eventsContent("plain") },
    { value: "artifacts" as const, label: "Artifacts", count: session.artifacts.length + session.proposals.length, content: artifactContent("plain") },
    { value: "details" as const, label: "Details", content: detailContent("plain") },
  ];

  const menuActions: ActionMenuItem[] = [
    attachToSession.actionItem,
    ...(session.status === "draft" ? [{
      label: "Refresh brief",
      description: "Regenerate the context brief before the session begins.",
      icon: <RefreshCw />,
      onSelect: () => void handleRefreshStartupBrief(),
      disabled: isMutating || startupBriefQuery.isFetching,
      loading: startupBriefQuery.isFetching,
      testId: "session-startup-brief-refresh",
    }] : []),
    {
      label: "Refresh",
      description: "Fetch the latest session state and agent activity.",
      icon: <RefreshCw />,
      onSelect: () => void handleRefresh(),
      disabled: isMutating || isRefreshing,
      loading: isRefreshing,
      testId: "session-refresh",
    },
    {
      label: "Delete session",
      description: "Permanently remove this session and its stored context.",
      icon: <Trash2 />,
      onSelect: handleDelete,
      disabled: deleteDisabled,
      destructive: true,
      testId: "session-delete-action",
    },
  ];

	const primaryAction = session.status === "draft" ? (
		<label className="flex items-center gap-2 text-xs text-slate-400">
			<span>Session kind</span>
			<select
				aria-label="Session kind"
				data-testid="session-kind-selector"
				value={session.kind}
				disabled={isMutating}
				onChange={(event) => void handleKindChange(event.target.value as CreatableAgentSessionKind)}
				className="rounded-md border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-100"
			>
				<option value="meta_orchestration">Plan Work</option>
				<option value="swarm_operations">Swarm Operations</option>
				<option value="workflow_authoring">Improve the System</option>
			</select>
		</label>
	) : (
    <Button variant="ghost" size="sm" onClick={() => void handleCancel()} disabled={cancelDisabled} data-testid="session-cancel">
      <Square className="mr-1.5 h-3.5 w-3.5" />
      Cancel
    </Button>
  );

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="Session"
          title={session.title || "Agent session"}
          subtitle={`${SESSION_KIND_LABELS[session.kind]} · ${formatRelativeTime(session.updatedAt)}`}
          status={formatDisplayText(session.status)}
          nodeId={null}
          lenses={[]}
          primaryAction={primaryAction}
          menuActions={menuActions}
          menuTriggerTestId="session-header-actions"
          menuTestId="session-actions-menu"
          tabBar={
            isMobile ? (
              <SessionSectionTabs
                sections={[
                  { value: "conversation", label: "Conversation", content: null },
                  { value: "events", label: "Events", count: events.length, content: null },
                  { value: "artifacts", label: "Artifacts", count: session.artifacts.length + session.proposals.length, content: null },
                  { value: "details", label: "Details", content: null },
                ]}
                activeValue={mobileSection}
                onValueChange={setMobileSection}
                listLabel="Session sections"
                tabBarClassName="border-y border-slate-200/20"
                contentClassName="hidden"
              />
            ) : undefined
          }
        />
      }
      className="min-h-screen md:h-full md:min-h-0 md:overflow-hidden"
      bodyClassName="md:flex md:min-h-0 md:flex-col md:overflow-hidden md:px-0 md:py-0"
    >
      <div className="mx-auto flex h-full w-full max-w-7xl flex-col gap-3 md:max-w-none md:gap-0">
        {(localError || error?.message) && (
          <div className="flex items-start gap-2 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200" role="alert">
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <span className="min-w-0 break-words">{localError || error?.message}</span>
          </div>
        )}

        {isMobile ? (
          // Bounded height, not min-height: the conversation owns its own
          // scrolling, and a pane that can only grow pushes the page instead
          // of scrolling inside itself. 100dvh tracks the mobile URL bar.
          <div className="flex h-[calc(100dvh-11rem)] min-h-0 flex-col overflow-hidden">
            {mobileSection === "conversation" && <SessionConversation {...conversationProps} variant="mobile" />}
            {mobileSection !== "conversation" && (
              <div className="min-h-0 flex-1 overflow-y-auto">
                {mobileSection === "events" && eventsContent("plain")}
                {mobileSection === "artifacts" && artifactContent("plain")}
                {mobileSection === "details" && detailContent("plain")}
              </div>
            )}
          </div>
        ) : (
          <div ref={desktopLayoutRef} className="flex min-h-0 flex-1" data-testid="session-desktop-layout">
            <SessionConversation {...conversationProps} desktopPresentation="pane" />
            <SessionInspector
              containerRef={desktopLayoutRef}
              sections={inspectorSections}
              defaultSection={defaultInspectorSection}
              isCollapsed={inspectorCollapsed}
              onCollapsedChange={setInspectorCollapsed}
              presentation="pane"
            />
          </div>
        )}
      </div>
      {attachToSession.sheet}
      <ConfirmDialog {...deleteDialogProps} />
    </DetailPageLayout>
  );
}
