import { useState } from "react";
import { Plus, Trash2, File, Link2, StickyNote, Tag } from "lucide-react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import { Badge } from "./ui/badge";
import type { ContextAttachmentData } from "../types";

interface ContextAttachmentEditorProps {
  attachments: ContextAttachmentData[];
  onChange: (attachments: ContextAttachmentData[]) => void;
  disabled?: boolean;
}

const ATTACHMENT_TYPES = [
  { value: "file", label: "File", icon: File },
  { value: "link", label: "Link", icon: Link2 },
  { value: "note", label: "Note", icon: StickyNote },
] as const;

export function ContextAttachmentEditor({
  attachments,
  onChange,
  disabled = false,
}: ContextAttachmentEditorProps) {
  const [newTagInput, setNewTagInput] = useState<Record<number, string>>({});

  const addAttachment = (type: string) => {
    const newAttachment: ContextAttachmentData = {
      type,
      key: "",
      tags: [],
      path: "",
      url: "",
      content: "",
      label: "",
    };
    onChange([...attachments, newAttachment]);
  };

  const updateAttachment = (index: number, updates: Partial<ContextAttachmentData>) => {
    const updated = [...attachments];
    updated[index] = { ...updated[index], ...updates };
    onChange(updated);
  };

  const removeAttachment = (index: number) => {
    onChange(attachments.filter((_, i) => i !== index));
  };

  const addTag = (index: number, tag: string) => {
    const trimmed = tag.trim().toLowerCase();
    if (!trimmed) return;

    const current = attachments[index];
    if (current.tags?.includes(trimmed)) return;

    updateAttachment(index, {
      tags: [...(current.tags || []), trimmed],
    });
    setNewTagInput({ ...newTagInput, [index]: "" });
  };

  const removeTag = (attachmentIndex: number, tagIndex: number) => {
    const current = attachments[attachmentIndex];
    const newTags = current.tags?.filter((_, i) => i !== tagIndex) || [];
    updateAttachment(attachmentIndex, { tags: newTags });
  };

  const getIcon = (type: string) => {
    const found = ATTACHMENT_TYPES.find((t) => t.value === type);
    return found ? <found.icon className="h-4 w-4" /> : null;
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Label>Context Attachments</Label>
        <div className="flex gap-2">
          {ATTACHMENT_TYPES.map((type) => (
            <Button
              key={type.value}
              type="button"
              variant="outline"
              size="sm"
              onClick={() => addAttachment(type.value)}
              disabled={disabled}
              className="gap-1"
            >
              <type.icon className="h-4 w-4" />
              {type.label}
            </Button>
          ))}
        </div>
      </div>

      {attachments.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No context attachments. Add files, links, or notes to provide additional context to the agent.
        </p>
      ) : (
        <div className="space-y-4">
          {attachments.map((attachment, index) => (
            <div
              key={index}
              className="border rounded-lg p-4 space-y-3 bg-muted/30"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {getIcon(attachment.type)}
                  <span className="text-sm font-medium capitalize">
                    {attachment.type}
                  </span>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => removeAttachment(index)}
                  disabled={disabled}
                  className="text-destructive hover:text-destructive"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>

              <div className="grid gap-3 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor={`key-${index}`}>Key (optional)</Label>
                  <Input
                    id={`key-${index}`}
                    value={attachment.key || ""}
                    onChange={(e) => updateAttachment(index, { key: e.target.value })}
                    placeholder="e.g., error-logs"
                    disabled={disabled}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor={`label-${index}`}>Label (optional)</Label>
                  <Input
                    id={`label-${index}`}
                    value={attachment.label || ""}
                    onChange={(e) => updateAttachment(index, { label: e.target.value })}
                    placeholder="Human-readable label"
                    disabled={disabled}
                  />
                </div>
              </div>

              {/* Type-specific fields */}
              {attachment.type === "file" && (
                <div className="space-y-2">
                  <Label htmlFor={`path-${index}`}>File Path *</Label>
                  <Input
                    id={`path-${index}`}
                    value={attachment.path || ""}
                    onChange={(e) => updateAttachment(index, { path: e.target.value })}
                    placeholder="/path/to/file or relative/path"
                    disabled={disabled}
                  />
                </div>
              )}

              {attachment.type === "link" && (
                <div className="space-y-2">
                  <Label htmlFor={`url-${index}`}>URL *</Label>
                  <Input
                    id={`url-${index}`}
                    value={attachment.url || ""}
                    onChange={(e) => updateAttachment(index, { url: e.target.value })}
                    placeholder="https://..."
                    disabled={disabled}
                  />
                </div>
              )}

              {(attachment.type === "note" || attachment.type === "link") && (
                <div className="space-y-2">
                  <Label htmlFor={`content-${index}`}>
                    {attachment.type === "note" ? "Content *" : "Description"}
                  </Label>
                  <Textarea
                    id={`content-${index}`}
                    value={attachment.content || ""}
                    onChange={(e) => updateAttachment(index, { content: e.target.value })}
                    placeholder={
                      attachment.type === "note"
                        ? "Inline content..."
                        : "Optional description of the link..."
                    }
                    rows={3}
                    disabled={disabled}
                  />
                </div>
              )}

              {/* Tags */}
              <div className="space-y-2">
                <Label>Tags</Label>
                <div className="flex flex-wrap gap-2">
                  {attachment.tags?.map((tag, tagIndex) => (
                    <Badge
                      key={tagIndex}
                      variant="secondary"
                      className="gap-1 cursor-pointer hover:bg-destructive/20"
                      onClick={() => !disabled && removeTag(index, tagIndex)}
                    >
                      <Tag className="h-3 w-3" />
                      {tag}
                    </Badge>
                  ))}
                  <div className="flex items-center gap-1">
                    <Input
                      value={newTagInput[index] || ""}
                      onChange={(e) =>
                        setNewTagInput({ ...newTagInput, [index]: e.target.value })
                      }
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          addTag(index, newTagInput[index] || "");
                        }
                      }}
                      placeholder="Add tag..."
                      className="h-7 w-24 text-xs"
                      disabled={disabled || (attachment.tags?.length || 0) >= 10}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => addTag(index, newTagInput[index] || "")}
                      disabled={disabled || (attachment.tags?.length || 0) >= 10}
                      className="h-7 w-7 p-0"
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
