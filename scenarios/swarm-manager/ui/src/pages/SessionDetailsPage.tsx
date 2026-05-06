/**
 * SessionDetailsPage — routed agent-session detail page.
 *
 * Owns route/data orchestration and assembles session-specific components.
 */

import { useCallback, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { AlertCircle, Bot, MoreVertical, PanelRightOpen, RefreshCw, Square } from "lucide-react";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { Button } from "../components/ui/button";
import { BottomSheet } from "../components/ui/bottom-sheet";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { SessionArtifactList } from "../components/session/SessionArtifactList";
import { SessionConversation } from "../components/session/SessionConversation";
import { SessionInspector } from "../components/session/SessionInspector";
import { SessionMetadata } from "../components/session/SessionMetadata";
import { SessionProposalList } from "../components/session/SessionProposalList";
import { SessionSectionTabs, type SessionSectionValue } from "../components/session/SessionSectionTabs";
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
  const continueSession = useAgentSessionStore((s) => s.continueSession);
  const refreshSession = useAgentSessionStore((s) => s.refreshSession);
  const cancelSession = useAgentSessionStore((s) => s.cancelSession);
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

  const [draft, setDraft] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);
  const [inspectorCollapsed, setInspectorCollapsed] = useState(false);
  const [mobileActionsOpen, setMobileActionsOpen] = useState(false);
  const [mobileSection, setMobileSection] = useState<SessionSectionValue>("conversation");

  const handleSend = useCallback(async () => {
    if (!session || !draft.trim()) return;
    const message = draft.trim();
    setDraft("");
    setLocalError(null);
    try {
      await continueSession({ sessionId: session.id, message });
    } catch (err) {
      setDraft(message);
      setLocalError(err instanceof Error ? err.message : "Unable to continue session.");
    }
  }, [continueSession, draft, session]);

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
    () => (session ? defaultSessionInspectorSection(session.proposals, session.artifacts) : "details"),
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
  const isWaitingForAgent = isSessionWaitingForAgent(session);

  const proposalContent = (variant: "panel" | "plain") => (
    <SessionProposalList proposals={session.proposals} isMutating={isMutating} onApply={handleApply} variant={variant} />
  );
  const artifactContent = (variant: "panel" | "plain") => (
    <SessionArtifactList artifacts={session.artifacts} onOpenArtifact={handleOpenArtifact} variant={variant} />
  );
  const detailContent = (variant: "panel" | "plain") => <SessionMetadata session={session} variant={variant} />;

  const inspectorSections = [
    { value: "proposals" as const, label: "Proposals", count: session.proposals.length, content: proposalContent("plain") },
    { value: "artifacts" as const, label: "Artifacts", count: session.artifacts.length, content: artifactContent("plain") },
    { value: "details" as const, label: "Details", content: detailContent("plain") },
  ];

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
        </>
      )}
    </>
  );

  const mobileActions = (
    <div className="flex flex-col gap-2 p-2">
      <Button
        variant="ghost"
        onClick={() => {
          setMobileActionsOpen(false);
          void handleRefresh();
        }}
        disabled={isMutating || isRefreshing}
        data-testid="session-refresh"
      >
        <RefreshCw className={cn("mr-2 h-4 w-4", isRefreshing && "animate-spin")} />
        Refresh
      </Button>
      <Button
        variant="ghost"
        onClick={() => {
          setMobileActionsOpen(false);
          void handleCancel();
        }}
        disabled={cancelDisabled}
        data-testid="session-cancel"
      >
        <Square className="mr-2 h-4 w-4" />
        Cancel
      </Button>
    </div>
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
                variant="mobile"
              />
            )}
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
          data-testid="session-mobile-actions-sheet"
        >
          {mobileActions}
        </BottomSheet>
      )}
    </DetailPageLayout>
  );
}
