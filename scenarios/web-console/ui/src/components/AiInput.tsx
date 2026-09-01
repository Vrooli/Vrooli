// DOC: docs/concepts/ARCHITECTURE.md#file-map
// [REQ:P0-005b] AI Input UI Component
import { useState, useCallback, useRef, useEffect } from "react";
import { Sparkles, Send, Copy, Play, Loader2, AlertCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "./ui/button";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { generateAICommand } from "../api/ai";
import { strings } from "../consts/strings";
import { toErrorInfo } from "../lib/errors";
import { writeText } from "../lib/clipboard";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

/**
 * AI Input modal for generating shell commands from natural language.
 * Displays a prompt field, generates a command via the API, and allows
 * one-click execution or copy.
 *
 * [REQ:P0-005b] AI Input UI Component
 */
export default function AiInput({ onExecute }: { onExecute: (command: string) => void }) {
  const { t } = useTranslation();
  const aiModalOpen = useWorkspaceStore((s) => s.aiModalOpen);
  const setAiModalOpen = useWorkspaceStore((s) => s.setAiModalOpen);
  const activePane = useWorkspaceStore((s) => s.activePane);
  const hasActiveTerminal = activePane !== null;

  const [prompt, setPrompt] = useState("");
  const [command, setCommand] = useState<string | null>(null);
  const [provider, setProvider] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  // Track the active generation request so we can ignore stale responses
  const generationIdRef = useRef(0);

  // Cancel any in-flight generation on unmount by bumping the generation ID
  useEffect(() => {
    return () => { generationIdRef.current += 1; };
  }, []);

  // Auto-focus input when modal opens
  useEffect(() => {
    if (aiModalOpen) {
      // Delay to allow the modal to render
      const timer = setTimeout(() => inputRef.current?.focus(), 50);
      return () => clearTimeout(timer);
    }
  }, [aiModalOpen]);

  const handleGenerate = useCallback(async () => {
    const trimmed = prompt.trim();
    if (!trimmed) return;

    setIsLoading(true);
    setError(null);
    setCommand(null);
    setProvider(null);

    const thisGeneration = ++generationIdRef.current;
    try {
      const result = await generateAICommand(trimmed);
      // Discard result if a newer generation was started or component unmounted
      if (generationIdRef.current !== thisGeneration) return;
      setCommand(result.command);
      setProvider(result.provider);
    } catch (err) {
      if (generationIdRef.current !== thisGeneration) return;
      const info = toErrorInfo(err);
      setError(info.message);
    } finally {
      if (generationIdRef.current === thisGeneration) {
        setIsLoading(false);
      }
    }
  }, [prompt]);

  const handleExecute = useCallback(() => {
    if (command && hasActiveTerminal) {
      onExecute(command + "\n");
      setCommand(null);
      setProvider(null);
      setPrompt("");
      inputRef.current?.focus();
    }
  }, [command, hasActiveTerminal, onExecute]);

  const handleCopy = useCallback(() => {
    if (command) {
      void writeText(command);
    }
  }, [command]);

  // Enter key has two phases:
  //   1. No command yet → generate a command from the prompt
  //   2. Command ready  → execute it in the active terminal
  const handleEnterKey = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        const hasGeneratedCommand = command !== null;
        if (hasGeneratedCommand) {
          handleExecute();
        } else {
          handleGenerate();
        }
      }
    },
    [command, handleGenerate, handleExecute],
  );

  const close = () => setAiModalOpen(false);

  return (
    <ResponsiveDialog
      avoidKeyboard
      open={aiModalOpen}
      onClose={close}
      size="md"
      closeLabel={t(strings.aiInput.closeAriaLabel)}
      title={
        <span className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 shrink-0 text-wc-accent" />
          {t(strings.aiInput.heading)}
        </span>
      }
      testId="ai-input"
    >
      <div className="h-full overflow-y-auto p-3">
          <div className="flex items-center gap-2">
            <input
              ref={inputRef}
              data-testid="ai-input-prompt"
              type="text"
              className="flex-1 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-sm text-wc-text-primary placeholder:text-wc-text-faint outline-none focus:border-wc-accent"
              placeholder={t(strings.aiInput.promptPlaceholder)}
              value={prompt}
              onChange={(e) => {
                setPrompt(e.target.value);
                if (command) {
                  setCommand(null);
                  setProvider(null);
                }
              }}
              onKeyDown={handleEnterKey}
              disabled={isLoading}
            />
            <Button
              data-testid="ai-input-generate"
              variant="ghost"
              size="icon"
              shape="square"
              className="h-7 w-7"
              onClick={handleGenerate}
              disabled={isLoading || !prompt.trim()}
              title={t(strings.aiInput.generateTitle)}
            >
              {isLoading ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Send className="h-3.5 w-3.5" />
              )}
            </Button>
          </div>

          {error && (
            <div data-testid="ai-input-error" className="mt-1.5 flex items-center gap-1.5 text-xs text-wc-error-detail">
              <AlertCircle className="h-3 w-3 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {command && (
            <div data-testid="ai-input-result" className="mt-1.5 flex items-center gap-2 rounded bg-wc-surface-base px-2 py-1.5">
              <code className="flex-1 text-xs text-wc-text-primary font-mono truncate">
                {command}
              </code>
              {provider && (
                <span className="text-[10px] text-wc-text-faint shrink-0">
                  {t(strings.aiInput.viaProvider, { provider })}
                </span>
              )}
              <Button
                data-testid="ai-input-copy"
                variant="ghost"
                size="icon"
                shape="square"
                className="h-6 w-6 shrink-0"
                onClick={handleCopy}
                title={t(strings.aiInput.copyTitle)}
              >
                <Copy className="h-3 w-3" />
              </Button>
              <Button
                data-testid="ai-input-execute"
                variant="ghost"
                size="icon"
                shape="square"
                className="h-6 w-6 shrink-0"
                onClick={handleExecute}
                disabled={!hasActiveTerminal}
                title={hasActiveTerminal ? t(strings.aiInput.executeTitle) : t(strings.aiInput.noActiveTerminal)}
              >
                <Play className="h-3 w-3" />
              </Button>
            </div>
          )}
      </div>
    </ResponsiveDialog>
  );
}
