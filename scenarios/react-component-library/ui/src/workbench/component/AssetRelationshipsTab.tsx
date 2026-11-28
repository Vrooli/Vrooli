import { useQuery } from "@tanstack/react-query";
import { CircleAlert, GitBranch } from "lucide-react";

import { getAssetRelationships } from "../../api/catalogGraph";
import { useTranslation } from "../../i18n";
import type { ComponentBundle } from "../data";
import { Banner, Empty, Skeleton } from "../primitives";

export function AssetRelationshipsTab({ bundle }: { bundle: ComponentBundle }) {
  const { t } = useTranslation();
  const relationships = useQuery({
    queryKey: ["wb", "relationships", bundle.asset.id],
    queryFn: () => getAssetRelationships(bundle.asset.catalogId || bundle.asset.id),
    staleTime: 60_000,
    retry: 0,
  });

  if (relationships.isLoading) return <Skeleton height={100} />;
  if (relationships.isError) {
    return (
      <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
        {t("workbench.component.relationshipsError", {
          defaultValue: "The relationship owner did not return this asset's graph.",
        })}
      </Banner>
    );
  }
  const data = relationships.data;
  if (!data) {
    return (
      <Empty
        title={t("workbench.component.noRelationships", {
          defaultValue: "No relationships recorded",
        })}
      >
        {t("workbench.component.noRelationshipsBody", {
          defaultValue:
            "This asset has no declared dependencies or consumers in the catalog graph.",
        })}
      </Empty>
    );
  }
  return (
    <div className="wb-drawer-grid">
      <div className="wb-stack tight">
        <span className="wb-caption">
          {t("workbench.component.dependsOn", { defaultValue: "Depends on" })}
        </span>
        {data.directDependencies.length === 0 ? (
          <span className="wb-muted wb-small">
            {t("workbench.component.none", { defaultValue: "None" })}
          </span>
        ) : (
          data.directDependencies.map((node) => (
            <div key={node.assetId} className="wb-row-between">
              <span className="wb-small">{node.name}</span>
              <span className="wb-tag">{node.rungName || node.kind}</span>
            </div>
          ))
        )}
      </div>
      <div className="wb-stack tight">
        <span className="wb-caption">
          {t("workbench.component.consumers", {
            defaultValue: "Consumers · {{count}}",
            count: data.transitiveDependentCount,
          })}
        </span>
        {data.directDependents.length === 0 ? (
          <span className="wb-muted wb-small">
            {t("workbench.component.none", { defaultValue: "None" })}
          </span>
        ) : (
          data.directDependents.map((node) => (
            <div key={node.assetId} className="wb-row-between">
              <span className="wb-small">{node.name}</span>
              <GitBranch aria-hidden size={12} style={{ color: "var(--wb-muted-2)" }} />
            </div>
          ))
        )}
      </div>
    </div>
  );
}
