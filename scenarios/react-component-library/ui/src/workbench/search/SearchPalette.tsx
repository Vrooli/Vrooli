import { X } from "lucide-react";

import { useTranslation } from "../../i18n";
import { useFocusTrap } from "../useFocusTrap";
import { SearchSurface } from "./SearchSurface";
import "./search-palette.css";

export interface SearchPaletteProps {
  open: boolean;
  onClose: () => void;
}

/**
 * The global command surface. It mounts the same SearchSurface as Start, so
 * ranking, availability and result kinds cannot drift between the two.
 */
export function SearchPalette({ open, onClose }: SearchPaletteProps) {
  const { t } = useTranslation();
  const layer = useFocusTrap<HTMLDivElement>(open, onClose, "input");

  if (!open) return null;

  return (
    <div
      ref={layer}
      className="wb-command-layer"
      role="dialog"
      aria-modal="true"
      aria-label={t("workbench.command.label", { defaultValue: "Workbench search" })}
    >
      <div className="wb-command-panel">
        <div className="wb-row-between" style={{ padding: "10px 12px 0" }}>
          <span className="wb-caption">
            {t("workbench.command.title", { defaultValue: "Search and commands" })}
          </span>
          <button
            type="button"
            className="wb-iconbtn"
            onClick={onClose}
            aria-label={t("workbench.command.close", { defaultValue: "Close search" })}
          >
            <X aria-hidden size={15} />
          </button>
        </div>
        <div style={{ padding: 12 }}>
          <SearchSurface compact onNavigate={onClose} />
        </div>
        <div className="wb-command-foot">
          <span>
            {t("workbench.command.foot", {
              defaultValue: "↑↓ to move · Enter to open · Esc to close",
            })}
          </span>
        </div>
      </div>
    </div>
  );
}
