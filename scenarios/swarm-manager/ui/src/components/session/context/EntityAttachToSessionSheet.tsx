import { useEffect, useMemo, useState, type ReactNode } from "react";
import { ListPlus, MessageCirclePlus, Plus, Search } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { sessionDetailPath } from "../../../app/routes/route-paths";
import { selectors } from "../../../consts/selectors";
import { useAgentSessionStore } from "../../../stores";
import { proposalSessionService } from "../../../services/proposal-session-service";
import type { AgentSession, CreatableAgentSessionKind } from "../../../types";
import { cn } from "../../../lib/utils";
import { BottomSheet } from "../../ui/bottom-sheet";
import { Button } from "../../ui/button";
import { Input } from "../../ui/input";
import { ContextChipTray } from "../../composer/ContextChipTray";
import {
  SESSION_KIND_DESCRIPTIONS,
  SESSION_KIND_ICONS,
  SESSION_KIND_LABELS,
} from "../session-view-model";
import { SessionSummaryCard } from "../session-summary-card";
import { attachStarterSuggestions } from "../session-starter-suggestions";
import { writeSessionDraft } from "../session-draft-storage";
import { CONTEXT_TYPE_LABELS, compatibleSessionKindsForContextType, sessionKindAllowsContextType } from "./session-context-config";
import { stageContextForSession } from "./pending-session-context";
import { type SessionContextOption } from "./session-context-refs";

type AttachMode = "new" | "existing";

interface EntityAttachToSessionSheetProps {
  isOpen: boolean;
  onClose: () => void;
  option: SessionContextOption;
  currentSessionId?: string;
  /**
   * Constrains a new session to the reviewed mutation-list proposal flow.
   * Existing sessions stay available so an operator can continue work already
   * in progress, but a new session is always created through the proposal API.
   */
  proposalMode?: boolean;
}

export function EntityAttachToSessionSheet({
  isOpen,
  onClose,
  option,
  currentSessionId,
  proposalMode = false,
}: EntityAttachToSessionSheetProps) {
  if (!isOpen) return null;
  return (
    <EntityAttachToSessionSheetContent
      isOpen={isOpen}
      onClose={onClose}
      option={option}
      currentSessionId={currentSessionId}
      proposalMode={proposalMode}
    />
  );
}

