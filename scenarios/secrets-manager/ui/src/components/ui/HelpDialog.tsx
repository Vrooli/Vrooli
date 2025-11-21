import { useState } from "react";
import { HelpCircle, X } from "lucide-react";
import { Button } from "./button";

interface HelpDialogProps {
  title: string;
  children: React.ReactNode;
}

export const HelpDialog = ({ title, children }: HelpDialogProps) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setIsOpen(true)}
        className="rounded-full p-1 text-white/40 transition hover:bg-white/10 hover:text-white/70"
        aria-label="Help"
      >
        <HelpCircle className="h-4 w-4" />
      </button>

      {isOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
          onClick={() => setIsOpen(false)}
        >
          <div
            className="w-full max-w-lg rounded-3xl border border-white/10 bg-slate-950 p-6 shadow-2xl shadow-emerald-500/20"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <h3 className="text-xl font-semibold text-white">{title}</h3>
              <Button variant="ghost" size="sm" onClick={() => setIsOpen(false)}>
                <X className="h-5 w-5" />
              </Button>
            </div>
            <div className="mt-4 space-y-3 text-sm text-white/70">
              {children}
            </div>
          </div>
        </div>
      )}
    </>
  );
};
