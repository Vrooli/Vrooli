import type { ReactNode } from "react";
import { Plus } from "lucide-react";

/** "Add a note": a quiet disclosure that opens a note field, and stays open once there is a note. */
export function NoteDisclosure({ label, open, children }: { label: string; open: boolean; children: ReactNode }) {
  return (
    <details className="group text-sm" open={open}>
      <summary className="inline-flex min-h-11 cursor-pointer items-center gap-1.5 font-medium text-app-primary marker:content-none [&::-webkit-details-marker]:hidden">
        <Plus size={16} aria-hidden="true" className="transition-transform group-open:rotate-45" />
        {label}
      </summary>
      {children}
    </details>
  );
}
