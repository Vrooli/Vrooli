import { useState } from "react";
import { Copy, Check } from "lucide-react";
import { Button } from "./ui/button";

interface CliCommandProps {
  command: string;
  description?: string;
  className?: string;
}

/**
 * Component that displays a CLI command with one-click copy functionality
 * Optimized for agent workflows - reduces friction in copying commands
 */
export function CliCommand({ command, description, className = "" }: CliCommandProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <div className={`group ${className}`}>
      {description && (
        <div className="text-xs sm:text-sm text-slate-300 mb-1.5 sm:mb-2">{description}</div>
      )}
      <div className="relative rounded-lg border border-white/10 bg-slate-900/80 hover:bg-slate-900 transition-colors">
        <code className="block px-3 sm:px-4 py-2 sm:py-2.5 pr-12 sm:pr-14 font-mono text-xs sm:text-sm text-blue-300 overflow-x-auto">
          {command}
        </code>
        <Button
          size="sm"
          variant="ghost"
          onClick={handleCopy}
          className="absolute right-1 top-1/2 -translate-y-1/2 h-7 sm:h-8 px-2 sm:px-2.5 hover:bg-white/10"
          aria-label="Copy command to clipboard"
        >
          {copied ? (
            <Check className="h-3.5 w-3.5 sm:h-4 sm:w-4 text-green-400" aria-hidden="true" />
          ) : (
            <Copy className="h-3.5 w-3.5 sm:h-4 sm:w-4 text-slate-400 group-hover:text-slate-200" aria-hidden="true" />
          )}
        </Button>
      </div>
    </div>
  );
}
