import { useState } from "react";
import { Edit2, RotateCcw, Lock, Copy, Check } from "lucide-react";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";

interface PromptEditorProps {
  preamble: string;         // Immutable safety preamble (read-only)
  body: string;             // Editable task body
  defaultBody: string;      // Default body for reset
  onBodyChange: (body: string) => void;
  fullPrompt: string;       // Combined preamble + body for copying
  titleSuffix?: string;
  summary?: string;
}

export function PromptEditor({
  preamble,
  body,
  defaultBody,
  onBodyChange,
  fullPrompt,
  titleSuffix,
  summary
}: PromptEditorProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [showPreamble, setShowPreamble] = useState(true);
  const [copied, setCopied] = useState(false);

  const handleReset = () => {
    onBodyChange(defaultBody);
    setIsEditing(false);
  };

  const handleCopyFull = async () => {
    try {
      await navigator.clipboard.writeText(fullPrompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Ignore clipboard errors
    }
  };

  const hasCustomBody = body !== defaultBody;

  return (
    <div data-testid={selectors.generate.promptEditor}>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Prompt</p>
          <h3 className="mt-2 text-lg font-semibold">
            Generated prompt{titleSuffix ? ` • ${titleSuffix}` : ""}
          </h3>
          {summary && <p className="mt-1 text-xs text-slate-400">{summary}</p>}
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleCopyFull}
            title="Copy full prompt (preamble + body)"
          >
            {copied ? (
              <Check className="mr-2 h-4 w-4 text-emerald-400" />
            ) : (
              <Copy className="mr-2 h-4 w-4" />
            )}
            {copied ? "Copied" : "Copy"}
          </Button>
          {!isEditing ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsEditing(true)}
              title="Edit task body (security preamble is locked)"
            >
              <Edit2 className="mr-2 h-4 w-4" />
              Edit Task
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsEditing(false)}
            >
              Done
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleReset}
            disabled={!hasCustomBody}
          >
            <RotateCcw className="mr-2 h-4 w-4" />
            Reset
          </Button>
        </div>
      </div>

      {/* Toggle preamble visibility */}
      <div className="mt-3 flex items-center gap-2">
        <button
          type="button"
          onClick={() => setShowPreamble(!showPreamble)}
          className="flex items-center gap-2 text-xs text-slate-400 hover:text-slate-300 transition"
        >
          <Lock className="h-3 w-3" />
          <span>{showPreamble ? "Hide" : "Show"} security preamble</span>
        </button>
        {hasCustomBody && (
          <span className="rounded-full border border-cyan-400/40 bg-cyan-400/10 px-2 py-0.5 text-[10px] text-cyan-300">
            Customized
          </span>
        )}
      </div>

      <div className="mt-4 space-y-3">
        {/* Immutable Safety Preamble */}
        {showPreamble && preamble && (
          <div className="relative">
            <div className="absolute -top-2 left-3 flex items-center gap-1 bg-slate-900 px-2 text-[10px] uppercase tracking-wider text-amber-400/80">
              <Lock className="h-3 w-3" />
              <span>Security constraints (read-only)</span>
            </div>
            <div
              className={cn(
                "rounded-xl border border-amber-400/30 bg-amber-950/20 p-4 pt-5 text-sm font-mono whitespace-pre-wrap",
                "text-amber-100/70"
              )}
            >
              {preamble}
            </div>
          </div>
        )}

        {/* Editable Task Body */}
        <div className="relative">
          {isEditing ? (
            <>
              <div className="absolute -top-2 left-3 flex items-center gap-1 bg-slate-900 px-2 text-[10px] uppercase tracking-wider text-cyan-400/80">
                <Edit2 className="h-3 w-3" />
                <span>Task instructions (editable)</span>
              </div>
              <textarea
                className="w-full min-h-[350px] rounded-xl border border-cyan-400/30 bg-black/30 px-4 py-5 text-sm text-white font-mono focus:outline-none focus:ring-2 focus:ring-cyan-400"
                value={body}
                onChange={(e) => onBodyChange(e.target.value)}
                spellCheck={false}
              />
            </>
          ) : (
            <>
              <div className="absolute -top-2 left-3 flex items-center gap-1 bg-slate-900 px-2 text-[10px] uppercase tracking-wider text-slate-400">
                <span>Task instructions</span>
              </div>
              <div
                className={cn(
                  "rounded-xl border border-white/10 bg-black/30 p-4 pt-5 text-sm font-mono whitespace-pre-wrap",
                  "max-h-[400px] overflow-y-auto"
                )}
              >
                {body || (
                  <span className="text-slate-500 italic">
                    Select phases and enter a scenario name to generate a prompt
                  </span>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
