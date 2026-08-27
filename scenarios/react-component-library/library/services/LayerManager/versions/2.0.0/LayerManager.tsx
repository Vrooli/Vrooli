/**
 * @libraryId react-component-library:LayerManager
 * @displayName LayerManager
 * @description Scoped runtime service for predictable LayerManager behavior.
 * @version 2.0.0
 * @tags ["runtime","state"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource services.layer-manager */

export interface LayerRecord {
  id: string;
  kind: string;
  modal?: boolean;
  dismiss?: () => void;
}

export interface ManagedLayer extends LayerRecord {
  z: number;
}

const PORTAL_ID = "rcl-layer-root";
const layerRecords: ManagedLayer[] = [];
const isolatedSiblings = new Map<HTMLElement, { inert: boolean; ariaHidden: string | null }>();

export function getLayerPortalContainer(): HTMLElement | null {
  if (typeof document === "undefined") return null;
  let container = document.getElementById(PORTAL_ID);
  if (!container) {
    container = document.createElement("div");
    container.id = PORTAL_ID;
    container.dataset.rclLayerRoot = "";
    document.body.append(container);
  }
  return container;
}

function syncModalIsolation() {
  if (typeof document === "undefined") return;
  const container = getLayerPortalContainer();
  const hasModal = layerRecords.some((layer) => layer.modal);
  if (hasModal && container) {
    for (const sibling of Array.from(document.body.children)) {
      if (sibling === container || !(sibling instanceof HTMLElement)) continue;
      if (!isolatedSiblings.has(sibling)) {
        isolatedSiblings.set(sibling, {
          inert: sibling.inert,
          ariaHidden: sibling.getAttribute("aria-hidden"),
        });
      }
      sibling.inert = true;
      sibling.setAttribute("aria-hidden", "true");
    }
    return;
  }
  for (const [sibling, previous] of isolatedSiblings) {
    sibling.inert = previous.inert;
    if (previous.ariaHidden === null) sibling.removeAttribute("aria-hidden");
    else sibling.setAttribute("aria-hidden", previous.ariaHidden);
  }
  isolatedSiblings.clear();
}

function layerBase(kind: string) {
  if (kind === "alertdialog") return 700;
  if (kind === "menu" || kind === "popover") return 610;
  return 500;
}

export const layerManager = {
  push: (layer: LayerRecord) => {
    const managed = { ...layer, z: layerBase(layer.kind) + layerRecords.length };
    layerRecords.push(managed);
    syncModalIsolation();
    return () => {
      const index = layerRecords.indexOf(managed);
      if (index >= 0) layerRecords.splice(index, 1);
      syncModalIsolation();
    };
  },
  top: () => layerRecords.at(-1),
  dismissTop: () => layerRecords.at(-1)?.dismiss?.(),
  list: () => [...layerRecords],
  isTop: (id: string) => layerRecords.at(-1)?.id === id,
  portalContainer: getLayerPortalContainer,
  modalCount: () => layerRecords.filter((layer) => layer.modal).length,
};
