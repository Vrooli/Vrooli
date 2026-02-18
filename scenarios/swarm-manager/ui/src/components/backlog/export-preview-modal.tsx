/**
 * Export Preview Modal
 *
 * Shows a preview of exported markdown content with options to copy to clipboard
 * or download as a file. Fetches the export when opened.
 */

import { useState, useEffect } from "react";
import { useMutation } from "@tanstack/react-query";
import { Copy, Check, Download } from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { backlogService } from "../../services";

export interface ExportPreviewModalProps {
  isOpen: boolean;
  onClose: () => void;
  params: Parameters<typeof backlogService.exportItems>[0];
}

export function ExportPreviewModal({ isOpen, onClose, params }: ExportPreviewModalProps) {
  const [copied, setCopied] = useState(false);
  const [content, setContent] = useState<string | null>(null);

  const exportMutation = useMutation({
    mutationFn: async () => {
      const blob = await backlogService.exportItems(params);
      return blob.text();
    },
    onSuccess: (text) => {
      setContent(text);
    },
  });

  useEffect(() => {
    if (isOpen) {
      setContent(null);
      setCopied(false);
      exportMutation.mutate();
    }
    // Only trigger on open/close transitions
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  const handleCopy = async () => {
    if (!content) return;
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(content);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
        return;
      } catch {
        // Fall through
      }
    }
  };

  const handleDownload = () => {
    if (!content) return;
    const blob = new Blob([content], { type: "text/markdown" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `backlog-export-${new Date().toISOString().slice(0, 10)}.md`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="Export Preview"
      maxWidth="max-w-3xl"
      isLoading={exportMutation.isPending}
      testId="export-preview-modal"
    >
      {exportMutation.isPending && (
        <div className="flex items-center justify-center py-12">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-slate-500 border-t-cyan-400" />
          <span className="ml-3 text-sm text-slate-400">Generating export...</span>
        </div>
      )}

      {exportMutation.isError && (
        <div className="rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          Failed to generate export. Please try again.
        </div>
      )}

      {content !== null && (
        <>
          <pre className="max-h-[50vh] overflow-auto rounded-lg border border-slate-700/70 bg-slate-800/60 p-4 text-sm text-slate-300 font-mono whitespace-pre-wrap break-words">
            {content}
          </pre>
          <div className="mt-4 flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={handleCopy}>
              {copied ? (
                <Check className="mr-2 h-4 w-4 text-green-400" />
              ) : (
                <Copy className="mr-2 h-4 w-4" />
              )}
              {copied ? "Copied" : "Copy to Clipboard"}
            </Button>
            <Button size="sm" onClick={handleDownload}>
              <Download className="mr-2 h-4 w-4" />
              Download
            </Button>
          </div>
        </>
      )}
    </Dialog>
  );
}
