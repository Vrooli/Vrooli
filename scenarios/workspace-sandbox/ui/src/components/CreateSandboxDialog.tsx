import { useState } from "react";
import { FolderOpen, Loader2, Plus } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Input, Label } from "./ui/input";
import { Select } from "./ui/select";
import type { CreateRequest, OwnerType } from "../lib/api";
import { SELECTORS } from "../consts/selectors";

interface CreateSandboxDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (req: CreateRequest) => void;
  isCreating: boolean;
}

const OWNER_TYPE_OPTIONS = [
  { value: "agent", label: "Agent" },
  { value: "user", label: "User" },
  { value: "task", label: "Task" },
  { value: "system", label: "System" },
];

export function CreateSandboxDialog({
  open,
  onOpenChange,
  onCreate,
  isCreating,
}: CreateSandboxDialogProps) {
  const [scopePath, setScopePath] = useState("");
  const [projectRoot, setProjectRoot] = useState("");
  const [owner, setOwner] = useState("");
  const [ownerType, setOwnerType] = useState<OwnerType>("user");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!scopePath.trim()) return;

    onCreate({
      scopePath: scopePath.trim(),
      projectRoot: projectRoot.trim() || undefined,
      owner: owner.trim() || undefined,
      ownerType,
    });
  };

  const handleClose = () => {
    if (!isCreating) {
      onOpenChange(false);
      // Reset form
      setScopePath("");
      setProjectRoot("");
      setOwner("");
      setOwnerType("user");
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent data-testid={SELECTORS.createDialog}>
        <DialogClose onClose={handleClose} />

        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FolderOpen className="h-5 w-5 text-slate-400" />
            Create Sandbox
          </DialogTitle>
          <DialogDescription>
            Create an isolated workspace for safe file modifications. Changes can be
            reviewed and selectively applied.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          {/* Scope Path - Required */}
          <div className="space-y-2">
            <Label htmlFor="scopePath">
              Scope Path <span className="text-red-400">*</span>
            </Label>
            <Input
              id="scopePath"
              placeholder="/path/to/directory"
              value={scopePath}
              onChange={(e) => setScopePath(e.target.value)}
              required
              autoFocus
              data-testid={SELECTORS.scopePathInput}
            />
            <p className="text-xs text-slate-500">
              The directory path that will be writable in the sandbox
            </p>
          </div>

          {/* Project Root - Optional */}
          <div className="space-y-2">
            <Label htmlFor="projectRoot">Project Root</Label>
            <Input
              id="projectRoot"
              placeholder="/path/to/project (optional)"
              value={projectRoot}
              onChange={(e) => setProjectRoot(e.target.value)}
              data-testid={SELECTORS.projectRootInput}
            />
            <p className="text-xs text-slate-500">
              Full project root for read-only access. Defaults to scope path if empty.
            </p>
          </div>

          {/* Owner - Optional */}
          <div className="space-y-2">
            <Label htmlFor="owner">Owner</Label>
            <Input
              id="owner"
              placeholder="agent-001 or username (optional)"
              value={owner}
              onChange={(e) => setOwner(e.target.value)}
              data-testid={SELECTORS.ownerInput}
            />
          </div>

          {/* Owner Type */}
          <div className="space-y-2">
            <Label>Owner Type</Label>
            <Select
              value={ownerType}
              onValueChange={(v) => setOwnerType(v as OwnerType)}
              options={OWNER_TYPE_OPTIONS}
              data-testid={SELECTORS.ownerTypeSelect}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isCreating}
              data-testid={SELECTORS.cancelCreate}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isCreating || !scopePath.trim()}
              data-testid={SELECTORS.submitCreate}
            >
              {isCreating ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Sandbox
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
