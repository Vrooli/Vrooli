/** @vrooliComponentSource services.layer-manager */

export interface LayerRecord {
  id: string;
  kind: string;
  dismiss?: () => void;
}

const layerRecords: LayerRecord[] = [];

export const layerManager = {
  push: (layer: LayerRecord) => {
    layerRecords.push(layer);
    return () => {
      const index = layerRecords.indexOf(layer);
      if (index >= 0) layerRecords.splice(index, 1);
    };
  },
  top: () => layerRecords.at(-1),
  dismissTop: () => layerRecords.at(-1)?.dismiss?.(),
  list: () => [...layerRecords],
};
