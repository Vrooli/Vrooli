import { useState } from "react";
import { Copy, Check, Sparkles } from "lucide-react";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";

interface ActionButtonsProps {
  prompt: string;
  disabled: boolean;
}

export function ActionButtons({ prompt, disabled }: ActionButtonsProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (!prompt || disabled) return;

    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
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
        {copied ? (
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
        variant="outline"
        disabled
        className="flex-1 sm:flex-none opacity-50"
        data-testid={selectors.generate.spawnButton}
        title="Coming soon: Spawn AI agent to generate tests"
      >
        <Sparkles className="mr-2 h-4 w-4" />
        Spawn agent
      </Button>
    </div>
  );
}
