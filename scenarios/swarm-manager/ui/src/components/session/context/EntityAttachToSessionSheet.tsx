import { useEffect, useMemo, useState, type ReactNode } from "react";
import { ListPlus, MessageCirclePlus, Plus, Search } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { sessionDetailPath } from "../../../app/routes/route-paths";
import { selectors } from "../../../consts/selectors";
import { useAgentSessionStore } from "../../../stores";
import type { AgentSession, AgentSessionKind } from "../../../types";
import { cn } from "../../../lib/utils";
import { BottomSheet } from "../../ui/bottom-sheet";
import { Button } from "../../ui/button";
import { Card } from "../../ui/card";
import { Input } from "../../ui/input";
import { ContextChipTray } from "../../composer/ContextChipTray";
import {
  SESSION_KIND_DESCRIPTIONS,
  SESSION_KIND_ICONS,
  SESSION_KIND_LABELS,
} from "../session-view-model";
import { SessionSummaryCard } from "../session-summary-card";
import { CONTEXT_TYPE_LABELS, compatibleSessionKindsForContextType, sessionKindAllowsContextType } from "./session-context-config";
import { stageContextForSession } from "./pending-session-context";
import { type SessionContextOption } from "./session-context-refs";

interface EntityAttachToSessionSheetProps {
  isOpen: boolean;
  onClose: () => void;
  option: SessionContextOption;
  currentSessionId?: string;
}

export function EntityAttachToSessionSheet({
  isOpen,
  onClose,
  option,
  currentSessionId,
}: EntityAttachToSessionSheetProps) {
  if (!isOpen) return null;
  return (
    <EntityAttachToSessionSheetContent
      isOpen={isOpen}
      onClose={onClose}
      option={option}
      currentSessionId={currentSessionId}
    />
  );
}

function EntityAttachToSessionSheetContent({
  isOpen,
  onClose,
  option,
  currentSessionId,
}: EntityAttachToSessionSheetProps) {
  const navigate = useNavigate();
  const fetchSessions = useAgentSessionStore((s) => s.fetchSessions);
  const sessions = useAgentSessionStore((s) => s.sessions);
  const createSession = useAgentSessionStore((s) => s.createSession);
  const isMutating = useAgentSessionStore((s) => s.isMutating);
  const [query, setQuery] = useState("");
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [selectedKind, setSelectedKind] = useState<AgentSessionKind>(() => compatibleSessionKindsForContextType(option.type)[0] ?? "meta_orchestration");
  const [localError, setLocalError] = useState<string | null>(null);

  const compatibleKinds = useMemo(() => compatibleSessionKindsForContextType(option.type), [option.type]);

  useEffect(() => {
    setSelectedSessionId(null);
    setSelectedKind(compatibleKinds[0] ?? "meta_orchestration");
    setQuery("");
    setLocalError(null);
    void fetchSessions({ limit: 100 }, { force: true });
  }, [compatibleKinds, fetchSessions, isOpen]);

  const visibleSessions = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return sessions
      .filter((session) => session.id !== currentSessionId)
      .filter((session) => {
        if (!needle) return true;
        return [
          session.id,
          session.title,
          session.kind,
          session.status,
          session.skillId,
          session.runId,
          session.taskId,
        ].filter(Boolean).join(" ").toLowerCase().includes(needle);
      })
      .slice(0, 100);
  }, [currentSessionId, query, sessions]);

  const selectedSession = visibleSessions.find((session) => session.id === selectedSessionId)
    ?? sessions.find((session) => session.id === selectedSessionId);
  const canAttachSelected = Boolean(selectedSession && sessionKindAllowsContextType(selectedSession.kind, option.type));

  const attachToExisting = () => {
    if (!selectedSession || !canAttachSelected) return;
    stageContextForSession(selectedSession.id, option);
    onClose();
    navigate(sessionDetailPath(selectedSession.id));
  };

  const quickStart = async () => {
    setLocalError(null);
    try {
      const session = await createSession({
        kind: selectedKind,
        title: titleForQuickStart(option),
      });
      stageContextForSession(session.id, option);
      onClose();
      navigate(sessionDetailPath(session.id));
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to create draft session.");
    }
  };

  return (
    <BottomSheet
      isOpen={isOpen}
      onClose={onClose}
      title="Attach to session"
      description="Stage this entity in a session composer."
      className="!max-w-3xl border-slate-700/80 bg-slate-900"
      contentClassName="px-0 py-0"
      footer={
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="min-w-0 truncate text-xs text-slate-400">
            {selectedSession
              ? `Selected ${selectedSession.title || selectedSession.id}`
              : "Pick a session below, or draft a new one above."}
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" size="sm" onClick={onClose}>Cancel</Button>
            <Button size="sm" onClick={attachToExisting} disabled={!canAttachSelected} data-testid={selectors.agentSessions.entityAttachConfirm}>
              Attach
            </Button>
          </div>
        </div>
      }
      data-testid={selectors.agentSessions.entityAttachSheet}
    >
      <div className="flex min-h-0 flex-col">
        <div className="space-y-4 border-b border-white/10 px-4 py-4">
          {/* What is being attached */}
          <div>
            <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wider text-slate-500">
              Attaching
            </p>
            <ContextChipTray items={[option]} className="max-h-16" />
          </div>

          {localError && (
            <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200" role="alert">
              {localError}
            </div>
          )}

          {/* Option A — start a fresh session */}
          <Card padding="none" className="p-3.5" data-testid="entity-attach-new-section">
            <SectionHeading
              icon={<MessageCirclePlus className="h-4 w-4" />}
              tone="accent"
              title="New draft session"
              subtitle={`${CONTEXT_TYPE_LABELS[option.type]} context will be staged before the first message.`}
            />
            <div className="mt-3">
              <SessionKindChoices
                kinds={compatibleKinds}
                selectedKind={selectedKind}
                onSelect={setSelectedKind}
              />
            </div>
            <div className="mt-3 flex justify-end">
              <Button size="sm" onClick={() => void quickStart()} disabled={isMutating || compatibleKinds.length === 0} data-testid={selectors.agentSessions.entityAttachQuickStart}>
                <Plus className="mr-1.5 h-3.5 w-3.5" />
                Draft session
              </Button>
            </div>
          </Card>

          {/* Either/or boundary */}
          <div className="flex items-center gap-3 text-[11px] font-medium uppercase tracking-wider text-slate-500" aria-hidden>
            <span className="h-px flex-1 bg-white/10" />
            or
            <span className="h-px flex-1 bg-white/10" />
          </div>

          {/* Option B — attach to something that already exists */}
          <div data-testid="entity-attach-existing-section">
            <SectionHeading
              icon={<ListPlus className="h-4 w-4" />}
              tone="muted"
              title="Add to an existing session"
              subtitle="Stage this context in a session you already have open."
            />
            <div className="relative mt-3">
              <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search sessions..."
                className="h-9 border-slate-700 bg-slate-950/70 pl-9 text-slate-100 placeholder:text-slate-500"
                data-testid={selectors.agentSessions.entityAttachSearch}
              />
            </div>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3" data-testid={selectors.agentSessions.entityAttachSessionList}>
          {visibleSessions.length > 0 ? (
            <div className="space-y-1.5">
              {visibleSessions.map((session) => (
                <AttachSessionRow
                  key={session.id}
                  session={session}
                  selected={session.id === selectedSessionId}
                  compatible={sessionKindAllowsContextType(session.kind, option.type)}
                  onSelect={() => setSelectedSessionId(session.id)}
                />
              ))}
            </div>
          ) : (
            <div className="rounded-md border border-dashed border-white/10 bg-slate-950/30 px-3 py-10 text-center text-sm text-slate-500">
              No matching sessions.
            </div>
          )}
        </div>
      </div>
    </BottomSheet>
  );
}

