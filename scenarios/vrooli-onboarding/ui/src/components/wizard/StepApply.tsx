import { ReadinessSurface } from "./StepReadiness";

/** The apply decision has its own route and evidence identity. */
export function StepApply({ target = "local" }: { target?: string }) {
  return <ReadinessSurface title={"Apply"} target={target} />;
}
