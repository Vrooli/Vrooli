// useSurfaceBaselineModal (Plan C Phase 3) — the shared "Capture baseline"
// affordance reused by evidence tabs. Every invocation captures the same
// comprehensive baseline run and marks it as the scenario default.

import { useState } from "react";
import { useDefaultBaseline } from "../../lib/hooks-baselines";
import { SetBaselineModal } from "./SetBaselineModal";

export function useSurfaceBaselineModal(
  scenario: string,
  repoId?: string | null,
) {
  const [open, setOpen] = useState(false);
  const { setDefaultBaseline } = useDefaultBaseline(scenario);

  const baselineModal = (
    <SetBaselineModal
      isOpen={open}
      scenario={scenario}
      repoId={repoId}
      onClose={() => setOpen(false)}
      onCreated={(name) => setDefaultBaseline(name)}
    />
  );

  return { openCaptureBaseline: () => setOpen(true), baselineModal };
}
