import { StepReady as ReadySurface } from "./StepReadiness";

/** Completion is a dedicated screen for the final readiness verdict. */
export function StepReady({ target = "local" }: { target?: string }) {
  return <ReadySurface target={target} />;
}
