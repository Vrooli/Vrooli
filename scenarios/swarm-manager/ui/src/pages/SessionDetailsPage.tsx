/**
 * SessionDetailsPage — routed agent-session detail page.
 *
 * Owns route/data orchestration and assembles session-specific components.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { AlertCircle, Bot, MoreVertical, PanelRightOpen, RefreshCw, Square, Trash2 } from "lucide-react";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { Button } from "../components/ui/button";
import { BottomSheet } from "../components/ui/bottom-sheet";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { SessionArtifactList } from "../components/session/SessionArtifactList";
import { SessionConversation } from "../components/session/SessionConversation";
import { SessionDeleteDialog } from "../components/session/SessionDeleteDialog";
import { SessionEventTimeline } from "../components/session/SessionEventTimeline";
import { SessionInspector } from "../components/session/SessionInspector";
import { SessionMetadata } from "../components/session/SessionMetadata";
import { SessionProposalList } from "../components/session/SessionProposalList";
import { SessionSectionTabs, type SessionSectionValue } from "../components/session/SessionSectionTabs";
import { useComposerImageAttachments } from "../components/composer/useComposerImageAttachments";
import { optionsToRefs } from "../components/session/context/session-context-options";
import { operationsBriefingOption, type SessionContextOption } from "../components/session/context/session-context-refs";
import { ActionMenu, ActionMenuSheetContent, type ActionMenuItem } from "../components/ui/action-menu";
import { nodeIdForSessionArtifact } from "../components/session/session-artifact-routing";
import {
  defaultSessionInspectorSection,
  isSessionWaitingForAgent,
  SESSION_KIND_ICONS,
  SESSION_KIND_LABELS,
  TERMINAL_SESSION_STATUSES,
} from "../components/session/session-view-model";
import { cn } from "../lib/utils";
import { formatDisplayText, formatRelativeTime } from "../lib/format-utils";
import { useAgentSessionStore } from "../stores";
import { useAgentSessionEvents } from "../hooks/useAgentSessionEvents";
import { useAgentSessionPolling } from "../hooks/useAgentSessionPolling";
import { useAppBack } from "../app/routes/useAppBack";
import { detailPathFromNodeId } from "../app/routes/route-paths";
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
  const { events, isLoading: eventsLoading, error: eventsError, refreshEvents } = useAgentSessionEvents(session);

  const [draft, setDraft] = useState("");
  const [pendingContext, setPendingContext] = useState<SessionContextOption[]>([]);
  const [localError, setLocalError] = useState<string | null>(null);
  const [inspectorCollapsed, setInspectorCollapsed] = useState(false);
  const [mobileActionsOpen, setMobileActionsOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [mobileSection, setMobileSection] = useState<SessionSectionValue>("conversation");
  const sessionDraftKey = sessionId ? `swarm-session-draft:${sessionId}` : "swarm-session-draft:unknown";
  const sessionAttachments = useComposerImageAttachments(sessionId ? `swarm-session-attachments:${sessionId}` : "swarm-session-attachments:unknown");

  useEffect(() => {
    try {
      setDraft(localStorage.getItem(sessionDraftKey) ?? "");
    } catch {
      setDraft("");
    }
    setPendingContext(session?.kind === "swarm_operations" && session.status === "draft" ? [operationsBriefingOption()] : []);
  }, [session?.kind, session?.status, sessionDraftKey]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      try {
        if (draft) {
          localStorage.setItem(sessionDraftKey, draft);
        } else {
          localStorage.removeItem(sessionDraftKey);
        }
      } catch {
        // ignore storage failures
      }
    }, 250);
    return () => window.clearTimeout(timer);
  }, [draft, sessionDraftKey]);

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
      const hasOperationsBriefing = contextRefs.some((ref) => ref.type === "operations_briefing");
      const sendArgs = {
        sessionId: session.id,
        message,
        ...(attachmentIds.length > 0 ? { attachmentIds } : {}),
        ...(contextRefs.length > 0 ? { contextRefs } : {}),
        ...(session.status === "draft" && session.kind === "swarm_operations" && !hasOperationsBriefing ? { autoContextPolicy: "none" as const } : {}),
      };
      if (session.status === "draft") {
        await startSession(sendArgs);
      } else {
        await continueSession(sendArgs);
      }
      sessionAttachments.clearAll();
      try { localStorage.removeItem(sessionDraftKey); } catch { /* ignore */ }
    } catch (err) {
      setDraft(message);
      setPendingContext(context);
      setLocalError(err instanceof Error ? err.message : "Unable to send session message.");
    }
  }, [continueSession, draft, pendingContext, session, sessionAttachments, sessionDraftKey, startSession, uploadSessionAttachments]);

  const handleRefresh = useCallback(async () => {
    if (!session) return;
    setLocalError(null);
    try {
      await refreshSession(session.id);
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to refresh session.");
    }
  }, [refreshSession, session]);

  const handleCancel = useCallback(async () => {
    if (!session) return;
    setLocalError(null);
    try {
      await cancelSession(session.id);
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to cancel session.");
    }
  }, [cancelSession, session]);

  const handleDelete = useCallback(async () => {
    if (!session) return;
    setLocalError(null);
    try {
      await deleteSession(session.id);
      setDeleteDialogOpen(false);
      closeDetail();
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to delete session.");
    }
  }, [closeDetail, deleteSession, session]);

  const handleApply = useCallback(
    async (proposalId: string) => {
      if (!session) return;
      setLocalError(null);
      try {
        await applyProposal(session.id, proposalId);
      } catch (err) {
        setLocalError(err instanceof Error ? err.message : "Unable to apply proposal.");
      }
    },
    [applyProposal, session],
  );

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

  const Icon = SESSION_KIND_ICONS[session.kind] ?? Bot;
  const cancelDisabled = isMutating || TERMINAL_SESSION_STATUSES.has(session.status);
  const deleteDisabled = isMutating;
  const isWaitingForAgent = isSessionWaitingForAgent(session);

  const proposalContent = (variant: "panel" | "plain") => (
    <SessionProposalList proposals={session.proposals} isMutating={isMutating} onApply={handleApply} variant={variant} />
  );
  const artifactContent = (variant: "panel" | "plain") => (
    <SessionArtifactList artifacts={session.artifacts} onOpenArtifact={handleOpenArtifact} variant={variant} />
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
    { value: "proposals" as const, label: "Proposals", count: session.proposals.length, content: proposalContent("plain") },
    { value: "artifacts" as const, label: "Artifacts", count: session.artifacts.length, content: artifactContent("plain") },
    { value: "details" as const, label: "Details", content: detailContent("plain") },
  ];

  const mobileActionItems: ActionMenuItem[] = [
    {
      label: "Refresh",
      icon: <RefreshCw />,
      onSelect: () => void handleRefresh(),
      disabled: isMutating || isRefreshing,
      loading: isRefreshing,
      testId: "session-refresh",
    },
    {
      label: "Cancel",
      icon: <Square />,
      onSelect: () => void handleCancel(),
      disabled: cancelDisabled,
      testId: "session-cancel",
    },
    {
      label: "Delete session",
      icon: <Trash2 />,
      onSelect: () => setDeleteDialogOpen(true),
      disabled: deleteDisabled,
      destructive: true,
      testId: "session-delete-action",
    },
  ];
  const desktopDeleteItems = mobileActionItems.filter((item) => item.testId === "session-delete-action");

  const headerActions = (
    <>
      {isMobile ? (
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setMobileActionsOpen(true)}
          aria-label="Session actions"
          data-testid="session-mobile-header-actions"
        >
          <MoreVertical className="h-4 w-4" />
        </Button>
      ) : (
        <>
          {inspectorCollapsed && (
            <Button variant="ghost" size="sm" onClick={() => setInspectorCollapsed(false)} data-testid="session-inspector-header-expand">
              <PanelRightOpen className="mr-1.5 h-3.5 w-3.5" />
              Inspector
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={() => void handleRefresh()} disabled={isMutating || isRefreshing} data-testid="session-refresh">
            <RefreshCw className={cn("mr-1.5 h-3.5 w-3.5", isRefreshing && "animate-spin")} />
            Refresh
          </Button>
          <Button variant="ghost" size="sm" onClick={() => void handleCancel()} disabled={cancelDisabled} data-testid="session-cancel">
            <Square className="mr-1.5 h-3.5 w-3.5" />
            Cancel
          </Button>
          <ActionMenu
            items={desktopDeleteItems}
            label="Session actions"
            triggerTestId="session-desktop-header-actions"
            menuTestId="session-desktop-actions-menu"
          />
        </>
      )}
    </>
  );

  const mobileActions = (
    <ActionMenuSheetContent items={mobileActionItems} onItemSelected={() => setMobileActionsOpen(false)} />
  );

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="Session"
          entityIcon={Icon}
          title={session.title || "Agent session"}
          subtitle={`${SESSION_KIND_LABELS[session.kind]} · ${formatRelativeTime(session.updatedAt)}`}
          status={formatDisplayText(session.status)}
          nodeId={null}
          lenses={[]}
          actions={headerActions}
          tabBar={
            isMobile ? (
              <SessionSectionTabs
                sections={[
                  { value: "conversation", label: "Conversation", content: null },
                  { value: "events", label: "Events", count: events.length, content: null },
                  { value: "proposals", label: "Proposals", count: session.proposals.length, content: null },
                  { value: "artifacts", label: "Artifacts", count: session.artifacts.length, content: null },
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
      className="min-h-screen"
    >
      <div className="mx-auto flex h-full w-full max-w-7xl flex-col gap-3">
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
            {mobileSection === "proposals" && proposalContent("plain")}
            {mobileSection === "artifacts" && artifactContent("plain")}
            {mobileSection === "details" && detailContent("plain")}
          </div>
        ) : (
          <div ref={desktopLayoutRef} className="flex min-h-[min(72vh,calc(100vh-10rem))] gap-3" data-testid="session-desktop-layout">
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
            />
            <SessionInspector
              containerRef={desktopLayoutRef}
              sections={inspectorSections}
              defaultSection={defaultInspectorSection}
              isCollapsed={inspectorCollapsed}
              onCollapsedChange={setInspectorCollapsed}
            />
          </div>
        )}
      </div>
      {isMobile && (
        <BottomSheet
          isOpen={mobileActionsOpen}
          onClose={() => setMobileActionsOpen(false)}
          title="Session actions"
          contentClassName="px-0 py-2 pb-[calc(0.5rem+env(safe-area-inset-bottom))]"
          data-testid="session-mobile-actions-sheet"
        >
          {mobileActions}
        </BottomSheet>
      )}
      <SessionDeleteDialog
        session={session}
        isOpen={deleteDialogOpen}
        isDeleting={isMutating}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={() => void handleDelete()}
      />
    </DetailPageLayout>
  );
}
