/**
 * @libraryId react-component-library:BoundedMeter
 * @displayName BoundedMeter
 * @description A semantic bounded-value meter shared by score, measurement, and budget components.
 * @version 1.0.3
 * @tags ["primitive","token-bound","status","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource react-component-library:BoundedMeter */
import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { CSSProperties, HTMLAttributes, ReactNode } from "react";
import { Stack } from "@vrooli/react-component-library/Stack/1.0.0";
import { Text } from "@vrooli/react-component-library/Text/1.0.0";
import { SURFACE_ELEVATIONS } from "@vrooli/react-component-library/VisualRecipes/1.0.0";

export type BoundedMeterTone = "neutral" | "success" | "warning" | "danger";

export interface BoundedMeterProps
  extends Omit<HTMLAttributes<HTMLElement>, "aria-label" | "children"> {
  label?: string;
  value?: number;
  min?: number;
  max?: number;
  valueText?: ReactNode;
  description?: ReactNode;
  status?: string;
  tone?: BoundedMeterTone;
  ariaLabel?: string;
  meterLabel?: string;
  assetId?: string;
  assetVersion?: string;
  assetStamp?: string;
  testId?: string;
}

const styles = `
  [data-rcl-bounded-meter] {
    --rcl-meter-track: var(--color-border-subtle, color-mix(in srgb, var(--color-foreground, #0f172a) 14%, transparent));
    display: block;
    color: var(--color-foreground, #0f172a);
  }
  [data-rcl-bounded-meter-value] {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--space-xs, .5rem);
  }
  [data-rcl-bounded-meter-control] {
    inline-size: 100%;
    block-size: .5rem;
    accent-color: var(--rcl-meter-fill);
  }
  [data-rcl-bounded-meter-control]::-webkit-meter-bar {
    border: 0;
    border-radius: var(--radius-pill, 999px);
    background: var(--rcl-meter-track);
  }
  [data-rcl-bounded-meter-control]::-webkit-meter-optimum-value,
  [data-rcl-bounded-meter-control]::-webkit-meter-suboptimum-value,
  [data-rcl-bounded-meter-control]::-webkit-meter-even-less-good-value {
    border-radius: var(--radius-pill, 999px);
    background: var(--rcl-meter-fill);
  }
`;

const toneColors: Record<BoundedMeterTone, string> = {
  neutral: "var(--color-primary, #2563eb)",
  success: "var(--color-success, #15803d)",
  warning: "var(--color-warning, #b45309)",
  danger: "var(--color-danger, #dc2626)",
};

function finite(value: number | undefined, fallback: number) {
  return Number.isFinite(value) ? (value as number) : fallback;
}

export const BoundedMeter = withClassName(function BoundedMeter({
  label = resolveStrings("primitives.meter.value", "Value"),
  value = 0,
  min = 0,
  max = 100,
  valueText,
  description,
  status,
  tone = "neutral",
  ariaLabel,
  meterLabel,
  assetId = "primitives.meter",
  assetVersion = "1.0.1-draft.1",
  assetStamp = "source",
  testId = "primitives-meter",
  className,
  style,
  ...sectionProps
}: BoundedMeterProps) {
  const safeMin = finite(min, 0);
  const safeMax = Math.max(safeMin + 1, finite(max, 100));
  const safeValue = Math.max(safeMin, Math.min(safeMax, finite(value, safeMin)));
  const mergedStyle = {
    boxShadow: "var(--elev-raised)",
    padding: "var(--space-xs)",
    "--rcl-meter-fill": toneColors[tone],
    ...style,
  } as CSSProperties;

  return (
    <>
      <style data-rcl-bounded-meter-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <section
        {...sectionProps}
        className={`${SURFACE_ELEVATIONS.raised}${className ? ` ${className}` : ""}`}
        aria-label={ariaLabel ?? `${label} meter`}
        data-status={status}
        data-tone={tone}
        data-rcl-asset={assetId}
        data-rcl-version={assetVersion}
        data-rcl-stamp={assetStamp}
        data-testid={testId || "primitives.meter"}
        data-rcl-bounded-meter
        style={mergedStyle}
      >
        <Stack gap="2xs">
          <div data-rcl-bounded-meter-value>
            <Text as="strong" textStyle="label">
              {label}
            </Text>
            {valueText !== undefined && <Text numeric>{valueText}</Text>}
          </div>
          <meter
            min={safeMin}
            max={safeMax}
            value={safeValue}
            aria-label={meterLabel ?? `${label} value`}
            data-rcl-bounded-meter-control
          />
          {description !== undefined && (
            <Text tone="muted" numeric>
              {description}
            </Text>
          )}
        </Stack>
      </section>
    </>
  );
});
