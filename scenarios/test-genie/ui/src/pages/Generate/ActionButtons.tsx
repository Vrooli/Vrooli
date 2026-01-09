import { useState } from "react";
import { Copy, Check, Sparkles } from "lucide-react";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";

interface ActionButtonsProps {
  prompt: string;
  allPrompts: string[];
  disabled: boolean;
  spawnDisabled?: boolean;
  spawnBusy?: boolean;
  onSpawnAll?: () => void;
}

export function ActionButtons({
  prompt,
  allPrompts,
  disabled,
  spawnDisabled = true,
  spawnBusy = false,
  onSpawnAll
}: ActionButtonsProps) {
  const [copied, setCopied] = useState<"one" | "all" | null>(null);

  const handleCopy = async () => {
    if (!prompt || disabled) return;

    try {
      await navigator.clipboard.writeText(prompt);
      setCopied("one");
      setTimeout(() => setCopied(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  const handleCopyAll = async () => {
    if (!allPrompts || allPrompts.length === 0 || disabled) return;
    const serialized = allPrompts
      .map((p, idx) => `# Prompt ${idx + 1}\n\n${p}`)
      .join("\n\n---\n\n");

    try {
      await navigator.clipboard.writeText(serialized);
      setCopied("all");
      setTimeout(() => setCopied(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  return (
    <div className="flex flex-wrap gap-3">
      <Button
        onClick={handleCopy}
        disabled={disabled || !prompt}
        className="flex-1 sm:flex-none"
        data-testid={selectors.generate.copyButton}
      >
        {copied === "one" ? (
          <>
            <Check className="mr-2 h-4 w-4" />
            Copied!
          </>
        ) : (
          <>
            <Copy className="mr-2 h-4 w-4" />
            Copy prompt
          </>
        )}
      </Button>

      <Button
        onClick={handleCopyAll}
        disabled={disabled || !allPrompts || allPrompts.length === 0}
        className="flex-1 sm:flex-none"
        variant="outline"
        data-testid={selectors.generate.copyAllButton}
      >
        {copied === "all" ? (
          <>
            <Check className="mr-2 h-4 w-4" />
            Copied all
          </>
        ) : (
          <>
            <Copy className="mr-2 h-4 w-4" />
            Copy all prompts
          </>
        )}
      </Button>

      <Button
        variant="outline"
        disabled={spawnDisabled}
        className="flex-1 sm:flex-none"
        data-testid={selectors.generate.spawnButton}
        title={spawnDisabled ? "Configure a model to spawn agents" : "Spawn agents for these prompts"}
        onClick={() => onSpawnAll?.()}
      >
        <Sparkles className="mr-2 h-4 w-4" />
        {spawnBusy ? "Spawning…" : "Spawn agents"}
      </Button>
    </div>
  );
}
