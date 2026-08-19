/** @vrooliComponentSource react-component-library:EvidenceCarousel */
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Text } from "../../../../primitives/Text/versions/1.0.0/Text";
import {
  CONTROL_VARIANTS,
  SURFACE_ELEVATIONS,
} from "../../../../foundations/VisualRecipes/versions/1.0.0/VisualRecipes";
export interface EvidenceItem {
  id: string;
  kind: string;
  status: "available" | "missing" | "stale";
  reference?: string;
}
export function EvidenceCarousel({ items = [] }: { items?: EvidenceItem[] }) {
  return (
    <section
      className={SURFACE_ELEVATIONS.raised}
      aria-label="Evidence carousel"
      data-rcl-asset="visualization.evidence-carousel"
      data-rcl-version="1.0.2"
      data-rcl-stamp="source"
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-xs)" }}
    >
      <Stack gap="xs">
        <Text as="strong" textStyle="label">
          Evidence
        </Text>
        <div
          role="list"
          style={{ display: "flex", gap: "var(--space-xs)", overflowX: "auto" }}
        >
          {items.length ? (
            items.map((item) => (
              <button
                key={item.id}
                type="button"
                className={CONTROL_VARIANTS.secondary}
                data-status={item.status}
              >
                <Text as="span" textStyle="label">
                  {item.kind}
                </Text>
                <Text as="small" tone="muted">
                  {item.reference ?? item.status}
                </Text>
              </button>
            ))
          ) : (
            <Text tone="muted">No evidence captured.</Text>
          )}
        </div>
      </Stack>
    </section>
  );
}
