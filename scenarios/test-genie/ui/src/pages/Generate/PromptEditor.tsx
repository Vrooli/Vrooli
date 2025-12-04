import { useState } from "react";
import { Edit2, RotateCcw } from "lucide-react";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";

interface PromptEditorProps {
  prompt: string;
  onPromptChange: (prompt: string) => void;
  defaultPrompt: string;
  titleSuffix?: string;
  summary?: string;
}

export function PromptEditor({ prompt, onPromptChange, defaultPrompt, titleSuffix, summary }: PromptEditorProps) {
  const [isEditing, setIsEditing] = useState(false);

  const handleReset = () => {
    onPromptChange(defaultPrompt);
    setIsEditing(false);
  };

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
          {!isEditing ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsEditing(true)}
            >
              <Edit2 className="mr-2 h-4 w-4" />
              Edit
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
            disabled={prompt === defaultPrompt}
          >
            <RotateCcw className="mr-2 h-4 w-4" />
            Reset
          </Button>
        </div>
      </div>

      <div className="mt-4">
        {isEditing ? (
          <textarea
            className="w-full min-h-[300px] rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-sm text-white font-mono focus:outline-none focus:ring-2 focus:ring-cyan-400"
            value={prompt}
            onChange={(e) => onPromptChange(e.target.value)}
            spellCheck={false}
          />
        ) : (
          <div
            className={cn(
              "rounded-xl border border-white/10 bg-black/30 p-4 text-sm font-mono whitespace-pre-wrap",
              "max-h-[400px] overflow-y-auto"
            )}
          >
            {prompt || (
              <span className="text-slate-500 italic">
                Select phases and enter a scenario name to generate a prompt
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
