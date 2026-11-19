import { useRef, useState } from "react";
import { Crop, ImagePlus, Plus, Trash2, Upload } from "lucide-react";

import { useTranslation } from "../../i18n";
import { Empty, WbButton } from "../primitives";
import { designAssetUrl, listAllDesignAssets, type DesignAsset } from "../designs";
import { RegionCapture } from "./RegionCapture";
import type { ReferenceAsset } from "./useStudio";
import "./region-capture.css";

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function StudioAssets({
  designId,
  assets,
  onImport,
  onRemove,
  onPlace,
}: {
  designId: string;
  assets: ReferenceAsset[];
  onImport: (file: File, fidelity: "reference" | "captured") => void;
  onRemove: (id: string) => void;
  onPlace: (asset: DesignAsset) => void;
}) {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const [other, setOther] = useState<DesignAsset[] | null>(null);
  const [loadingOther, setLoadingOther] = useState(false);
  const [cropping, setCropping] = useState<DesignAsset | null>(null);

  const loadOther = () => {
    setLoadingOther(true);
    void listAllDesignAssets()
      .then((list) => setOther(list.filter((asset) => asset.designId !== designId)))
      .catch(() => setOther([]))
      .finally(() => setLoadingOther(false));
  };

  const fidelityLabel = (asset: DesignAsset): string =>
    asset.fidelity === "captured"
      ? t("workbench.studio.fidelityCaptured", { defaultValue: "captured" })
      : t("workbench.studio.reference", { defaultValue: "reference" });

  const fidelityTitle = (asset: DesignAsset): string =>
    asset.fidelity === "captured"
      ? t("workbench.studio.fidelityCapturedTitle", { defaultValue: "captured from this image" })
      : t("workbench.studio.fidelityReferenceTitle", { defaultValue: "uploaded as reference" });

  return (
    <>
      <div className="studio-pane-head">
        <span className="studio-pane-title">
          <ImagePlus aria-hidden size={13} />
          {t("workbench.studio.assets", { defaultValue: "Assets" })}
        </span>
        <span className="studio-frame-chip">
          {t("workbench.studio.assetCountAll", {
            defaultValue: "{{count}} asset",
            count: assets.length,
          })}
        </span>
      </div>

      <div className="studio-assets">
        <p className="studio-note">
          {t("workbench.studio.assetsNote", {
            defaultValue:
              "Imports are references for this design. They are not components and are never converted silently.",
          })}
        </p>

        <input
          ref={inputRef}
          type="file"
          accept="image/*"
          className="studio-file-input"
          aria-label={t("workbench.studio.importAsset", { defaultValue: "Import a reference image" })}
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) onImport(file, "reference");
            event.target.value = "";
          }}
        />

        <WbButton variant="ghost" size="sm" onClick={() => inputRef.current?.click()}>
          <Upload aria-hidden size={13} />
          {t("workbench.studio.import", { defaultValue: "Import reference" })}
        </WbButton>

        {assets.length === 0 ? (
          <Empty title={t("workbench.studio.assetsEmpty", { defaultValue: "No references yet" })}>
            {t("workbench.studio.assetsEmptyBody", {
              defaultValue:
                "Import a screenshot or image to keep it beside this design. It stays labelled reference fidelity.",
            })}
          </Empty>
        ) : (
          <ul className="studio-asset-grid">
            {assets.map((asset) => (
              <li key={asset.id} className="studio-asset">
                <img src={designAssetUrl(asset)} alt={asset.name} className="studio-asset-image" />
                <div className="studio-asset-meta">
                  <b title={asset.name}>{asset.name}</b>
                  <span>
                    <span
                      className="studio-asset-fidelity"
                      role="note"
                      title={fidelityTitle(asset)}
                      aria-label={fidelityTitle(asset)}
                      data-fidelity={asset.fidelity}
                      data-testid={`studio-asset-fidelity-${asset.id}`}
                    >
                      {fidelityLabel(asset)}
                    </span>
                    <small>{formatBytes(asset.bytes)}</small>
                  </span>
                  <button
                    type="button"
                    className="studio-link-btn studio-asset-crop"
                    aria-label={t("workbench.studio.cropRegionAsset", {
                      defaultValue: "Crop a region from {{name}}",
                      name: asset.name,
                    })}
                    onClick={() => setCropping(asset)}
                    data-testid={`studio-asset-crop-${asset.id}`}
                  >
                    <Crop aria-hidden size={11} />
                    {t("workbench.studio.cropRegionAction", { defaultValue: "Crop region" })}
                  </button>
                </div>
                <button
                  type="button"
                  className="studio-iconbtn studio-asset-place"
                  aria-label={t("workbench.studio.placeAsset", {
                    defaultValue: "Place {{name}} on the page",
                    name: asset.name,
                  })}
                  onClick={() => onPlace(asset)}
                >
                  <Plus aria-hidden size={12} />
                </button>
                <button
                  type="button"
                  className="studio-iconbtn is-danger studio-asset-remove"
                  aria-label={t("workbench.studio.removeAsset", {
                    defaultValue: "Remove {{name}}",
                    name: asset.name,
                  })}
                  onClick={() => onRemove(asset.id)}
                >
                  <Trash2 aria-hidden size={12} />
                </button>
              </li>
            ))}
          </ul>
        )}

        <div className="studio-asset-other">
          <div className="studio-asset-other-head">
            <span className="studio-pane-title">
              {t("workbench.studio.otherDesigns", { defaultValue: "Other designs" })}
            </span>
            <button type="button" className="studio-link-btn" onClick={loadOther} disabled={loadingOther}>
              {other === null
                ? t("workbench.studio.showOtherAssets", { defaultValue: "Show uploads" })
                : t("workbench.studio.refreshOtherAssets", { defaultValue: "Refresh" })}
            </button>
          </div>
          {other === null ? null : other.length === 0 ? (
            <p className="studio-note">
              {t("workbench.studio.otherAssetsEmpty", {
                defaultValue: "No uploads in the other designs yet.",
              })}
            </p>
          ) : (
            <ul className="studio-asset-grid">
              {other.map((asset) => (
                <li key={asset.id} className="studio-asset">
                  <img src={designAssetUrl(asset)} alt={asset.name} className="studio-asset-image" />
                  <div className="studio-asset-meta">
                    <b title={asset.name}>{asset.name}</b>
                    <span>
                      <small>{formatBytes(asset.bytes)}</small>
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {cropping ? (
        <RegionCapture
          asset={cropping}
          onClose={() => setCropping(null)}
          onImport={onImport}
        />
      ) : null}
    </>
  );
}
