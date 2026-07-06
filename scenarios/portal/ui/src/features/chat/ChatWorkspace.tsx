import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Bot,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  FolderPlus,
  GitBranch,
  Pencil,
  Play,
  Plus,
  RefreshCcw,
  Send,
  Square,
  User,
} from "lucide-react";
import { CompletionEventKind, MessageRole } from "@vrooli/proto-types/portal/v1/message/message_pb";

import {
  AgentHarness,
  ChatMode,
  createPortalChat,
  createPortalGroup,
  editPortalMessage,
  getMessageTree,
  listChats,
  regeneratePortalMessage,
  sendPortalMessage,
  streamPortalCompletion,
  updatePortalGroupCollapsed,
  type Chat,
  type ChatGroup,
  type Message,
} from "../../api/chat";
import { Button } from "../../components/ui/button";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { cn } from "../../lib/utils";
import { EcosystemOmnibox } from "../search/EcosystemOmnibox";

const defaultModel = "openai/gpt-4.1-mini";
const fallbackGroupId = "__ungrouped__";
const groupColors = [
  "var(--color-primary)",
  "var(--color-success)",
  "var(--color-warning)",
  "var(--color-info)",
] as const;

interface BranchInfo {
  index: number;
  total: number;
}

function parseSkillIds(raw: string): string[] {
  return raw
    .split(",")
    .map((part) => part.trim())
    .filter((part) => part.length > 0);
}

function lastMessageId(messages: Message[]): string {
  const sorted = [...messages].sort((a, b) => a.createdAt.localeCompare(b.createdAt));
  return sorted.at(-1)?.id ?? "";
}

function roleLabelKey(role: MessageRole) {
  if (role === MessageRole.USER) {
    return strings.chat.message.roleUser;
  }
  if (role === MessageRole.AGENT) {
    return strings.chat.message.roleAgent;
  }
  return strings.chat.message.roleAssistant;
}

function roleIcon(role: MessageRole) {
  if (role === MessageRole.USER) {
    return <User aria-hidden className="h-4 w-4" />;
  }
  return <Bot aria-hidden className="h-4 w-4" />;
}

function renderMarkdownLite(content: string) {
  const blocks = content.split(/\n{2,}/);
  return blocks.map((block, index) => {
    const trimmed = block.trim();
    if (trimmed.startsWith("```") && trimmed.endsWith("```")) {
      return (
        <pre
          key={`${index}-${trimmed.length}`}
          className="overflow-x-auto rounded-control bg-app-background px-3 py-2 text-xs text-app-foreground"
        >
          <code>{trimmed.replace(/^```[a-zA-Z0-9_-]*\n?/, "").replace(/```$/, "")}</code>
        </pre>
      );
    }
    return (
      <p key={`${index}-${trimmed.length}`} className="whitespace-pre-wrap leading-relaxed">
        {trimmed}
      </p>
    );
  });
}

function branchInfoFor(message: Message, messages: Message[]): BranchInfo {
  const siblings = messages
    .filter((candidate) => candidate.parentMessageId === message.parentMessageId)
    .sort((a, b) => a.siblingIndex - b.siblingIndex);
  const index = Math.max(0, siblings.findIndex((candidate) => candidate.id === message.id));
  return { index, total: siblings.length };
}

function branchSiblingId(message: Message, messages: Message[], offset: number): string {
  const siblings = messages
    .filter((candidate) => candidate.parentMessageId === message.parentMessageId)
    .sort((a, b) => a.siblingIndex - b.siblingIndex);
  const index = siblings.findIndex((candidate) => candidate.id === message.id);
  return siblings.at(index + offset)?.id ?? message.id;
}

