import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "../ui/dialog";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Badge } from "../ui/badge";
import type { Label } from "../../lib/api";

interface LabelManagerProps {
  open: boolean;
  onClose: () => void;
  labels: Label[];
  onCreateLabel: (data: { name: string; color: string }) => void;
  onDeleteLabel: (id: string) => void;
}

const PRESET_COLORS = [
  "#6366f1", // Indigo
  "#8b5cf6", // Violet
  "#ec4899", // Pink
  "#ef4444", // Red
  "#f97316", // Orange
  "#eab308", // Yellow
  "#22c55e", // Green
  "#06b6d4", // Cyan
  "#3b82f6", // Blue
  "#64748b", // Slate
] as const;

const DEFAULT_COLOR = PRESET_COLORS[0];

export function LabelManager({
  open,
  onClose,
  labels,
  onCreateLabel,
  onDeleteLabel,
}: LabelManagerProps) {
  const [newLabelName, setNewLabelName] = useState("");
  const [selectedColor, setSelectedColor] = useState<string>(DEFAULT_COLOR);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  const handleCreateLabel = () => {
    if (newLabelName.trim()) {
      onCreateLabel({ name: newLabelName.trim(), color: selectedColor });
      setNewLabelName("");
      setSelectedColor(DEFAULT_COLOR);
    }
  };

  const handleDeleteLabel = (id: string) => {
    onDeleteLabel(id);
    setConfirmDelete(null);
  };

  return (
    <Dialog open={open} onClose={onClose} className="max-w-md">
      <DialogHeader onClose={onClose}>Manage Labels</DialogHeader>
      <DialogBody className="space-y-6">
        {/* Create New Label */}
        <div className="space-y-3">
          <label className="text-sm font-medium text-white">Create New Label</label>
          <div className="flex gap-2">
            <Input
              value={newLabelName}
              onChange={(e) => setNewLabelName(e.target.value)}
              placeholder="Label name..."
              className="flex-1"
              onKeyDown={(e) => {
                if (e.key === "Enter") handleCreateLabel();
              }}
              data-testid="new-label-input"
            />
            <Button
              onClick={handleCreateLabel}
              disabled={!newLabelName.trim()}
              size="icon"
              data-testid="create-label-button"
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>

          {/* Color Picker */}
          <div className="flex items-center gap-2">
            <span className="text-xs text-slate-500">Color:</span>
            <div className="flex gap-1.5 flex-wrap">
              {PRESET_COLORS.map((color) => (
                <button
                  key={color}
                  onClick={() => setSelectedColor(color)}
                  className={`w-6 h-6 rounded-full transition-all ${
                    selectedColor === color
                      ? "ring-2 ring-white ring-offset-2 ring-offset-slate-900"
                      : "hover:scale-110"
                  }`}
                  style={{ backgroundColor: color }}
                  data-testid={`color-${color}`}
                />
              ))}
            </div>
          </div>

          {/* Preview */}
          {newLabelName.trim() && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-slate-500">Preview:</span>
              <Badge color={selectedColor}>{newLabelName}</Badge>
            </div>
          )}
        </div>

        {/* Existing Labels */}
        <div className="space-y-3">
          <label className="text-sm font-medium text-white">
            Your Labels ({labels.length})
          </label>
          {labels.length === 0 ? (
            <p className="text-sm text-slate-500 py-4 text-center">
              No labels yet. Create one above to organize your chats.
            </p>
          ) : (
            <div className="space-y-2 max-h-60 overflow-y-auto">
              {labels.map((label) => (
                <div
                  key={label.id}
                  className="flex items-center justify-between p-2 rounded-lg bg-white/5 border border-white/10"
                  data-testid={`label-item-${label.id}`}
                >
                  <div className="flex items-center gap-2">
                    <span
                      className="w-3 h-3 rounded-full"
                      style={{ backgroundColor: label.color }}
                    />
                    <span className="text-sm text-white">{label.name}</span>
                  </div>

                  {confirmDelete === label.id ? (
                    <div className="flex items-center gap-1">
                      <span className="text-xs text-slate-500 mr-2">Delete?</span>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setConfirmDelete(null)}
                        className="text-xs h-7 px-2"
                      >
                        No
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDeleteLabel(label.id)}
                        className="text-xs h-7 px-2"
                        data-testid={`confirm-delete-label-${label.id}`}
                      >
                        Yes
                      </Button>
                    </div>
                  ) : (
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => setConfirmDelete(label.id)}
                      data-testid={`delete-label-${label.id}`}
                    >
                      <Trash2 className="h-3.5 w-3.5 text-slate-500 hover:text-red-400" />
                    </Button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="ghost" onClick={onClose}>
          Close
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
