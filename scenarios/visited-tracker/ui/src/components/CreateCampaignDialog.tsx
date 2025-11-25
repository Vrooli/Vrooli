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
            <div className="flex items-center gap-2">
              <Label htmlFor="name">Campaign Name *</Label>
              <HelpButton content="A descriptive name for your campaign. Examples: 'UX Improvements Q1', 'API Refactoring', 'Test Coverage Phase'" />
            </div>
            <Input
              id="name"
              placeholder="e.g., UX Improvements"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={isLoading}
            />
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
