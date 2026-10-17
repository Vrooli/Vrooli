import type { ReactNode } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '../../../../shared/ui/dialog';

interface HelpModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

export function HelpModal({ open, onClose, title, children }: HelpModalProps) {
  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) onClose(); }}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className="sr-only">
            Reference guidance for configuring download artifact storage.
          </DialogDescription>
        </DialogHeader>
        <div className="prose prose-invert prose-sm max-w-none">
          {children}
        </div>
      </DialogContent>
    </Dialog>
  );
}
