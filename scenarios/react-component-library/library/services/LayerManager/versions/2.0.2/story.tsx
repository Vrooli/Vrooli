import { layerManager } from "./LayerManager";

export function Default() {
  return (
    <output data-testid="services.layer-manager">{layerManager.list().length} active layers</output>
  );
}
