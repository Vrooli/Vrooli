import { useEffect, useState } from "react";
import { AlertCircle, Search, Zap, Settings, Microscope } from "lucide-react";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import type { InvestigationDepth } from "../types";

interface InvestigateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  confirmLabel: string;
  onSubmit: (customContext: string, depth: InvestigationDepth) => Promise<void>;
  loading?: boolean;
  error?: string | null;
}

const depthOptions: {
  value: InvestigationDepth;
  label: string;
  description: string;
  icon: React.ReactNode;
}[] = [
  {
    value: "quick",
    label: "Quick",
    description: "Fast analysis of error messages and immediate causes (2-3 min)",
    icon: <Zap className="h-4 w-4" />,
  },
  {
    value: "standard",
    label: "Standard",
    description: "Balanced analysis with targeted code exploration",
    icon: <Settings className="h-4 w-4" />,
  },
  {
    value: "deep",
    label: "Deep",
    description: "Thorough investigation exploring all relevant code paths",
    icon: <Microscope className="h-4 w-4" />,
  },
];

export function InvestigateModal({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  onSubmit,
  loading = false,
  error = null,
}: InvestigateModalProps) {
  const [customContext, setCustomContext] = useState("");
  const [depth, setDepth] = useState<InvestigationDepth>("standard");

  useEffect(() => {
    if (!open) {
      setCustomContext("");
      setDepth("standard");
    }
  }, [open]);

  const handleSubmit = async () => {
    await onSubmit(customContext.trim(), depth);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader onClose={() => onOpenChange(false)}>
          <DialogTitle className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            {title}
          </DialogTitle>
          {description && (
            <DialogDescription>
              {description}
            </DialogDescription>
          )}
        </DialogHeader>

        <DialogBody className="space-y-6">
          {/* Investigation Depth */}
          <div className="space-y-3">
            <Label>Investigation Depth</Label>
            <div className="grid gap-2">
              {depthOptions.map((option) => (
                <label
                  key={option.value}
                  className={`flex items-start gap-3 rounded-lg border p-3 cursor-pointer transition-colors ${
                    depth === option.value
                      ? "border-primary bg-primary/5"
                      : "border-border hover:border-primary/50"
                  }`}
                >
                  <input
                    type="radio"
                    name="depth"
                    value={option.value}
                    checked={depth === option.value}
                    onChange={(e) => setDepth(e.target.value as InvestigationDepth)}
                    className="mt-1"
                  />
                  <div className="flex-1">
                    <div className="flex items-center gap-2 font-medium">
                      {option.icon}
                      {option.label}
                    </div>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {option.description}
                    </p>
                  </div>
                </label>
              ))}
            </div>
          </div>

          {/* Custom Context */}
          <div className="space-y-2">
            <Label htmlFor="customContext">Additional Context (optional)</Label>
            <Textarea
              id="customContext"
              value={customContext}
              onChange={(e) => setCustomContext(e.target.value)}
              placeholder="Provide any additional context for this investigation..."
              rows={3}
            />
            <p className="text-xs text-muted-foreground">
              Share any extra details, suspected causes, or specific areas to investigate.
            </p>
          </div>

          {error && (
            <div className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <AlertCircle className="h-4 w-4 mt-0.5" />
              <span>{error}</span>
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={loading}
            className="gap-2"
          >
            {loading ? (
              "Starting..."
            ) : (
              <>
                <Search className="h-4 w-4" />
                {confirmLabel}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
