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
import { sessionOption, startupBriefOption, type SessionContextOption } from "../components/session/context/session-context-refs";
import { clearStagedContextForSession, mergeContextOptions, peekStagedContextForSession } from "../components/session/context/pending-session-context";
import { readSessionDraft, writeSessionDraft } from "../components/session/session-draft-storage";
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
import { useAgentSessionStore } from "../stores";
import { useAgentSessionEvents } from "../hooks/useAgentSessionEvents";
import { useAgentSessionPolling } from "../hooks/useAgentSessionPolling";
import { useAppBack } from "../app/routes/useAppBack";
import { backlogDetailPath, detailPathFromNodeId, goalDetailPath } from "../app/routes/route-paths";
import { useIsMobile } from "../hooks/useMediaQuery";
import type { AgentSession, AgentSessionArtifact } from "../types";

export function SessionDetailsPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const closeDetail = useAppBack();
  const isMobile = useIsMobile();
  const desktopLayoutRef = useRef<HTMLDivElement>(null);

  const storeSession = useAgentSessionStore((s) =>
    s.sessions.find((session) => session.id === sessionId),
  );
  const loadSession = useAgentSessionStore((s) => s.loadSession);
  const loadStartupBrief = useAgentSessionStore((s) => s.loadStartupBrief);
  const startSession = useAgentSessionStore((s) => s.startSession);
  const continueSession = useAgentSessionStore((s) => s.continueSession);
  const uploadSessionAttachments = useAgentSessionStore((s) => s.uploadSessionAttachments ?? (() => Promise.resolve([])));
  const refreshSession = useAgentSessionStore((s) => s.refreshSession);
  const cancelSession = useAgentSessionStore((s) => s.cancelSession);
  const deleteSession = useAgentSessionStore((s) => s.deleteSession);
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
  const [pendingContext, setPendingContext] = useState<SessionContextOption[]>([]);
  const [localError, setLocalError] = useState<string | null>(null);
  const [inspectorCollapsed, setInspectorCollapsed] = useState(false);
  const { requestDelete: requestDeleteSession, dialogProps: deleteDialogProps } =
    useDeleteConfirm("session");
  const [mobileSection, setMobileSection] = useState<SessionSectionValue>("conversation");
  const draftSessionId = sessionId ?? "unknown";
  const sessionAttachments = useComposerImageAttachments(sessionId ? `swarm-session-attachments:${sessionId}` : "swarm-session-attachments:unknown");

  useEffect(() => {
    setDraft(readSessionDraft(draftSessionId));
    setPendingContext(session?.status === "draft" ? [startupBriefOption(session.kind)] : []);
  }, [session?.id, session?.kind, session?.status, draftSessionId]);

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
        startupBriefOption(session.kind, brief.title, brief.selectedAt),
        ...withoutStartupBrief,
      ];
    });
  }, [session, startupBriefQuery.data]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      writeSessionDraft(draftSessionId, draft);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [draft, draftSessionId]);

  const handleSend = useCallback(async () => {
    if (!session || (!draft.trim() && sessionAttachments.attachments.length === 0 && pendingContext.length === 0)) return;
    const message = draft.trim();
    const context = pendingContext;
    setDraft("");
    setPendingContext([]);
    setLocalError(null);
    try {
      const uploaded = await uploadSessionAttachments(session.id, sessionAttachments.getFiles());
      const attachmentIds = uploaded.map((attachment) => attachment.id);
      const contextRefs = optionsToRefs(context);
      const hasAutoStartupContext = contextRefs.some((ref) => ref.type === "startup_brief" || ref.type === "operations_briefing");
      const sendArgs = {
        sessionId: session.id,
        message,
        ...(attachmentIds.length > 0 ? { attachmentIds } : {}),
        ...(contextRefs.length > 0 ? { contextRefs } : {}),
        ...(session.status === "draft" && hasAutoStartupContext ? { autoContextPolicy: "none" as const } : {}),
      };
      if (session.status === "draft") {
        await startSession(sendArgs);
      } else {
        await continueSession(sendArgs);
      }
      sessionAttachments.clearAll();
      writeSessionDraft(draftSessionId, "");
    } catch (err) {
      setDraft(message);
      setPendingContext(context);
      setLocalError(err instanceof Error ? err.message : "Unable to send session message.");
    }
  }, [continueSession, draft, draftSessionId, pendingContext, session, sessionAttachments, startSession, uploadSessionAttachments]);

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

  const artifactContent = (variant: "panel" | "plain") => (
    <SessionArtifactList
      artifacts={session.artifacts}
      proposals={session.proposals}
      proposalTarget={session.proposalTarget}
      onOpenArtifact={handleOpenArtifact}
      onOpenProposal={() => {
        const target = session.proposalTarget;
        if (!target) return;
        if (target.type === "goal") {
          navigate(goalDetailPath(target.ref, { tab: "proposals" }));
          return;
        }
        const slashIndex = target.ref.indexOf("/");
        if (slashIndex > 0 && slashIndex < target.ref.length - 1) {
          navigate(backlogDetailPath(target.ref.slice(0, slashIndex), target.ref.slice(slashIndex + 1), { tab: "proposals" }));
        }
      }}
      variant={variant}
    />
  );
  const detailContent = (variant: "panel" | "plain") => <SessionMetadata session={session} variant={variant} />;
  const eventsContent = (variant: "panel" | "plain") => (
    <SessionEventTimeline
      events={events}
      isLoading={eventsLoading}
      error={eventsError}
      onRefresh={() => void refreshEvents()}
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

  const primaryAction = (
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
          <div className="min-h-[calc(100vh-11rem)]">
            {mobileSection === "conversation" && (
              <SessionConversation
                messages={session.messages}
                draft={draft}
                onDraftChange={setDraft}
                onSend={() => void handleSend()}
                isMutating={isMutating}
                isWaitingForAgent={isWaitingForAgent}
                sessionKind={session.kind}
                sessionStatus={session.status}
                sessionId={session.id}
                attachments={session.attachments ?? []}
                pendingAttachments={sessionAttachments.attachments}
                onAttachFiles={(files) => files.forEach(sessionAttachments.addFile)}
                onRemovePendingAttachment={sessionAttachments.removeFile}
                pendingContext={pendingContext}
                onPendingContextChange={setPendingContext}
                variant="mobile"
              />
            )}
            {mobileSection === "events" && eventsContent("plain")}
            {mobileSection === "artifacts" && artifactContent("plain")}
            {mobileSection === "details" && detailContent("plain")}
          </div>
        ) : (
          <div ref={desktopLayoutRef} className="flex min-h-0 flex-1" data-testid="session-desktop-layout">
            <SessionConversation
              messages={session.messages}
              draft={draft}
              onDraftChange={setDraft}
              onSend={() => void handleSend()}
              isMutating={isMutating}
              isWaitingForAgent={isWaitingForAgent}
              sessionKind={session.kind}
              sessionStatus={session.status}
              sessionId={session.id}
              attachments={session.attachments ?? []}
              pendingAttachments={sessionAttachments.attachments}
              onAttachFiles={(files) => files.forEach(sessionAttachments.addFile)}
              onRemovePendingAttachment={sessionAttachments.removeFile}
              pendingContext={pendingContext}
              onPendingContextChange={setPendingContext}
              desktopPresentation="pane"
            />
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
