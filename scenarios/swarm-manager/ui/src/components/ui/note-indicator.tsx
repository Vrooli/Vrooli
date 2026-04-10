import { StickyNote } from "lucide-react";

interface NoteIndicatorProps {
  note?: string;
  className?: string;
}

export function NoteIndicator({ note, className }: NoteIndicatorProps) {
  if (!note) return null;

  const preview = note.length > 80 ? note.slice(0, 80) + "…" : note;

  return (
    <span title={preview} className={className}>
      <StickyNote className="h-3.5 w-3.5 text-amber-400/70" />
    </span>
  );
}
