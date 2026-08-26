/**
 * @libraryId react-component-library:OverlayCanvas
 * @displayName OverlayCanvas
 * @description A measured subject overlay that names implicated bounds.
 * @version 1.0.11
 * @tags ["visualization","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource react-component-library:OverlayCanvas */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { Stack } from "@vrooli/react-component-library/Stack/1.0.0";
import { Text } from "@vrooli/react-component-library/Text/1.0.0";
import {
  CONTROL_VARIANTS,
  CONTROL_SIZES,
  SURFACE_ELEVATIONS,
} from "@vrooli/react-component-library/VisualRecipes/1.0.0";
export interface OverlaySubject {
  id: string;
  label: string;
  x: number;
  y: number;
  width: number;
  height: number;
}
export const OverlayCanvas = withClassName(function OverlayCanvas({
  subjects = [],
  message = "Select a claim to inspect its subjects.",
}: {
  subjects?: OverlaySubject[];
  message?: string;
}) {
  const strings = useStrings();
  return (
    <section
      className={SURFACE_ELEVATIONS.raised}
      aria-label={strings("visualization.overlay-canvas.claim-overlay", "Claim overlay")}
      data-rcl-asset="visualization.overlay-canvas"
      data-rcl-version="1.0.7"
      data-rcl-stamp="source"
      data-testid="visualization-overlay-canvas"
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-xs)" }}
    >
      <Stack gap="xs">
        <Text as="strong" textStyle="label">
          {strings("visualization.overlay-canvas.measured-subjects", "Measured subjects")}
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
        <button
          data-testid="visualization.overlay-canvas"
          type="button"
          className={`${CONTROL_VARIANTS.ghost} ${CONTROL_SIZES.md}`}
          data-bespoke="evidence disclosure remains a native action"
          style={{ minHeight: "var(--control-height)" }}
        >
          {strings("visualization.overlay-canvas.open-evidence", "Open evidence")}
        </button>
      </Stack>
    </section>
  );
});
