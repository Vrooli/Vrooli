import { X } from "lucide-react";
import { useEffect } from "react";
import { Button } from "../../../shared/ui/button";
import type { DocHealthViewModel } from "../../../shared/controllers/documentationController";
import { HealthPanel } from "./HealthPanel";

export type HealthModalProps = {
  isOpen: boolean;
  onClose: () => void;
  scenarioName: string | null;
  healthViewModel: DocHealthViewModel;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
  onRefresh: () => void;
};

export function HealthModal({
  isOpen,
  onClose,
  scenarioName,
  healthViewModel,
  isLoading,
  hasError,
  errorMessage,
  onRefresh,
}: HealthModalProps) {
  // Close on escape key
  useEffect(() => {
    if (!isOpen) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [isOpen, onClose]);

  // Prevent body scroll when modal is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div className="ko-modal-backdrop" onClick={onClose}>
      <div className="ko-modal" onClick={(e) => e.stopPropagation()}>
        <div className="ko-modal-header">
          <div>
            <p className="ko-text-sm ko-subtle">Documentation Health</p>
            <p className="text-lg font-semibold ko-text-strong">{scenarioName ?? "No scenario selected"}</p>
          </div>
          <Button variant="outline" size="sm" onClick={onClose} aria-label="Close modal">
            <X className="h-4 w-4" />
          </Button>
        </div>

        <div className="ko-modal-body">
          <HealthPanel
            scenarioName={scenarioName}
            healthViewModel={healthViewModel}
            isLoading={isLoading}
            hasError={hasError}
            errorMessage={errorMessage}
            onRefresh={onRefresh}
          />
        </div>
      </div>
    </div>
  );
}
