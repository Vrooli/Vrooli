import { useViewportEnvironment } from "./useViewportEnvironment.ingest";

export function Default() {
  const viewport = useViewportEnvironment();
  return <output data-rcl-viewport-environment>{JSON.stringify(viewport)}</output>;
}
