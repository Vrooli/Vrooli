/** @vrooliComponentSource react-component-library:QuarantineBadge */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import {
  CONTROL_VARIANTS,
  SURFACE_ELEVATIONS,
} from "@vrooli/react-component-library/VisualRecipes/1.0.0";
export const QuarantineBadge = withClassName(function QuarantineBadge({
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
      data-rcl-asset="feedback.quarantine-badge"
      data-rcl-version="1.0.3"
      data-rcl-stamp="source"
      data-testid="feedback-quarantine-badge"
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-2xs)" }}
    >
      Quarantined · {gate} · {reason}
    </section>
  );
});
