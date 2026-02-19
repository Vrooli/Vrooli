import { RefreshCw } from "lucide-react";
import { Button } from "./button";

interface RefreshButtonProps {
  onClick: () => void;
  disabled?: boolean;
  "aria-label"?: string;
  "data-testid"?: string;
}

export function RefreshButton({ onClick, disabled, "aria-label": ariaLabel, "data-testid": testId }: RefreshButtonProps) {
  return (
    <Button variant="outline" size="sm" onClick={onClick} disabled={disabled} aria-label={ariaLabel} data-testid={testId}>
      <RefreshCw className={`h-4 w-4 ${disabled ? "animate-spin" : ""}`} aria-hidden="true" />
    </Button>
  );
}
