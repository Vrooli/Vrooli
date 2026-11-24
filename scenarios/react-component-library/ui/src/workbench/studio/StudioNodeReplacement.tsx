import { useMemo, useState } from "react";
import { ArrowRightLeft, Search } from "lucide-react";

import { useTranslation } from "../../i18n";
import { TextInput } from "../primitives";
import type { TemplateNode, TemplateNodeType } from "../templates/spec";
import { rclPrimitive } from "./rcl";
import type { CatalogAsset } from "../data";
import { rankCatalogCandidates, rankReplacementCandidates } from "./nodeReplacement";

export function StudioNodeReplacement({
  node,
  onReplace,
  catalogCandidates = [],
  catalogLoading = false,
  catalogFailed = false,
  pendingCatalogAssetId = "",
  onReplaceCatalogAsset,
}: {
  node: TemplateNode;
  onReplace: (type: TemplateNodeType) => void;
  catalogCandidates?: readonly CatalogAsset[];
  catalogLoading?: boolean;
  catalogFailed?: boolean;
  pendingCatalogAssetId?: string;
  onReplaceCatalogAsset?: (componentId: string) => void;
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const candidates = useMemo(() => rankReplacementCandidates(node, query), [node, query]);
  const catalog = useMemo(
    () => rankCatalogCandidates(node, catalogCandidates, query),
    [catalogCandidates, node, query],
  );
  const currentIsRcl = Boolean(rclPrimitive(node.type));

  return (
    <section className="studio-node-replacement" aria-labelledby="studio-node-replacement-title">
      <div className="studio-system-head">
        <span id="studio-node-replacement-title">
          {t("workbench.studio.replaceWithRcl", { defaultValue: "RCL building blocks" })}
        </span>
        <small>
          {currentIsRcl
            ? t("workbench.studio.rclCurrent", { defaultValue: "RCL layer" })
            : t("workbench.studio.customCurrent", { defaultValue: "Custom layer" })}
        </small>
      </div>
      <p className="studio-note">
        {t("workbench.studio.replaceWithRclNote", {
          defaultValue:
            "Choose a native primitive or layout pattern. Best matches appear first; search to browse the full set.",
        })}
      </p>
      {node.type === "page" ? (
        <p className="studio-note" role="status">
          {t("workbench.studio.frameRootNotReplaceable", {
            defaultValue:
              "The frame root stays a page. Select a layer inside it to replace that part.",
          })}
        </p>
      ) : null}
      <div className="studio-node-replacement-search">
        <Search aria-hidden size={13} />
        <TextInput
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-label={t("workbench.studio.searchRclCandidates", {
            defaultValue: "Search RCL building blocks",
          })}
          placeholder={t("workbench.studio.searchRclPlaceholder", {
            defaultValue: "Search buttons, cards, progress…",
          })}
        />
      </div>
      {node.type === "page" ? null : candidates.length === 0 && catalog.length === 0 ? (
        <p className="studio-note" role="status">
          {t("workbench.studio.noRclCandidates", {
            defaultValue: "No compatible building blocks match this search.",
          })}
        </p>
      ) : candidates.length > 0 ? (
        <ul className="studio-node-replacement-list">
          {candidates.map((candidate) => (
            <li key={candidate.type}>
              <button
                type="button"
                className="studio-node-replacement-option"
                aria-label={t("workbench.studio.replaceWithCandidate", {
                  defaultValue: "Replace with {{candidate}}",
                  candidate: candidate.name,
                })}
                aria-current={node.type === candidate.type ? "true" : undefined}
                disabled={node.type === candidate.type}
                onClick={() => onReplace(candidate.type)}
              >
                <span className="studio-node-replacement-icon" aria-hidden>
                  <ArrowRightLeft size={12} />
                </span>
                <span className="studio-node-replacement-copy">
                  <b>{candidate.name}</b>
                  <small>{candidate.description}</small>
                </span>
                <span className="studio-node-replacement-kind">
                  {candidate.kind === "primitive" ? "RCL" : "Pattern"}
                </span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      {node.type === "page" ? null : (
        <div className="studio-node-catalog-matches">
          <div className="studio-system-head">
            <span>
              {t("workbench.studio.catalogComponents", { defaultValue: "Catalog components" })}
            </span>
            <small>{catalog.length}</small>
          </div>
          {catalogFailed ? (
            <p className="studio-note" role="alert">
              {t("workbench.studio.catalogComponentsUnavailable", {
                defaultValue: "The component catalog is unavailable. Retry after it is healthy.",
              })}
            </p>
          ) : catalogLoading && catalogCandidates.length === 0 ? (
            <p className="studio-note" role="status">
              {t("workbench.studio.catalogComponentsLoading", {
                defaultValue: "Searching published components…",
              })}
            </p>
          ) : catalog.length === 0 ? (
            <p className="studio-note" role="status">
              {t("workbench.studio.catalogComponentsEmpty", {
                defaultValue: "No published components match this layer.",
              })}
            </p>
          ) : (
            <ul className="studio-node-replacement-list">
              {catalog.map((candidate) => (
                <li key={candidate.id}>
                  <button
                    type="button"
                    className="studio-node-replacement-option"
                    aria-label={t("workbench.studio.replaceWithCatalogComponent", {
                      defaultValue: "Replace with {{candidate}} component",
                      candidate: candidate.name,
                    })}
                    disabled={pendingCatalogAssetId !== ""}
                    onClick={() => onReplaceCatalogAsset?.(candidate.id)}
                  >
                    <span className="studio-node-replacement-icon" aria-hidden>
                      <ArrowRightLeft size={12} />
                    </span>
                    <span className="studio-node-replacement-copy">
                      <b>{candidate.name}</b>
                      <small>
                        {pendingCatalogAssetId === candidate.id
                          ? t("workbench.studio.catalogComponentLoading", {
                              defaultValue: "Loading component story…",
                            })
                          : candidate.description}
                      </small>
                    </span>
                    <span className="studio-node-replacement-kind">
                      {t("workbench.studio.catalogRclBadge", { defaultValue: "RCL" })}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  );
}