function EntityAttachToSessionSheetContent({
  isOpen,
  onClose,
  option,
  currentSessionId,
  proposalMode = false,
}: EntityAttachToSessionSheetProps) {
  const navigate = useNavigate();
  const fetchSessions = useAgentSessionStore((s) => s.fetchSessions);
  const sessions = useAgentSessionStore((s) => s.sessions);
  const createSession = useAgentSessionStore((s) => s.createSession);
  const isMutating = useAgentSessionStore((s) => s.isMutating);
  const [mode, setMode] = useState<AttachMode>("new");
  const [query, setQuery] = useState("");
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [selectedKind, setSelectedKind] = useState<CreatableAgentSessionKind>(() => compatibleSessionKindsForContextType(option.type)[0] ?? "meta_orchestration");
  const [selectedSuggestionId, setSelectedSuggestionId] = useState<string | null>(null);
  const [localError, setLocalError] = useState<string | null>(null);

  const compatibleKinds = useMemo(() => compatibleSessionKindsForContextType(option.type), [option.type]);

  useEffect(() => {
    setMode("new");
    setSelectedSessionId(null);
    setSelectedKind(compatibleKinds[0] ?? "meta_orchestration");
    setSelectedSuggestionId(null);
    setQuery("");
    setLocalError(null);
    void fetchSessions({ limit: 100 }, { force: true });
  }, [compatibleKinds, fetchSessions, isOpen]);

  const suggestions = useMemo(
    () => attachStarterSuggestions(selectedKind, option).filter((suggestion) => !proposalMode || suggestion.proposalFlavor === "mutation_list"),
    [option, proposalMode, selectedKind],
  );
  const selectedSuggestion = suggestions.find((suggestion) => suggestion.id === selectedSuggestionId) ?? null;

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
      const createsProposal = proposalMode || selectedSuggestion?.proposalFlavor === "mutation_list";
      const session = createsProposal && (option.type === "initiative" || option.type === "backlog_item")
        ? await proposalSessionService.create({
          title: `Proposal for ${option.title || option.ref}`,
          target: { type: option.type, ref: option.ref, name: option.title || option.ref },
        })
        : await createSession({ kind: selectedKind, title: titleForQuickStart(option) });
      stageContextForSession(session.id, option);
      if (selectedSuggestion) {
        // The detail page restores this like any saved composer draft.
        writeSessionDraft(session.id, selectedSuggestion.text);
      } else if (proposalMode) {
        writeSessionDraft(session.id, `Review ${option.title || option.ref} and return a validated mutation_list proposal.`);
      }
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
      title={proposalMode ? "Start proposal" : "Attach to session"}
      description={proposalMode ? "Choose a proposal type, then review the resulting mutation list before anything changes." : "Stage this entity in a session composer."}
      className="!max-w-3xl border-slate-700/80 bg-slate-900"
      contentClassName="px-0 py-0"
      footer={
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="min-w-0 truncate text-xs text-slate-400">
            {mode === "new"
              ? selectedSuggestion
                ? "The chosen prompt will prefill the first message."
                : proposalMode
                  ? "A proposal session starts with a structured mutation-list request."
                  : "Context is staged before the first message."
              : selectedSession
                ? `Selected ${selectedSession.title || selectedSession.id}`
                : "Pick a session from the list."}
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" size="sm" onClick={onClose}>Cancel</Button>
            {mode === "new" ? (
              <Button
                size="sm"
                onClick={() => void quickStart()}
                disabled={isMutating || compatibleKinds.length === 0}
                data-testid={selectors.agentSessions.entityAttachQuickStart}
              >
                <Plus className="mr-1.5 h-3.5 w-3.5" />
                {proposalMode ? "Start proposal" : "Draft session"}
              </Button>
            ) : (
              <Button size="sm" onClick={attachToExisting} disabled={!canAttachSelected} data-testid={selectors.agentSessions.entityAttachConfirm}>
                Attach
              </Button>
            )}
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

          {/* Destination: fresh session or one that already exists */}
          <div className="grid gap-1.5 sm:grid-cols-2">
            <ModeCard
              icon={<MessageCirclePlus className="h-4 w-4" />}
              title={proposalMode ? "New proposal session" : "New session"}
              subtitle={proposalMode ? "Choose a proposal type before the first message." : `${CONTEXT_TYPE_LABELS[option.type]} context will be staged before the first message.`}
              selected={mode === "new"}
              onSelect={() => setMode("new")}
              testId={selectors.agentSessions.entityAttachModeNew}
            />
            <ModeCard
              icon={<ListPlus className="h-4 w-4" />}
              title={proposalMode ? "Continue existing session" : "Add to existing session"}
              subtitle="Stage this context in a session you already have open."
              selected={mode === "existing"}
              onSelect={() => setMode("existing")}
              testId={selectors.agentSessions.entityAttachModeExisting}
            />
          </div>

          {mode === "existing" && (
            <div className="relative" data-testid="entity-attach-existing-section">
              <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search sessions..."
                className="h-9 border-slate-700 bg-slate-950/70 pl-9 text-slate-100 placeholder:text-slate-500"
                data-testid={selectors.agentSessions.entityAttachSearch}
              />
            </div>
          )}
        </div>

        {mode === "new" ? (
          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-3" data-testid="entity-attach-new-section">
            {proposalMode ? (
              <div className="rounded-md border border-violet-400/20 bg-violet-400/5 px-3 py-2 text-sm text-slate-300">
                Proposal sessions use managed Swarm Operations and always produce a reviewable mutation list.
              </div>
            ) : (
              <div>
                <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wider text-slate-500">
                  Session type
                </p>
                <SessionKindChoices
                  kinds={compatibleKinds}
                  selectedKind={selectedKind}
                  onSelect={(kind) => {
                    setSelectedKind(kind);
                    // Suggestion lists differ per kind; a stale pick would prefill
                    // a prompt the user never saw.
                    setSelectedSuggestionId(null);
                  }}
                />
              </div>
            )}
            {suggestions.length > 0 && (
              <div>
                <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wider text-slate-500">
                  {proposalMode ? "Proposal type · optional" : "Start with · optional"}
                </p>
                <div className="space-y-3">
                  {[...new Set(suggestions.map((suggestion) => suggestion.group ?? "Discover"))].map((group) => (
                    <div key={group} className="space-y-1.5">
                      <p className="text-[11px] font-medium uppercase tracking-wider text-slate-500">{group}</p>
                      {suggestions.filter((suggestion) => (suggestion.group ?? "Discover") === group).map((suggestion) => {
                    const Icon = suggestion.icon;
                    const selected = suggestion.id === selectedSuggestionId;
                    return (
                      <button
                        key={suggestion.id}
                        type="button"
                        aria-pressed={selected}
                        onClick={() => setSelectedSuggestionId(selected ? null : suggestion.id)}
                        className={cn(
                          "flex w-full items-start gap-2.5 rounded-md border px-2.5 py-2 text-left transition-colors",
                          selected
                            ? "border-cyan-400/60 bg-cyan-400/10 text-cyan-50"
                            : "border-slate-800 bg-slate-950/45 text-slate-200 hover:border-slate-700 hover:bg-slate-800/55",
                        )}
                        data-testid={selectors.agentSessions.entityAttachSuggestion}
                      >
                        <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-cyan-500/20 bg-cyan-500/10 text-cyan-200">
                          <Icon className="h-3.5 w-3.5" />
                        </span>
                        <span className="min-w-0"><span className="block text-sm leading-5">{suggestion.text}</span>{suggestion.detail && <span className="mt-0.5 block text-xs leading-4 text-slate-400">{suggestion.detail}</span>}</span>
                      </button>
                    );
                  })}</div>
                  ))}
                </div>
              </div>
            )}
          </div>
        ) : (
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
        )}
      </div>
    </BottomSheet>
  );
}

