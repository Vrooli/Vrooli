/** @vrooliComponentSource react-component-library:OverlayCanvas */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Text } from "../../../../primitives/Text/versions/1.0.0/Text";
import {
  CONTROL_VARIANTS,
  CONTROL_SIZES,
  SURFACE_ELEVATIONS,
} from "../../../../foundations/VisualRecipes/versions/1.0.0/VisualRecipes";
export interface OverlaySubject {
  id: string;
  label: string;
  x: number;
  y: number;
  width: number;
  height: number;
}
export function OverlayCanvas({
  subjects = [],
  message = "Select a claim to inspect its subjects.",
}: {
  subjects?: OverlaySubject[];
  message?: string;
}) {
  return (
    <section
      className={SURFACE_ELEVATIONS.raised}
      aria-label={translate("visualization.overlay-canvas.aria-label.1", "Claim overlay")}
      data-rcl-asset="visualization.overlay-canvas"
      data-rcl-version="1.0.7"
      data-rcl-stamp="source"
      data-testid="visualization-overlay-canvas"
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-xs)" }}
    >
      <Stack gap="xs">
        <Text as="strong" textStyle="label">
          {translate("visualization.overlay-canvas.text.2", "Measured subjects")}
        </Text>
        <div
          role="img"
          aria-label={message}
          data-bespoke="claim geometry layer exposes measured subject bounds"
          style={{
            position: "relative",
            minHeight: "8rem",
            background: "var(--color-surface-muted)",
          }}
        >
          {subjects.map((subject) => (
            <span
              key={subject.id}
              data-bespoke="measured subject marker preserves geometry coordinates"
              style={{
                position: "absolute",
                insetInlineStart: `${subject.x}px`,
                insetBlockStart: `${subject.y}px`,
                inlineSize: `${subject.width}px`,
                blockSize: `${subject.height}px`,
                border: "2px solid var(--color-danger)",
              }}
              aria-label={subject.label}
            />
          ))}
          {!subjects.length ? <Text tone="muted">{message}</Text> : null}
        </div>
        <button data-testid="visualization.overlay-canvas"
          type="button"
          className={`${CONTROL_VARIANTS.ghost} ${CONTROL_SIZES.md}`}
          data-bespoke="evidence disclosure remains a native action"
          style={{ minHeight: "var(--control-height)" }}
        >
          {translate("visualization.overlay-canvas.text.3", "Open evidence")}
        </button>
      </Stack>
    </section>
  );
}