function SectionHeading({
  icon,
  title,
  subtitle,
  tone,
}: {
  icon: ReactNode;
  title: string;
  subtitle: string;
  tone: "accent" | "muted";
}) {
  return (
    <div className="flex items-start gap-2.5">
      <span
        className={cn(
          "mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-md",
          tone === "accent" ? "bg-cyan-400/10 text-cyan-300" : "bg-slate-700/40 text-slate-300",
        )}
      >
        {icon}
      </span>
      <div className="min-w-0">
        <h3 className="text-sm font-semibold text-slate-100">{title}</h3>
        <p className="mt-0.5 text-xs leading-snug text-slate-400">{subtitle}</p>
      </div>
    </div>
  );
}

function SessionKindChoices({
  kinds,
  selectedKind,
  onSelect,
}: {
  kinds: AgentSessionKind[];
  selectedKind: AgentSessionKind;
  onSelect: (kind: AgentSessionKind) => void;
}) {
  if (kinds.length === 0) {
    return <p className="text-xs text-amber-300">No compatible draft session types.</p>;
  }

  return (
    <div className="grid gap-1.5 sm:grid-cols-2" data-testid={selectors.agentSessions.entityAttachKindSelect}>
      {kinds.map((kind) => {
        const Icon = SESSION_KIND_ICONS[kind];
        const selected = kind === selectedKind;
        return (
          <button
            key={kind}
            type="button"
            aria-pressed={selected}
            onClick={() => onSelect(kind)}
            className={cn(
              "flex min-w-0 items-start gap-2 rounded-md border px-2.5 py-2 text-left transition-colors",
              selected
                ? "border-cyan-400/60 bg-cyan-400/10 text-cyan-50"
                : "border-slate-800 bg-slate-950/55 text-slate-200 hover:border-slate-700 hover:bg-slate-800/55",
            )}
          >
            <Icon className="mt-0.5 h-4 w-4 shrink-0 text-cyan-300" />
            <span className="min-w-0">
              <span className="block truncate text-sm font-medium leading-5">{SESSION_KIND_LABELS[kind]}</span>
              <span className="line-clamp-2 text-xs leading-4 text-slate-400">{SESSION_KIND_DESCRIPTIONS[kind]}</span>
            </span>
          </button>
        );
      })}
    </div>
  );
}

function AttachSessionRow({
  session,
  selected,
  compatible,
  onSelect,
}: {
  session: AgentSession;
  selected: boolean;
  compatible: boolean;
  onSelect: () => void;
}) {
  return (
    <SessionSummaryCard
      session={session}
      selection={{
        selectionMode: true,
        selected,
        disabled: !compatible,
        disabledReason: compatible ? undefined : "This session kind does not allow this context type.",
        onToggleSelect: onSelect,
      }}
    />
  );
}

export function AttachToSessionActionIcon() {
  return <MessageCirclePlus />;
}

function titleForQuickStart(option: SessionContextOption): string {
  const title = option.title.trim() || option.ref;
  return `Discuss ${title.length > 70 ? `${title.slice(0, 67)}...` : title}`;
}
