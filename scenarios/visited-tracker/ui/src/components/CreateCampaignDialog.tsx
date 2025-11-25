import { useState } from "react";
import { Plus } from "lucide-react";
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

  const handleSubmit = () => {
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
  };

  const resetForm = () => {
    setName("");
    setFromAgent("manual");
    setDescription("");
    setPatterns("");
  };

  const handleClose = () => {
    resetForm();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={handleClose}>
        <DialogHeader>
          <DialogTitle>Create New Campaign</DialogTitle>
          <DialogDescription>
            Set up a new file tracking campaign with custom patterns
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Campaign Name *</Label>
            <Input
              id="name"
              placeholder="e.g., Security Audit Q1"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-slate-500">A unique name for this campaign</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="fromAgent">Created By</Label>
            <Input
              id="fromAgent"
              placeholder="e.g., security-scanner"
              value={fromAgent}
              onChange={(e) => setFromAgent(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-slate-500">Agent or system that created this campaign</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              placeholder="Describe the purpose of this campaign..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={isLoading}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="patterns">File Patterns *</Label>
            <Input
              id="patterns"
              placeholder="e.g., **/*.go, scenarios/**/*.md, packages/ui/**/*.tsx"
              value={patterns}
              onChange={(e) => setPatterns(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-slate-500">
              Comma-separated glob patterns relative to project root
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isLoading || !name || !patterns}>
            <Plus className="mr-2 h-4 w-4" />
            {isLoading ? "Creating..." : "Create Campaign"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
