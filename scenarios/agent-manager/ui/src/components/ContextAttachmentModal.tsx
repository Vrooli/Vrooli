import { useState } from "react";
import { Check, Copy, File, Link2, StickyNote, Tag } from "lucide-react";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { CodeBlock } from "./markdown/components/CodeBlock";
import type { ContextAttachmentData } from "../types";

interface ContextAttachmentModalProps {
  attachment: ContextAttachmentData | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ContextAttachmentModal({
  attachment,
  open,
  onOpenChange,
}: ContextAttachmentModalProps) {
  const [viewMode, setViewMode] = useState<"formatted" | "json">("formatted");
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied" | "error">("idle");

  if (!attachment) return null;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(attachment, null, 2));
      setCopyStatus("copied");
      setTimeout(() => setCopyStatus("idle"), 2000);
    } catch {
      setCopyStatus("error");
      setTimeout(() => setCopyStatus("idle"), 2000);
    }
  };

  const getTypeIcon = () => {
    switch (attachment.type) {
      case "file":
        return <File className="h-5 w-5 text-muted-foreground" />;
      case "link":
        return <Link2 className="h-5 w-5 text-muted-foreground" />;
      case "note":
        return <StickyNote className="h-5 w-5 text-muted-foreground" />;
      default:
        return <File className="h-5 w-5 text-muted-foreground" />;
    }
  };

  const getTitle = () => {
    if (attachment.label) return attachment.label;
    if (attachment.key) return attachment.key;
    return `${attachment.type.charAt(0).toUpperCase()}${attachment.type.slice(1)} Attachment`;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange} contentClassName="max-w-2xl">
      <DialogContent>
        <DialogHeader onClose={() => onOpenChange(false)}>
          <div className="flex items-center gap-3">
            {getTypeIcon()}
            <DialogTitle>{getTitle()}</DialogTitle>
          </div>
        </DialogHeader>

        <div className="flex gap-1 px-6 pt-4">
          <Button
            variant={viewMode === "formatted" ? "default" : "outline"}
            size="sm"
            onClick={() => setViewMode("formatted")}
          >
            Formatted
          </Button>
          <Button
            variant={viewMode === "json" ? "default" : "outline"}
            size="sm"
            onClick={() => setViewMode("json")}
          >
            Raw JSON
          </Button>
        </div>

        <DialogBody>
          {viewMode === "formatted" ? (
            <div className="space-y-4">
              <div className="grid gap-3">
                <div className="flex items-center gap-2">
                  <span className="text-sm text-muted-foreground w-16">Type</span>
                  <Badge variant="outline" className="capitalize">
                    {attachment.type}
                  </Badge>
                </div>

                {attachment.key && (
                  <div className="flex items-start gap-2">
                    <span className="text-sm text-muted-foreground w-16">Key</span>
                    <code className="text-sm bg-muted px-2 py-1 rounded break-all">
                      {attachment.key}
                    </code>
                  </div>
                )}

                {attachment.label && (
                  <div className="flex items-start gap-2">
                    <span className="text-sm text-muted-foreground w-16">Label</span>
                    <span className="text-sm font-medium">{attachment.label}</span>
                  </div>
                )}

                {attachment.path && (
                  <div className="flex items-start gap-2">
                    <span className="text-sm text-muted-foreground w-16">Path</span>
                    <code className="text-sm bg-muted px-2 py-1 rounded break-all">
                      {attachment.path}
                    </code>
                  </div>
                )}

                {attachment.url && (
                  <div className="flex items-start gap-2">
                    <span className="text-sm text-muted-foreground w-16">URL</span>
                    <a
                      href={attachment.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-sm text-primary hover:underline break-all"
                    >
                      {attachment.url}
                    </a>
                  </div>
                )}

                {attachment.tags && attachment.tags.length > 0 && (
                  <div className="flex items-start gap-2">
                    <span className="text-sm text-muted-foreground w-16">Tags</span>
                    <div className="flex flex-wrap gap-1">
                      {attachment.tags.map((tag, i) => (
                        <Badge key={`${tag}-${i}`} variant="outline" className="text-xs gap-1">
                          <Tag className="h-3 w-3" />
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {attachment.content && (
                <div className="space-y-2">
                  <span className="text-sm text-muted-foreground">Content</span>
                  <CodeBlock code={attachment.content} />
                </div>
              )}
            </div>
          ) : (
            <CodeBlock code={JSON.stringify(attachment, null, 2)} language="json" />
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" size="sm" onClick={handleCopy} className="gap-2">
            {copyStatus === "copied" ? (
              <>
                <Check className="h-4 w-4 text-success" />
                Copied
              </>
            ) : copyStatus === "error" ? (
              <>
                <Copy className="h-4 w-4" />
                Copy failed
              </>
            ) : (
              <>
                <Copy className="h-4 w-4" />
                Copy JSON
              </>
            )}
          </Button>
          <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
