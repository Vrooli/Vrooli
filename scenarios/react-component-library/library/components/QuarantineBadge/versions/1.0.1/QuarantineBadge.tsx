/** @vrooliComponentSource react-component-library:QuarantineBadge */
import {
  CONTROL_VARIANTS,
  SURFACE_ELEVATIONS,
} from "../../../../foundations/VisualRecipes/versions/1.0.0/VisualRecipes";
export function QuarantineBadge({
  gate = "visual",
  reason = "calibration failed",
}: {
  gate?: string;
  reason?: string;
}) {
  return (
    <section
      role="status"
      className={`${SURFACE_ELEVATIONS.raised} ${CONTROL_VARIANTS.secondary}`}
      aria-label={`Gate ${gate} quarantined`}
      data-gate={gate}
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-2xs)" }}
    >
      Quarantined · {gate} · {reason}
    </section>
  );
}
