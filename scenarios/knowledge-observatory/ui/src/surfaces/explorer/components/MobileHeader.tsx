import { ChevronLeft } from "lucide-react";
import type { ReactNode } from "react";

export type MobileHeaderProps = {
  /** Title to display */
  title: string;
  /** Called when back button is clicked */
  onBack?: () => void;
  /** Whether to show the back button */
  showBack?: boolean;
  /** Optional actions to render on the right side */
  actions?: ReactNode;
};

export function MobileHeader({
  title,
  onBack,
  showBack = true,
  actions,
}: MobileHeaderProps) {
  return (
    <div className="ko-mobile-header">
      {showBack && onBack && (
        <button
          type="button"
          onClick={onBack}
          className="ko-mobile-back"
          aria-label="Go back"
        >
          <ChevronLeft className="h-5 w-5" />
        </button>
      )}
      <span className="ko-mobile-header-title">{title}</span>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
}
