import { useTranslation } from "react-i18next";
import { NumberField } from "@vrooli/react-component-library/NumberField";
import { FONT_SIZE_MIN, FONT_SIZE_MAX, FONT_SIZE_STEP } from "../../lib/fontSizeUtils";
import { strings } from "../../consts/strings";

interface FontSizeStepperProps {
  currentSize: number;
  onChangeSize: (size: number) => void;
  testIdPrefix?: string;
}

/**
 * The clamp, the draft-then-commit editing and the bound-aware steppers now
 * live in `NumberField`, which was extracted from this component precisely
 * because two other numeric settings had each re-implemented them and
 * disagreed. What remains here is the heading, the sample glyph, and the
 * bounds this particular setting declares.
 */
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
      <div className="flex items-center gap-2">
        <NumberField
          testId={`${testIdPrefix}-font`}
          label={t(strings.appearance.fontSizeInputAriaLabel)}
          value={currentSize}
          onChange={onChangeSize}
          min={FONT_SIZE_MIN}
          max={FONT_SIZE_MAX}
          step={FONT_SIZE_STEP}
          unit={t(strings.appearance.fontSizeUnit)}
          decreaseLabel={t(strings.appearance.fontSizeDecreaseAriaLabel)}
          increaseLabel={t(strings.appearance.fontSizeIncreaseAriaLabel)}
          shape="pill"
          size="sm"
        />
        <span
          data-testid={`${testIdPrefix}-font-sample`}
          className="ms-auto font-mono text-wc-text-secondary"
          style={{ fontSize: `${currentSize}px`, lineHeight: 1 }}
          aria-hidden="true"
        >
          Aa
        </span>
      </div>
    </section>
  );
}
