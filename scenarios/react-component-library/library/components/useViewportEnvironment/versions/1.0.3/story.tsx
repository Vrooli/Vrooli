import { useViewportEnvironment } from "../../../../support/useViewportEnvironment/1.0.3/useViewportEnvironment.ingest";

export function Default() {
  const viewport = useViewportEnvironment();
  return <output data-rcl-viewport-environment>{JSON.stringify(viewport)}</output>;
}
