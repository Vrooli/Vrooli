/** @vrooliComponentSource react-component-library:QuarantineBadge */
import { Badge } from "../../../../primitives/Badge/versions/1.0.0/Badge";
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
    <Badge
      tone="warning"
      className={`${SURFACE_ELEVATIONS.raised} ${CONTROL_VARIANTS.secondary}`}
      aria-label={`Gate ${gate} quarantined`}
      data-gate={gate}
    >
      Quarantined · {gate} · {reason}
    </Badge>
  );
}
