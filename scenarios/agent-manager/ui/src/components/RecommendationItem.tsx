import { useState } from "react";
import { Check, MessageSquare, Trash2, X } from "lucide-react";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import type { Recommendation } from "../types";

interface RecommendationItemProps {
  recommendation: Recommendation;
  onUpdate: (updated: Recommendation) => void;
  onDelete: () => void;
}

export function RecommendationItem({
  recommendation,
  onUpdate,
  onDelete,
}: RecommendationItemProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editText, setEditText] = useState(recommendation.text);
  const [showNoteInput, setShowNoteInput] = useState(!!recommendation.note);
  const [noteText, setNoteText] = useState(recommendation.note ?? "");

  const handleToggleSelected = () => {
    onUpdate({ ...recommendation, selected: !recommendation.selected });
  };

  const handleSaveEdit = () => {
    if (editText.trim()) {
      onUpdate({
        ...recommendation,
        text: editText.trim(),
        isEdited: editText.trim() !== recommendation.text,
      });
      setIsEditing(false);
    }
  };

  const handleCancelEdit = () => {
    setEditText(recommendation.text);
    setIsEditing(false);
  };

  const handleSaveNote = () => {
    onUpdate({ ...recommendation, note: noteText.trim() || undefined });
    if (!noteText.trim()) {
      setShowNoteInput(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent, action: () => void) => {
    if (e.key === "Enter") {
      e.preventDefault();
      action();
    } else if (e.key === "Escape") {
      if (isEditing) handleCancelEdit();
      else if (showNoteInput && !recommendation.note) {
        setShowNoteInput(false);
        setNoteText("");
      }
    }
  };

  return (
    <div
      className={`group flex flex-col gap-2 rounded-lg border p-3 transition-all ${
        recommendation.selected
          ? "border-border bg-background"
          : "border-border/50 bg-muted/30"
      }`}
    >
      <div className="flex items-start gap-3">
        <Checkbox
          checked={recommendation.selected}
          onCheckedChange={handleToggleSelected}
          className="mt-0.5"
        />

        <div className="flex-1 min-w-0">
          {isEditing ? (
            <div className="flex items-center gap-2">
              <Input
                value={editText}
                onChange={(e) => setEditText(e.target.value)}
                onKeyDown={(e) => handleKeyDown(e, handleSaveEdit)}
                className="flex-1"
                autoFocus
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={handleSaveEdit}
                className="h-8 w-8"
              >
                <Check className="h-4 w-4" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={handleCancelEdit}
                className="h-8 w-8"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => setIsEditing(true)}
              className={`text-left text-sm transition-colors hover:text-primary ${
                recommendation.selected
                  ? "text-foreground"
                  : "text-muted-foreground line-through"
              }`}
            >
              {recommendation.text}
              {recommendation.isEdited && (
                <span className="ml-1 text-xs text-muted-foreground">
                  (edited)
                </span>
              )}
              {recommendation.isNew && (
                <span className="ml-1 text-xs text-primary">(new)</span>
              )}
            </button>
          )}
        </div>

        <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={() => setShowNoteInput(!showNoteInput)}
            className={`h-7 w-7 ${recommendation.note ? "text-primary" : ""}`}
            title="Add note"
          >
            <MessageSquare className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={onDelete}
            className="h-7 w-7 text-muted-foreground hover:text-destructive"
            title="Delete"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {showNoteInput && (
        <div className="ml-7 flex items-center gap-2">
          <span className="text-xs text-muted-foreground">Note:</span>
          <Input
            value={noteText}
            onChange={(e) => setNoteText(e.target.value)}
            onBlur={handleSaveNote}
            onKeyDown={(e) => handleKeyDown(e, handleSaveNote)}
            placeholder="Add a note..."
            className="flex-1 h-8 text-sm"
          />
        </div>
      )}

      {recommendation.note && !showNoteInput && (
        <button
          type="button"
          onClick={() => setShowNoteInput(true)}
          className="ml-7 text-xs text-muted-foreground hover:text-foreground"
        >
          Note: {recommendation.note}
        </button>
      )}
    </div>
  );
}
