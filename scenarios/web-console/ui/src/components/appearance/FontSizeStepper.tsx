import { Plus, Minus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { FONT_SIZE_MIN, FONT_SIZE_MAX } from "../../lib/fontSizeUtils";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";

interface FontSizeStepperProps {
  currentSize: number;
  onChangeSize: (size: number) => void;
  testIdPrefix?: string;
}

export default function FontSizeStepper({
  currentSize,
  onChangeSize,
  testIdPrefix = "appearance",
}: FontSizeStepperProps) {
  const { t } = useTranslation();
  return (
    <section>
      <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
        {t(strings.appearance.fontSizeHeading)}
      </h3>
      <div className="flex items-center gap-1.5">
        <Button
          data-testid={`${testIdPrefix}-font-decrease`}
          variant="outline"
          size="icon"
          className="h-7 w-7"
          disabled={currentSize <= FONT_SIZE_MIN}
          onClick={() => onChangeSize(currentSize - 1)}
        >
          <Minus className="h-3 w-3" />
        </Button>
        <span
          data-testid={`${testIdPrefix}-font-value`}
          className="w-8 text-center text-sm font-mono text-wc-text-primary"
        >
          {currentSize}
        </span>
        <Button
          data-testid={`${testIdPrefix}-font-increase`}
          variant="outline"
          size="icon"
          className="h-7 w-7"
          disabled={currentSize >= FONT_SIZE_MAX}
          onClick={() => onChangeSize(currentSize + 1)}
        >
          <Plus className="h-3 w-3" />
        </Button>
      </div>
    </section>
  );
}
