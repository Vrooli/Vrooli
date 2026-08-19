/** @vrooliComponentSource react-component-library:OverlayCanvas */
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Surface } from "../../../../primitives/Surface/versions/1.0.0/Surface";
import { Text } from "../../../../primitives/Text/versions/1.0.0/Text";
import {
  CONTROL_VARIANTS,
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
    <Surface
      elevation="raised"
      className={SURFACE_ELEVATIONS.raised}
      aria-label="Claim overlay"
    >
      <Stack gap="xs">
        <Text as="strong" textStyle="label">
          Measured subjects
        </Text>
        <div
          role="img"
          aria-label={message}
          style={{
            position: "relative",
            minHeight: "8rem",
            background: "var(--color-surface-muted)",
          }}
        >
          {subjects.map((subject) => (
            <span
              key={subject.id}
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
        <button type="button" className={CONTROL_VARIANTS.ghost}>
          Open evidence
        </button>
      </Stack>
    </Surface>
  );
}
