import { useState } from "react";
import { Link } from "react-router-dom";
import { Boxes } from "lucide-react";
import { useTranslation } from "../../i18n";
import type { CatalogAsset, CoverageRow } from "../data";
import {
  assetAdoptions,
  assetHasDraft,
  assetVersion,
  formatNumber,
  formatRelative,
  timestampToDate,
} from "../data";
import { Chip, Dot } from "../primitives";

function previewURL(asset: CatalogAsset): string {
  const version = asset.latestVersion || asset.version || asset.draftVersion || "";
  return `/preview/${encodeURIComponent(asset.libraryId)}/preview.png?version=${encodeURIComponent(version)}`;
}

function assetStatus(
  asset: CatalogAsset,
  row: CoverageRow | undefined,
): { label: string; tone: "green" | "amber" | "red" | "ice" } {
  if (assetHasDraft(asset)) return { label: "Draft", tone: "amber" };
  if (!row) return { label: "Unmeasured", tone: "ice" };
  if (row.failedGates.length > 0)
    return { label: "Needs review", tone: row.blocksDownstream > 0 ? "red" : "amber" };
  if (row.achieved) return { label: "Verified", tone: "green" };
  return { label: "Indexed", tone: "ice" };
}

export function AssetCard({ asset, row }: { asset: CatalogAsset; row?: CoverageRow }) {
  const { t } = useTranslation();
  const [preview, setPreview] = useState<"loading" | "ready" | "failed">("loading");
  const status = assetStatus(asset, row);
  return (
    <Link to={`/assets/${encodeURIComponent(asset.id)}`} className="wb-asset-card">
      <span className="wb-asset-thumb">
        {preview === "ready" ? null : (
          <span className="wb-asset-thumb-state">
            {preview === "failed"
              ? t("workbench.components.previewPending", { defaultValue: "Preview pending" })
              : null}
          </span>
        )}
        <img
          src={previewURL(asset)}
          alt=""
          loading="lazy"
          data-state={preview}
          onLoad={() => setPreview("ready")}
          onError={() => setPreview("failed")}
        />
        <span className="wb-asset-thumb-corner">
          {assetHasDraft(asset)
            ? t("workbench.components.draft", { defaultValue: "draft" })
            : asset.catalogKind || t("workbench.components.asset", { defaultValue: "asset" })}
        </span>
      </span>
      <span className="wb-asset-body">
        <b>{asset.displayName}</b>
        <small>
          {asset.catalogId || asset.libraryId} · {assetVersion(asset)}
        </small>
      </span>
      <span className="wb-asset-foot">
        <Chip
          tone={status.tone}
          title={row?.failedGates.length ? `${row.failedGates.length} failed gates` : undefined}
        >
          <Dot tone={status.tone} />
          {status.label}
        </Chip>
        <span>{formatRelative(timestampToDate(asset.updatedAt))}</span>
      </span>
    </Link>
  );
}

export function AssetListRow({
  asset,
  row,
  tree,
}: {
  asset: CatalogAsset;
  row?: CoverageRow;
  tree?: boolean;
}) {
  const status = assetStatus(asset, row);
  return (
    <Link
      to={`/assets/${encodeURIComponent(asset.id)}`}
      className={tree ? "wb-tree-row" : "wb-list-row"}
    >
      <Boxes aria-hidden size={14} style={{ color: "var(--wb-muted-2)", flex: "none" }} />
      <span style={{ minWidth: 0, flex: 1 }}>
        <b>{asset.displayName}</b>
        <small style={{ display: "block" }}>
          {asset.catalogId || asset.libraryId} · {assetVersion(asset)}
        </small>
      </span>
      <Chip tone={status.tone}>{status.label}</Chip>
      <span className="wb-mono wb-small wb-muted-2">{formatNumber(assetAdoptions(asset))}</span>
    </Link>
  );
}