export function ChatWorkspace() {
  const { t } = useTranslation();
  const [chats, setChats] = useState<Chat[]>([]);
  const [groups, setGroups] = useState<ChatGroup[]>([]);
  const [selectedChatId, setSelectedChatId] = useState("");
  const [messages, setMessages] = useState<Message[]>([]);
  const [activeLeafMessageId, setActiveLeafMessageId] = useState("");
  const [draft, setDraft] = useState("");
  const [model, setModel] = useState(defaultModel);
  const [mode, setMode] = useState<ChatMode>(ChatMode.LLM);
  const [agentHarness, setAgentHarness] = useState<AgentHarness>(AgentHarness.CLAUDE_CODE);
  const [webSearchEnabled, setWebSearchEnabled] = useState(true);
  const [skillDraft, setSkillDraft] = useState("");
  const [streamText, setStreamText] = useState("");
  const [activityText, setActivityText] = useState("");
  const [loading, setLoading] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const [error, setError] = useState("");
  const abortRef = useRef<AbortController | null>(null);

  const selectedChat = useMemo(
    () => chats.find((chat) => chat.id === selectedChatId),
    [chats, selectedChatId],
  );

  const groupedChats = useMemo(() => {
    const knownGroups = [...groups].sort((a, b) => a.sortOrder - b.sortOrder);
    const buckets = knownGroups.map((group) => ({
      id: group.id,
      name: group.name,
      color: group.color,
      collapsed: group.collapsed,
      chats: chats.filter((chat) => chat.groupId === group.id),
    }));
    const ungrouped = chats.filter((chat) => !chat.groupId);
    return [
      ...buckets,
      {
        id: fallbackGroupId,
        name: t(strings.chat.sidebar.ungrouped),
        color: "var(--color-muted-foreground)",
        collapsed: false,
        chats: ungrouped,
      },
    ].filter((group) => group.chats.length > 0 || group.id !== fallbackGroupId);
  }, [chats, groups, t]);

  const activeMessages = useMemo(() => {
    if (!activeLeafMessageId) {
      return [...messages].sort((a, b) => a.createdAt.localeCompare(b.createdAt));
    }
    const byId = new Map(messages.map((message) => [message.id, message]));
    const path: Message[] = [];
    let current = byId.get(activeLeafMessageId);
    while (current) {
      path.push(current);
      current = current.parentMessageId ? byId.get(current.parentMessageId) : undefined;
    }
    return path.reverse();
  }, [activeLeafMessageId, messages]);

  const refreshChats = useCallback(async () => {
    const response = await listChats();
    setChats(response.chats);
    setGroups(response.groups);
    setSelectedChatId((current) => current || response.chats.at(0)?.id || "");
  }, []);

  const refreshTree = useCallback(async (chatId: string) => {
    if (!chatId) {
      setMessages([]);
      setActiveLeafMessageId("");
      return;
    }
    const response = await getMessageTree(chatId);
    setMessages(response.messages);
    setActiveLeafMessageId(response.activeLeafMessageId || lastMessageId(response.messages));
  }, []);

  useEffect(() => {
    let canceled = false;
    setLoading(true);
    void refreshChats()
      .catch((err: unknown) => {
        if (!canceled) {
          setError(errorMessage(err, t));
        }
      })
      .finally(() => {
        if (!canceled) {
          setLoading(false);
        }
      });
    return () => {
      canceled = true;
    };
  }, [refreshChats, t]);

  useEffect(() => {
    let canceled = false;
    if (!selectedChatId) {
      setMessages([]);
      setActiveLeafMessageId("");
      return;
    }
    void refreshTree(selectedChatId).catch((err: unknown) => {
      if (!canceled) {
        setError(errorMessage(err, t));
      }
    });
    return () => {
      canceled = true;
    };
  }, [refreshTree, selectedChatId, t]);

  useEffect(() => {
    if (!selectedChat) {
      return;
    }
    setModel(selectedChat.model || defaultModel);
    setMode(selectedChat.mode === ChatMode.AGENT ? ChatMode.AGENT : ChatMode.LLM);
    setAgentHarness(
      selectedChat.agentHarness === AgentHarness.UNSPECIFIED
        ? AgentHarness.CLAUDE_CODE
        : selectedChat.agentHarness,
    );
    setWebSearchEnabled(selectedChat.webSearchEnabled);
  }, [selectedChat]);

  const handleCreateChat = useCallback(async (nextMode: ChatMode) => {
    setError("");
    setLoading(true);
    try {
      const chat = await createPortalChat({
        title:
          nextMode === ChatMode.AGENT
            ? t(strings.chat.sidebar.newAgentTitle)
            : t(strings.chat.sidebar.newChatTitle),
        mode: nextMode,
        agentHarness,
        model,
        webSearchEnabled,
      });
      await refreshChats();
      setSelectedChatId(chat.id);
    } catch (err: unknown) {
      setError(errorMessage(err, t));
    } finally {
      setLoading(false);
    }
  }, [agentHarness, model, refreshChats, t, webSearchEnabled]);

  const handleCreateGroup = useCallback(async () => {
    setError("");
    setLoading(true);
    try {
      const color = groupColors[groups.length % groupColors.length] ?? groupColors[0];
      await createPortalGroup(t(strings.chat.sidebar.newGroupTitle), color);
      await refreshChats();
    } catch (err: unknown) {
      setError(errorMessage(err, t));
    } finally {
      setLoading(false);
    }
  }, [groups.length, refreshChats, t]);

  const handleToggleGroup = useCallback(async (group: ChatGroup) => {
    setError("");
    try {
      await updatePortalGroupCollapsed(group.id, !group.collapsed);
      await refreshChats();
    } catch (err: unknown) {
      setError(errorMessage(err, t));
    }
  }, [refreshChats, t]);

  const handleSubmit = useCallback(async () => {
    const content = draft.trim();
    if (!content || !selectedChatId || streaming) {
      return;
    }
    setDraft("");
    setError("");
    setStreamText("");
    setActivityText("");
    const selectedSkillIds = parseSkillIds(skillDraft);
    try {
      const userMessage = await sendPortalMessage({
        chatId: selectedChatId,
        parentMessageId: activeLeafMessageId,
        content,
        model,
        webSearchEnabled,
        selectedSkillIds,
      });
      await refreshTree(selectedChatId);
      const controller = new AbortController();
      abortRef.current = controller;
      setStreaming(true);
      let assistantText = "";
      for await (const event of streamPortalCompletion({
        chatId: selectedChatId,
        fromMessageId: userMessage.id,
        model,
        webSearchEnabled,
        selectedSkillIds,
        mode,
        agentHarness,
        signal: controller.signal,
      })) {
        if (event.kind === CompletionEventKind.TOKEN) {
          assistantText += event.text;
          setStreamText(assistantText);
        } else if (event.kind === CompletionEventKind.AGENT_ACTIVITY) {
          setActivityText((current) => `${current}${current ? "\n" : ""}${event.text}`);
        } else if (event.kind === CompletionEventKind.ERROR) {
          setError(event.errorMessage || event.errorCode);
        } else if (event.kind === CompletionEventKind.SEARCH_ATTACHMENT) {
          await refreshTree(selectedChatId);
        }
      }
      await refreshTree(selectedChatId);
    } catch (err: unknown) {
      if (!(err instanceof DOMException && err.name === "AbortError")) {
        setError(errorMessage(err, t));
      }
    } finally {
      abortRef.current = null;
      setStreaming(false);
      setStreamText("");
    }
  }, [
    activeLeafMessageId,
    agentHarness,
    draft,
    mode,
    model,
    refreshTree,
    selectedChatId,
    skillDraft,
    streaming,
    t,
    webSearchEnabled,
  ]);

  const handleStop = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  const handleEdit = useCallback(async (message: Message) => {
    const next = window.prompt(t(strings.chat.message.editPrompt), message.content);
    if (next === null || next.trim().length === 0) {
      return;
    }
    setError("");
    try {
      await editPortalMessage(message.id, next.trim());
      await refreshTree(message.chatId);
    } catch (err: unknown) {
      setError(errorMessage(err, t));
    }
  }, [refreshTree, t]);

  const handleRegenerate = useCallback(async (message: Message) => {
    setError("");
    try {
      await regeneratePortalMessage(message.id, model);
      await refreshTree(message.chatId);
    } catch (err: unknown) {
      setError(errorMessage(err, t));
    }
  }, [model, refreshTree, t]);

  return (
    <section
      data-testid={selectors.chat.workspace}
      aria-labelledby="portal-chat-heading"
      className="grid min-h-[42rem] gap-4 lg:grid-cols-[18rem_minmax(0,1fr)]"
    >
      <div
        data-testid={selectors.chat.sidebar}
        aria-label={t(strings.chat.sidebar.label)}
        className="flex min-h-0 flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-3"
      >
        <div className="flex items-center justify-between gap-2">
          <div>
            <h2 id="portal-chat-heading" className="text-lg font-semibold">
              {t(strings.chat.title)}
            </h2>
            <p className="text-xs text-app-muted-foreground">{t(strings.chat.sidebar.description)}</p>
          </div>
          <Button
            type="button"
            size="sm"
            variant="outline"
            data-testid={selectors.chat.newGroupButton}
            aria-label={t(strings.chat.sidebar.newGroup)}
            onClick={() => void handleCreateGroup()}
          >
            <FolderPlus aria-hidden className="h-4 w-4" />
          </Button>
        </div>
        <div className="grid grid-cols-2 gap-2">
          <Button
            type="button"
            size="sm"
            data-testid={selectors.chat.newChatButton}
            onClick={() => void handleCreateChat(ChatMode.LLM)}
          >
            <Plus aria-hidden className="mr-2 h-4 w-4" />
            {t(strings.chat.sidebar.newChat)}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            data-testid={selectors.chat.newAgentChatButton}
            onClick={() => void handleCreateChat(ChatMode.AGENT)}
          >
            <Bot aria-hidden className="mr-2 h-4 w-4" />
            {t(strings.chat.sidebar.newAgent)}
          </Button>
        </div>
        {error ? <p className="rounded-control bg-app-danger/10 p-2 text-sm text-app-danger">{error}</p> : null}
        <div className="min-h-0 flex-1 space-y-3 overflow-auto pr-1">
          {groupedChats.map((group) => (
            <div key={group.id} data-testid={selectors.chat.group({ id: group.id })} className="space-y-1">
              <button
                type="button"
                onClick={() => {
                  const sourceGroup = groups.find((candidate) => candidate.id === group.id);
                  if (sourceGroup) {
                    void handleToggleGroup(sourceGroup);
                  }
                }}
                className="flex w-full items-center gap-2 rounded-control px-2 py-1 text-left text-xs font-semibold uppercase text-app-muted-foreground hover:bg-app-surface-muted"
              >
                <span
                  aria-hidden
                  className="h-2.5 w-2.5 rounded-full"
                  style={{ backgroundColor: group.color }}
                />
                <span className="min-w-0 flex-1 truncate">{group.name}</span>
                {group.id !== fallbackGroupId ? <ChevronDown aria-hidden className="h-3 w-3" /> : null}
              </button>
              {!group.collapsed
                ? group.chats.map((chat) => (
                    <button
                      key={chat.id}
                      type="button"
                      data-testid={selectors.chat.chat({ id: chat.id })}
                      onClick={() => setSelectedChatId(chat.id)}
                      className={cn(
                        "flex w-full items-center gap-2 rounded-control px-3 py-2 text-left text-sm hover:bg-app-surface-muted",
                        chat.id === selectedChatId ? "bg-app-primary/10 text-app-primary" : "text-app-foreground",
                      )}
                    >
                      {chat.mode === ChatMode.AGENT ? (
                        <Bot aria-hidden className="h-4 w-4 shrink-0" />
                      ) : (
                        <GitBranch aria-hidden className="h-4 w-4 shrink-0" />
                      )}
                      <span className="min-w-0 flex-1 truncate">{chat.title}</span>
                    </button>
                  ))
                : null}
            </div>
          ))}
          {!loading && chats.length === 0 ? (
            <p className="rounded-control border border-dashed border-app-border p-3 text-sm text-app-muted-foreground">
              {t(strings.chat.sidebar.empty)}
            </p>
          ) : null}
        </div>
      </div>

      <div className="flex min-w-0 flex-col gap-3">
        <div className="grid gap-3 rounded-panel border border-app-border bg-app-surface p-3 xl:grid-cols-[minmax(0,1fr)_22rem]">
          <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
            <label className="flex flex-col gap-1 text-xs font-medium text-app-muted-foreground">
              <span>{t(strings.chat.controls.mode)}</span>
              <select
                data-testid={selectors.chat.modeSelect}
                value={String(mode)}
                onChange={(event) => setMode(Number(event.target.value))}
                className="rounded-control border border-app-border bg-app-background px-3 py-2 text-sm text-app-foreground"
              >
                <option value={String(ChatMode.LLM)}>{t(strings.chat.controls.modeLlm)}</option>
                <option value={String(ChatMode.AGENT)}>{t(strings.chat.controls.modeAgent)}</option>
              </select>
            </label>
            <label className="flex flex-col gap-1 text-xs font-medium text-app-muted-foreground">
              <span>{t(strings.chat.controls.model)}</span>
              <select
                data-testid={selectors.chat.modelSelect}
                value={model}
                onChange={(event) => setModel(event.target.value)}
                className="rounded-control border border-app-border bg-app-background px-3 py-2 text-sm text-app-foreground"
              >
                <option value="openai/gpt-4.1-mini">{t(strings.chat.controls.modelFast)}</option>
                <option value="anthropic/claude-3.5-sonnet">{t(strings.chat.controls.modelReasoning)}</option>
              </select>
            </label>
            <label className="flex flex-col gap-1 text-xs font-medium text-app-muted-foreground">
              <span>{t(strings.chat.controls.agentHarness)}</span>
              <select
                data-testid={selectors.chat.harnessSelect}
                value={String(agentHarness)}
                onChange={(event) => setAgentHarness(Number(event.target.value))}
                disabled={mode !== ChatMode.AGENT}
                className="rounded-control border border-app-border bg-app-background px-3 py-2 text-sm text-app-foreground disabled:opacity-60"
              >
                <option value={String(AgentHarness.CLAUDE_CODE)}>{t(strings.chat.controls.harnessClaude)}</option>
                <option value={String(AgentHarness.CODEX)}>{t(strings.chat.controls.harnessCodex)}</option>
                <option value={String(AgentHarness.OPENCODE)}>{t(strings.chat.controls.harnessOpencode)}</option>
                <option value={String(AgentHarness.GROK)}>{t(strings.chat.controls.harnessGrok)}</option>
              </select>
            </label>
            <label className="flex items-center gap-2 self-end rounded-control border border-app-border bg-app-background px-3 py-2 text-sm">
              <input
                data-testid={selectors.chat.webSearchToggle}
                type="checkbox"
                checked={webSearchEnabled}
                onChange={(event) => setWebSearchEnabled(event.target.checked)}
              />
              <span>{t(strings.chat.controls.webSearch)}</span>
            </label>
          </div>
          <EcosystemOmnibox />
        </div>

        <div
          data-testid={selectors.chat.messageList}
          className="min-h-[24rem] flex-1 overflow-auto rounded-panel border border-app-border bg-app-surface p-4"
        >
          {activeMessages.length === 0 ? (
            <div
              data-testid={selectors.chat.emptyState}
              className="flex h-full min-h-64 flex-col items-center justify-center gap-3 text-center text-app-muted-foreground"
            >
              <Bot aria-hidden className="h-10 w-10" />
              <p className="max-w-md text-sm">{t(strings.chat.empty.description)}</p>
            </div>
          ) : (
            <div className="space-y-4">
              {activeMessages.map((message) => {
                const branch = branchInfoFor(message, messages);
                return (
                  <article
                    key={message.id}
                    data-testid={selectors.chat.message({ id: message.id })}
                    className={cn(
                      "rounded-panel border border-app-border p-3",
                      message.role === MessageRole.USER ? "bg-app-background" : "bg-app-surface-muted",
                    )}
                  >
                    <header className="mb-2 flex flex-wrap items-center justify-between gap-2">
                      <div className="flex items-center gap-2 text-sm font-semibold">
                        {roleIcon(message.role)}
                        <span>{t(roleLabelKey(message.role))}</span>
                        {branch.total > 1 ? (
                          <span className="rounded-control bg-app-background px-2 py-0.5 text-xs text-app-muted-foreground">
                            {t(strings.chat.message.branchCount, {
                              current: branch.index + 1,
                              total: branch.total,
                            })}
                          </span>
                        ) : null}
                      </div>
                      <div className="flex items-center gap-1">
                        {branch.total > 1 ? (
                          <>
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              data-testid={selectors.chat.branchPrevious}
                              aria-label={t(strings.chat.message.previousBranch)}
                              disabled={branch.index === 0}
                              onClick={() => setActiveLeafMessageId(branchSiblingId(message, messages, -1))}
                            >
                              <ChevronLeft aria-hidden className="h-4 w-4" />
                            </Button>
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              data-testid={selectors.chat.branchNext}
                              aria-label={t(strings.chat.message.nextBranch)}
                              disabled={branch.index >= branch.total - 1}
                              onClick={() => setActiveLeafMessageId(branchSiblingId(message, messages, 1))}
                            >
                              <ChevronRight aria-hidden className="h-4 w-4" />
                            </Button>
                          </>
                        ) : null}
                        {message.role === MessageRole.USER ? (
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            data-testid={selectors.chat.editButton}
                            aria-label={t(strings.chat.message.edit)}
                            onClick={() => void handleEdit(message)}
                          >
                            <Pencil aria-hidden className="h-4 w-4" />
                          </Button>
                        ) : null}
                        {message.role !== MessageRole.USER ? (
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            data-testid={selectors.chat.regenerateButton}
                            aria-label={t(strings.chat.message.regenerate)}
                            onClick={() => void handleRegenerate(message)}
                          >
                            <RefreshCcw aria-hidden className="h-4 w-4" />
                          </Button>
                        ) : null}
                      </div>
                    </header>
                    <div className="space-y-2 text-sm">{renderMarkdownLite(message.content)}</div>
                    {message.searchAttachments.length > 0 ? (
                      <details className="mt-3 rounded-control border border-app-border bg-app-background p-2 text-sm">
                        <summary className="cursor-pointer font-medium">
                          {t(strings.chat.message.searchAttachments)}
                        </summary>
                        <div className="mt-2 space-y-2">
                          {message.searchAttachments.flatMap((attachment) =>
                            attachment.hits.map((hit, index) => (
                              <div key={`${attachment.id}-${hit.path}-${index}`} className="rounded-control p-2">
                                <p className="font-medium">{hit.title || hit.path}</p>
                                {hit.snippet ? (
                                  <p className="text-app-muted-foreground">{hit.snippet}</p>
                                ) : null}
                                <p className="text-xs text-app-muted-foreground">{hit.providerId}</p>
                              </div>
                            )),
                          )}
                        </div>
                      </details>
                    ) : null}
                  </article>
                );
              })}
              {activityText ? (
                <article className="rounded-panel border border-app-border bg-app-surface-muted p-3 text-sm">
                  <header className="mb-2 flex items-center gap-2 font-semibold">
                    <Play aria-hidden className="h-4 w-4" />
                    <span>{t(strings.chat.message.agentActivity)}</span>
                  </header>
                  <pre className="whitespace-pre-wrap text-app-muted-foreground">{activityText}</pre>
                </article>
              ) : null}
              {streamText ? (
                <article className="rounded-panel border border-app-border bg-app-surface-muted p-3 text-sm">
                  <header className="mb-2 flex items-center gap-2 font-semibold">
                    <Bot aria-hidden className="h-4 w-4" />
                    <span>{t(strings.chat.message.streaming)}</span>
                  </header>
                  <div className="space-y-2">{renderMarkdownLite(streamText)}</div>
                </article>
              ) : null}
            </div>
          )}
        </div>

        <form
          data-testid={selectors.chat.composer}
          className="rounded-panel border border-app-border bg-app-surface p-3"
          onSubmit={(event) => {
            event.preventDefault();
            void handleSubmit();
          }}
        >
          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_14rem]">
            <label className="flex flex-col gap-2 text-sm font-medium">
              <span>{t(strings.chat.composer.label)}</span>
              <Textarea
                data-testid={selectors.chat.composerInput}
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                placeholder={t(strings.chat.composer.placeholder)}
                rows={4}
                className="border-app-border bg-app-background text-app-foreground placeholder:text-app-muted-foreground focus:ring-app-primary"
              />
            </label>
            <div className="flex flex-col gap-3">
              <label className="flex flex-col gap-2 text-sm font-medium">
                <span>{t(strings.chat.composer.skills)}</span>
                <input
                  data-testid={selectors.chat.skillInput}
                  value={skillDraft}
                  onChange={(event) => setSkillDraft(event.target.value)}
                  placeholder={t(strings.chat.composer.skillsPlaceholder)}
                  className="rounded-control border border-app-border bg-app-background px-3 py-2 text-sm text-app-foreground"
                />
              </label>
              <div className="mt-auto flex gap-2">
                {streaming ? (
                  <Button
                    type="button"
                    variant="outline"
                    data-testid={selectors.chat.stopButton}
                    onClick={handleStop}
                  >
                    <Square aria-hidden className="mr-2 h-4 w-4" />
                    {t(strings.chat.composer.stop)}
                  </Button>
                ) : (
                  <Button
                    type="submit"
                    data-testid={selectors.chat.sendButton}
                    disabled={!selectedChatId || draft.trim().length === 0}
                  >
                    <Send aria-hidden className="mr-2 h-4 w-4" />
                    {t(strings.chat.composer.send)}
                  </Button>
                )}
              </div>
            </div>
          </div>
        </form>
      </div>
    </section>
  );
}
