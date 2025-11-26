import { useState, useEffect, useCallback } from "react";
import { Plus, Info } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "./ui/dialog";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import { Button } from "./ui/button";
import { HelpButton } from "./HelpButton";

interface CreateCampaignDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: {
    name: string;
    from_agent: string;
    description: string;
    patterns: string[];
  }) => void;
  isLoading?: boolean;
}

export function CreateCampaignDialog({
  open,
  onOpenChange,
  onSubmit,
  isLoading
}: CreateCampaignDialogProps) {
  const [name, setName] = useState("");
  const [fromAgent, setFromAgent] = useState("manual");
  const [description, setDescription] = useState("");
  const [patterns, setPatterns] = useState("");
  const [errors, setErrors] = useState<{ name?: string; patterns?: string }>({});

  const validateForm = () => {
    const newErrors: { name?: string; patterns?: string } = {};

    if (!name.trim()) {
      newErrors.name = "Campaign name is required";
    } else if (name.trim().length < 3) {
      newErrors.name = "Campaign name must be at least 3 characters";
    }

    if (!patterns.trim()) {
      newErrors.patterns = "At least one file pattern is required";
    } else {
      const patternArray = patterns.split(",").map((p) => p.trim()).filter((p) => p);
      if (patternArray.length === 0) {
        newErrors.patterns = "At least one valid pattern is required";
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = useCallback(() => {
    if (!validateForm()) {
      return;
    }

    const patternArray = patterns
      .split(",")
      .map((p) => p.trim())
      .filter((p) => p);

    onSubmit({
      name,
      from_agent: fromAgent,
      description,
      patterns: patternArray
    });
  }, [name, fromAgent, description, patterns, onSubmit]);

  const resetForm = () => {
    setName("");
    setFromAgent("manual");
    setDescription("");
    setPatterns("");
    setErrors({});
  };

  const handleClose = () => {
    resetForm();
    onOpenChange(false);
  };

  // Keyboard shortcut: Ctrl+Enter to submit
  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Enter' && e.ctrlKey && !isLoading) {
        e.preventDefault();
        handleSubmit();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [open, isLoading, handleSubmit]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={handleClose}>
        <DialogHeader>
          <DialogTitle>Create New Campaign</DialogTitle>
          <DialogDescription>
            Set up a new file tracking campaign with custom patterns
          </DialogDescription>
        </DialogHeader>

        <div className="mb-3 sm:mb-4 rounded-lg border border-blue-500/20 bg-blue-500/5 p-3 sm:p-4">
          <div className="flex items-start gap-2 text-xs sm:text-sm text-slate-300">
            <Info className="h-4 w-4 text-blue-400 shrink-0 mt-0.5" aria-hidden="true" />
            <div className="space-y-1">
              <div className="font-medium text-slate-200">For agents: CLI auto-creation is usually faster</div>
              <div className="text-[10px] sm:text-xs text-slate-400">
                Try <code className="px-1 py-0.5 rounded bg-slate-800 text-blue-300">visited-tracker least-visited --location DIR --pattern GLOB --tag TAG</code> instead
              </div>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Label htmlFor="name">Campaign Name *</Label>
              <HelpButton content="A descriptive name for your campaign. Examples: 'UX Improvements Q1', 'API Refactoring', 'Test Coverage Phase'" />
            </div>
            <Input
              id="name"
              placeholder="e.g., UX Improvements"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                if (errors.name && e.target.value.trim().length >= 3) {
                  setErrors({ ...errors, name: undefined });
                }
              }}
              disabled={isLoading}
              aria-invalid={!!errors.name}
              aria-describedby={errors.name ? "name-error" : undefined}
              className={errors.name ? "border-red-500/50" : ""}
            />
            {errors.name && (
              <p id="name-error" className="text-xs text-red-400" role="alert">
                {errors.name}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Label htmlFor="fromAgent">Agent / Creator</Label>
              <HelpButton content="Identifier for the agent or person running this campaign. Used to group work by agent type (ux, refactor, test, etc.)" />
            </div>
            <Input
              id="fromAgent"
              placeholder="e.g., ux-agent, manual, ecosystem-manager"
              value={fromAgent}
              onChange={(e) => setFromAgent(e.target.value)}
              disabled={isLoading}
            />
          </div>

          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Label htmlFor="description">Description</Label>
              <HelpButton content="Optional description explaining the campaign's goals or scope. Helps identify the purpose when viewing campaign lists." />
            </div>
            <Textarea
              id="description"
              placeholder="e.g., Systematic UI improvements across all scenario pages, focusing on accessibility and visual hierarchy"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={isLoading}
              rows={3}
            />
          </div>

          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Label htmlFor="patterns">File Patterns *</Label>
              <HelpButton content="Glob patterns to match files for tracking. Use ** for recursive search, * for wildcards. Examples: **/*.tsx (all TypeScript files), scenarios/*/api/*.go (Go files in scenario APIs)" />
            </div>
            <Input
              id="patterns"
              placeholder="**/*.tsx, api/**/*.go"
              value={patterns}
              onChange={(e) => {
                setPatterns(e.target.value);
                if (errors.patterns && e.target.value.trim()) {
                  setErrors({ ...errors, patterns: undefined });
                }
              }}
              disabled={isLoading}
              aria-invalid={!!errors.patterns}
              aria-describedby={errors.patterns ? "patterns-error" : "patterns-help"}
              className={errors.patterns ? "border-red-500/50" : ""}
            />
            {errors.patterns ? (
              <p id="patterns-error" className="text-xs text-red-400" role="alert">
                {errors.patterns}
              </p>
            ) : (
              <div id="patterns-help" className="space-y-1.5">
                <p className="text-xs text-slate-500">
                  Comma-separated glob patterns relative to project root
                </p>
                <details className="text-xs">
                  <summary className="cursor-pointer text-blue-400 hover:text-blue-300 transition-colors">
                    Show pattern examples
                  </summary>
                  <div className="mt-2 space-y-1 pl-4 border-l-2 border-blue-500/20">
                    <div className="text-slate-400">
                      <code className="text-blue-300">**/*.tsx</code> - All TSX files
                    </div>
                    <div className="text-slate-400">
                      <code className="text-blue-300">scenarios/*/ui/**/*.tsx</code> - UI files in scenarios
                    </div>
                    <div className="text-slate-400">
                      <code className="text-blue-300">api/**/*.go</code> - Go files in API directories
                    </div>
                    <div className="text-slate-400">
                      <code className="text-blue-300">**/*.test.ts</code> - All test files
                    </div>
                  </div>
                </details>
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <div className="flex w-full items-center justify-between">
            <p className="text-xs text-slate-600">
              <kbd className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-400">Ctrl+Enter</kbd> to submit
            </p>
            <div className="flex gap-2">
              <Button variant="outline" onClick={handleClose} disabled={isLoading}>
                Cancel
              </Button>
              <Button onClick={handleSubmit} disabled={isLoading}>
                <Plus className="mr-2 h-4 w-4" />
                {isLoading ? "Creating..." : "Create Campaign"}
              </Button>
            </div>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
