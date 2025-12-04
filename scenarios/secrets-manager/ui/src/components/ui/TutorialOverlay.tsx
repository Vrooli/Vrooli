import { useEffect, useState } from "react";
import { X, GripVertical, ChevronDown } from "lucide-react";
import { Button } from "./button";

interface TutorialOverlayProps {
  title: string;
  subtitle?: string;
  stepLabel: string;
  content?: React.ReactNode;
  anchorId?: string;
  onClose: () => void;
  onNext?: () => void;
  onBack?: () => void;
  disableNext?: boolean;
  tutorials?: Array<{ id: string; label: string }>;
  activeTutorialId?: string | null;
  onSelectTutorial?: (id: string) => void;
}

export const TutorialOverlay = ({
  title,
  subtitle,
  stepLabel,
  content,
  anchorId,
  onClose,
  onNext,
  onBack,
  disableNext,
  tutorials = [],
  activeTutorialId,
  onSelectTutorial
}: TutorialOverlayProps) => {
  const [position, setPosition] = useState({ x: 40, y: 120 });
  const [dragging, setDragging] = useState(false);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [showTutorialMenu, setShowTutorialMenu] = useState(false);

  const startDrag = (event: React.MouseEvent) => {
    event.preventDefault();
    setDragging(true);
    setOffset({ x: event.clientX - position.x, y: event.clientY - position.y });
  };

  useEffect(() => {
    if (!dragging) return;
    const handleMove = (event: MouseEvent) => {
      setPosition({ x: event.clientX - offset.x, y: event.clientY - offset.y });
    };
    const stopDrag = () => setDragging(false);

    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", stopDrag);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", stopDrag);
    };
  }, [dragging, offset.x, offset.y]);

  // Scroll to anchor target when provided
  useEffect(() => {
    if (!anchorId) return;
    const target = document.getElementById(anchorId);
    if (target) {
      target.scrollIntoView({ behavior: "smooth", block: "center" });
      target.classList.add("tutorial-highlight");
      return () => {
        target.classList.remove("tutorial-highlight");
      };
    }
  }, [anchorId]);

  return (
    <div
      className="fixed z-50 w-[360px] rounded-2xl border border-emerald-400/40 bg-slate-900/95 shadow-2xl shadow-emerald-500/20 backdrop-blur"
      style={{ left: position.x, top: position.y }}
    >
      <div
        className="flex items-center justify-between border-b border-white/10 px-3 py-2 cursor-move"
        onMouseDown={startDrag}
      >
        <div className="flex items-center gap-2">
          <GripVertical className="h-4 w-4 text-emerald-300" />
          <div className="relative">
            <button
              type="button"
              className="flex items-center gap-2 rounded-md border border-transparent px-1 py-0.5 text-left transition hover:border-emerald-400/40"
              onClick={() => setShowTutorialMenu((open) => !open)}
              disabled={tutorials.length === 0}
            >
              <div>
                <p className="text-xs uppercase tracking-[0.3em] text-emerald-200">Tutorial</p>
                <p className="text-sm font-semibold text-white">{title}</p>
              </div>
              {tutorials.length > 0 ? <ChevronDown className="h-4 w-4 text-emerald-200" /> : null}
            </button>
            {showTutorialMenu && tutorials.length > 0 ? (
              <div className="absolute left-0 top-full z-10 mt-2 w-56 rounded-xl border border-white/10 bg-slate-900 shadow-lg">
                {tutorials.map((tutorial) => (
                  <button
                    key={tutorial.id}
                    type="button"
                    className={`block w-full px-3 py-2 text-left text-sm transition hover:bg-white/5 ${
                      tutorial.id === activeTutorialId ? "text-emerald-200" : "text-white/80"
                    }`}
                    onClick={() => {
                      setShowTutorialMenu(false);
                      onSelectTutorial?.(tutorial.id);
                    }}
                  >
                    {tutorial.label}
                  </button>
                ))}
              </div>
            ) : null}
          </div>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X className="h-4 w-4 text-white/70" />
        </Button>
      </div>

      <div className="space-y-3 px-4 py-3">
        {subtitle ? <p className="text-xs text-white/60">{subtitle}</p> : null}
        <div className="rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs text-white/70">
          {content}
        </div>
        <div className="flex items-center justify-between text-xs text-white/60">
          <span>{stepLabel}</span>
          <div className="space-x-2">
            {onBack ? (
              <Button variant="outline" size="sm" onClick={onBack}>
                Back
              </Button>
            ) : null}
            {onNext ? (
              <Button size="sm" onClick={onNext} disabled={disableNext}>
                Next
              </Button>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
};
