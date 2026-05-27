// useSurfaceBaselineModal (Plan C Phase 3) — the shared "Capture baseline"
// affordance for a surface tab. Opens SetBaselineModal pre-scoped to one
// surface (Decision 2) and, on success, marks the new baseline as the
// per-scenario default (Decision 4) so the surface's compare bar targets it
// immediately. Centralizes the modal state + preselect + default-on-create
// wiring so each tab doesn't re-implement it.

import { useState } from "react";
import { useDefaultBaseline } from "../../lib/hooks-baselines";
import { SetBaselineModal } from "./SetBaselineModal";
import type { BaselineSurface } from "./model";

export function useSurfaceBaselineModal(
  scenario: string,
  surface: BaselineSurface,
  repoId?: string | null,
) {
  const [open, setOpen] = useState(false);
  const { setDefaultBaseline } = useDefaultBaseline(scenario);

  const baselineModal = (
    <SetBaselineModal
      isOpen={open}
      scenario={scenario}
      repoId={repoId}
      preselectedSurfaces={[surface]}
      onClose={() => setOpen(false)}
      onCreated={(name) => setDefaultBaseline(name)}
    />
  );

  return { openCaptureBaseline: () => setOpen(true), baselineModal };
}
