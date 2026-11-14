/** Agent-session collection surface. Domain content remains in SessionSummaryCard. */
import { memo, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { CollectionList } from "@vrooli/react-component-library/CollectionList/1";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { isActiveAgentSession, useAgentSessionStore } from "../../../../stores";
import { SessionSummaryCard } from "../../../../components/session/session-summary-card";
import { CollectionRow } from "../../../../components/ui/collection-row";
import { sessionDetailPath } from "../../../../app/routes/route-paths";
import { matchesSearch } from "./useSidebarSearch";
import { applySessionFilters, applySessionSort } from "./session-list-utils";
import type { AgentSession } from "../../../../types";
import type { SessionFilters, SortConfig } from "./types";
import { SidebarEmptyState } from "./SidebarEmptyState";

interface SessionsTabProps {
  searchQuery: string;
  filters: SessionFilters;
  sort: SortConfig;
  onOpenSession?: (sessionId: string) => void;
  onClearSearch?: () => void;
}

function SessionsTabImpl({ searchQuery, filters, sort, onOpenSession, onClearSearch }: SessionsTabProps) {
  const sessions = useAgentSessionStore((s) => s.sessions);
  const navigate = useNavigate();
  const sorted = useMemo(
    () =>
      applySessionSort(
        applySessionFilters(sessions, filters).filter(
          (session) =>
            !searchQuery ||
            matchesSearch(searchQuery, session.title, session.kind, session.status, session.skillId, session.runId),
        ),
        sort,
      ),
    [filters, searchQuery, sessions, sort],
  );

  const open = (session: AgentSession) => {
    navigate(sessionDetailPath(session.id));
    onOpenSession?.(session.id);
  };

  if (sorted.length === 0) {
    return (
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.sessions}
        title={
          searchQuery || filters.statuses.length || filters.kinds.length
            ? "No sessions match your filters."
            : "No agent sessions yet."
        }
        hint="Plan-work, operations, and authoring conversations show up here once started."
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <CollectionList
      items={sorted}
      getKey={(session) => session.id}
      label="Sessions"
      virtualize
      onOpen={open}
      selection={{ mode: "none", enterOn: ["shortcut"] }}
      actions={[
        {
          id: "open",
          label: "Open",
          onSelect: ([session]) => {
            if (session) open(session);
          },
        },
        {
          id: "cancel",
          label: "Cancel",
          bulk: true,
          disabled: (session) => (isActiveAgentSession(session) ? false : "Only active sessions can be canceled"),
          onSelect: async (rows) => {
            for (const session of rows) await useAgentSessionStore.getState().cancelSession(session.id);
          },
        },
        {
          id: "delete",
          label: "Delete",
          tone: "destructive",
          bulk: true,
          disabled: (session) => (isActiveAgentSession(session) ? "Active sessions cannot be deleted" : false),
          onSelect: async (rows) => {
            for (const session of rows) await useAgentSessionStore.getState().deleteSession(session.id);
          },
        },
      ]}
      renderItem={(session) => (
        <CollectionRow data-testid="sidebar-session-item">
          <SessionSummaryCard session={session} />
        </CollectionRow>
      )}
    />
  );
}

export const SessionsTab = memo(SessionsTabImpl);
