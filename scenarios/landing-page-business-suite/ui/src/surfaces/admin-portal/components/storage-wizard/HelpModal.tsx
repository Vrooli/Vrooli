import type { ReactNode } from 'react';
import {
  Dialog,
  DialogContent,
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
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <div className="prose prose-invert prose-sm max-w-none">
          {children}
        </div>
      </DialogContent>
    </Dialog>
  );
}
