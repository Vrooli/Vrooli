import { useTranslation } from "react-i18next";
import { HEADER_COLORS } from "../../consts/config";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";

interface HeaderColorPickerProps {
  currentColor: string;
  onSelectColor: (color: string) => void;
  testIdPrefix?: string;
}

export default function HeaderColorPicker({
  currentColor,
  onSelectColor,
  testIdPrefix = "appearance",
}: HeaderColorPickerProps) {
  const { t } = useTranslation();
  return (
    <section>
      <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
        {t(strings.appearance.headerColorHeading)}
      </h3>
      <div className="flex flex-wrap gap-1.5">
        {/* Transparent option */}
        <button
          type="button"
          data-testid={`${testIdPrefix}-header-color-transparent`}
          className={cn(
            "h-6 w-6 rounded-full border",
            currentColor === "transparent" ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
          )}
          style={{ background: "rgb(var(--wc-surface-input))" }}
          onClick={() => onSelectColor("transparent")}
          title={t(strings.appearance.noColorTitle)}
        />
        {HEADER_COLORS.map((color) => (
          <button
            key={color}
            type="button"
            data-testid={`${testIdPrefix}-header-color-${color}`}
            className={cn(
              "h-6 w-6 rounded-full border",
              currentColor === color ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
            )}
            style={{ backgroundColor: color }}
            onClick={() => onSelectColor(color)}
            title={color}
          />
        ))}
      </div>
    </section>
  );
}