function ModeCard({
  icon,
  title,
  subtitle,
  selected,
  onSelect,
  testId,
}: {
  icon: ReactNode;
  title: string;
  subtitle: string;
  selected: boolean;
  onSelect: () => void;
  testId: string;
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onSelect}
      className={cn(
        "flex min-w-0 items-start gap-2 rounded-md border px-2.5 py-2 text-left transition-colors",
        selected
          ? "border-cyan-400/60 bg-cyan-400/10 text-cyan-50"
          : "border-slate-800 bg-slate-950/55 text-slate-200 hover:border-slate-700 hover:bg-slate-800/55",
      )}
      data-testid={testId}
    >
      <span
        className={cn(
          "mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-md",
          selected ? "bg-cyan-400/10 text-cyan-300" : "bg-slate-700/40 text-slate-300",
        )}
      >
        {icon}
      </span>
      <span className="min-w-0">
        <span className="block truncate text-sm font-medium leading-5">{title}</span>
        <span className="line-clamp-2 text-xs leading-4 text-slate-400">{subtitle}</span>
      </span>
    </button>
  );
}

function SessionKindChoices({
  kinds,
  selectedKind,
  onSelect,
}: {
  kinds: CreatableAgentSessionKind[];
  selectedKind: CreatableAgentSessionKind;
  onSelect: (kind: CreatableAgentSessionKind) => void;
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
  return <MessageCirclePlus className="h-4 w-4" />;
}

function titleForQuickStart(option: SessionContextOption): string {
  const title = option.title.trim() || option.ref;
  return `Discuss ${title.length > 70 ? `${title.slice(0, 67)}...` : title}`;
}
