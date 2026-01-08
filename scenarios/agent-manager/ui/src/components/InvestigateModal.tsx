import { useEffect, useState } from "react";
import { AlertCircle, Search } from "lucide-react";
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

interface InvestigateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  confirmLabel: string;
  onSubmit: (customContext: string) => Promise<void>;
  loading?: boolean;
  error?: string | null;
}

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

  useEffect(() => {
    if (!open) {
      setCustomContext("");
    }
  }, [open]);

  const handleSubmit = async () => {
    await onSubmit(customContext.trim());
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
          {/* Custom Context */}
          <div className="space-y-2">
            <Label htmlFor="customContext">Additional Context (optional)</Label>
            <Textarea
              id="customContext"
              value={customContext}
              onChange={(e) => setCustomContext(e.target.value)}
              placeholder="Provide any additional context for this run..."
              rows={3}
            />
            <p className="text-xs text-muted-foreground">
              Share any extra details that should guide the agent.
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
