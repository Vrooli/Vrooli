/**
 * Dialogs used by ChatHeader - Rename, Delete confirmation, and Export.
 *
 * Extracted from ChatHeader.tsx for modularity.
 */
import { useState } from "react";
import { FileText, FileJson, File } from "lucide-react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "../ui/dialog";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { exportChat, type ExportFormat } from "../../lib/api";

interface RenameDialogProps {
  open: boolean;
  onClose: () => void;
  chatName: string;
  onRename: (newName: string) => void;
}

export function RenameDialog({ open, onClose, chatName, onRename }: RenameDialogProps) {
  const [newName, setNewName] = useState(chatName);

  // Sync when opening
  if (open && newName !== chatName && !document.activeElement?.getAttribute("data-testid")?.includes("rename")) {
    // Will sync on next render
  }

  const handleRename = () => {
    if (newName.trim() && newName !== chatName) onRename(newName.trim());
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogHeader onClose={onClose}>Rename Chat</DialogHeader>
      <DialogBody>
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder="Enter chat name..."
          autoFocus
          onKeyDown={(e) => { if (e.key === "Enter") handleRename(); }}
          data-testid="rename-chat-input"
        />
      </DialogBody>
      <DialogFooter>
        <Button variant="ghost" onClick={onClose}>Cancel</Button>
        <Button onClick={handleRename} disabled={!newName.trim()}>Save</Button>
      </DialogFooter>
    </Dialog>
  );
}

interface DeleteDialogProps {
  open: boolean;
  onClose: () => void;
  chatName: string;
  onDelete: () => void;
  confirmTestId?: string;
}

export function DeleteDialog({ open, onClose, chatName, onDelete, confirmTestId }: DeleteDialogProps) {
  const handleDelete = () => { onDelete(); onClose(); };

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogHeader onClose={onClose}>Delete Chat</DialogHeader>
      <DialogBody>
        <p className="text-slate-300">
          Are you sure you want to delete <strong>"{chatName}"</strong>? This action cannot be undone.
        </p>
      </DialogBody>
      <DialogFooter>
        <Button variant="ghost" onClick={onClose}>Cancel</Button>
        <Button variant="destructive" onClick={handleDelete} data-testid={confirmTestId}>Delete</Button>
      </DialogFooter>
    </Dialog>
  );
}

interface ExportDialogProps {
  open: boolean;
  onClose: () => void;
  chatId: string;
  chatName: string;
}

export function ExportDialog({ open, onClose, chatId, chatName }: ExportDialogProps) {
  const [isExporting, setIsExporting] = useState(false);

  const handleExport = async (format: ExportFormat) => {
    try {
      setIsExporting(true);
      await exportChat(chatId, format);
      onClose();
    } catch (error) {
      console.error("Export failed:", error);
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogHeader onClose={onClose}>Export Chat</DialogHeader>
      <DialogBody>
        <p className="text-slate-400 text-sm mb-4">Choose a format to export "{chatName}"</p>
        <div className="space-y-2">
          <button onClick={() => { void handleExport("markdown"); }} disabled={isExporting} className="w-full flex items-center gap-3 p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors text-left" data-testid="export-markdown-button">
            <FileText className="h-5 w-5 text-indigo-400" />
            <div><div className="font-medium text-white">Markdown (.md)</div><div className="text-xs text-slate-500">Best for documentation and readability</div></div>
          </button>
          <button onClick={() => { void handleExport("json"); }} disabled={isExporting} className="w-full flex items-center gap-3 p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors text-left" data-testid="export-json-button">
            <FileJson className="h-5 w-5 text-emerald-400" />
            <div><div className="font-medium text-white">JSON (.json)</div><div className="text-xs text-slate-500">Complete data with all metadata</div></div>
          </button>
          <button onClick={() => { void handleExport("txt"); }} disabled={isExporting} className="w-full flex items-center gap-3 p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors text-left" data-testid="export-txt-button">
            <File className="h-5 w-5 text-slate-400" />
            <div><div className="font-medium text-white">Plain Text (.txt)</div><div className="text-xs text-slate-500">Simple format for any text editor</div></div>
          </button>
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="ghost" onClick={onClose}>Cancel</Button>
      </DialogFooter>
    </Dialog>
  );
}
