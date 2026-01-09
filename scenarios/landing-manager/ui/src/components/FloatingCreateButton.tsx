import { memo } from 'react';
import { Plus } from 'lucide-react';

interface FloatingCreateButtonProps {
  onClick: () => void;
}

/**
 * Floating action button (FAB) that appears in the bottom-right corner.
 * Provides quick access to the create scenario dialog from anywhere on the page.
 */
export const FloatingCreateButton = memo(function FloatingCreateButton({
  onClick,
}: FloatingCreateButtonProps) {
  return (
    <div className="fixed bottom-6 right-6 z-40">
      <button
        onClick={onClick}
        className="group w-14 h-14 rounded-full bg-gradient-to-r from-emerald-500 to-blue-500 text-white shadow-2xl shadow-emerald-500/40 hover:shadow-emerald-500/60 hover:scale-110 active:scale-95 transition-all duration-200 focus:outline-none focus:ring-4 focus:ring-emerald-500/50 flex items-center justify-center"
        aria-label="Create new landing page (⌘K)"
        title="Create new landing page"
      >
        <Plus className="h-7 w-7 group-hover:rotate-90 transition-transform duration-300" aria-hidden="true" />
      </button>
      {/* Pulse animation ring */}
      <span className="absolute inset-0 rounded-full bg-emerald-500/30 animate-ping pointer-events-none" style={{ animationDuration: '2s' }} />
    </div>
  );
});
