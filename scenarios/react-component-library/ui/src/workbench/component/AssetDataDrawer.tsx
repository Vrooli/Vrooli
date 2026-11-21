import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CircleAlert, FileCode2, GitBranch, Play, RotateCw } from "lucide-react";

import { ComponentVersionStatus } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { getAssetRelationships } from "../../api/catalogGraph";
import { runComponentTest } from "../../api/componentTests";
import { useTranslation } from "../../i18n";
import {
  assetAdoptions,
  formatRelative,
  timestampToDate,
  wbKeys,
  type ComponentBundle,
} from "../data";
import { Banner, Chip, Empty, Skeleton, Tabs, WbButton } from "../primitives";
import { SourceWorkbench } from "./SourceWorkbench";

export type DrawerTab = "overview" | "source" | "versions" | "experience" | "tests" | "relationships";

export function AssetDataDrawer({
  bundle,
  tab,
  onTab,
  version,
  onVersion,
}: {
  bundle: ComponentBundle;
  tab: DrawerTab;
  onTab: (tab: DrawerTab) => void;
  version: string;
  onVersion: (version: string) => void;
}) {
  const { t } = useTranslation();
  const versionRows = bundle.versions;
  const selectedVersion = versionRows.find((row) => row.version === version) ?? versionRows[0];

  return (
    <section className="wb-drawer" aria-label={t("workbench.component.drawer", { defaultValue: "Asset data" })}>
      <div className="wb-drawer-tabs">
        <Tabs
          label={t("workbench.component.drawerTabs", { defaultValue: "Asset data sections" })}
          value={tab}
          onChange={onTab}
          items={[
            { id: "overview", label: t("workbench.component.overview", { defaultValue: "Overview" }) },
            { id: "source", label: t("workbench.component.source", { defaultValue: "Source" }) },
            {
              id: "versions",
              label: t("workbench.component.versions", { defaultValue: "Versions" }),
              count: versionRows.length,
            },
            {
              id: "experience",
              label: t("workbench.component.experience", { defaultValue: "Experience" }),
              count: bundle.experience?.claims.length,
            },
            {
              id: "tests",
              label: t("workbench.component.tests", { defaultValue: "Tests" }),
              count: bundle.testReports.length,
            },
            { id: "relationships", label: t("workbench.component.relationships", { defaultValue: "Relationships" }) },
          ]}
        />
      </div>
      <div className="wb-drawer-body">
        {tab === "overview" ? <OverviewTab bundle={bundle} /> : null}
        {tab === "source" ? (
          <SourceTab bundle={bundle} version={selectedVersion?.version ?? version} onVersion={onVersion} />
        ) : null}
        {tab === "versions" ? (
          <VersionsTab bundle={bundle} selected={selectedVersion?.version ?? version} onSelect={onVersion} />
        ) : null}
        {tab === "experience" ? <ExperienceTab bundle={bundle} /> : null}
        {tab === "tests" ? <TestsTab bundle={bundle} version={selectedVersion?.version ?? version} /> : null}
        {tab === "relationships" ? <RelationshipsTab bundle={bundle} /> : null}
        {bundle.partial.length > 0 ? (
          <div style={{ marginTop: 14 }}>
            <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
              {t("workbench.component.partial", {
                defaultValue:
                  "Some evidence owners did not answer: {{parts}}. Their sections show an explicit unavailable state rather than an empty result.",
                parts: bundle.partial.join(", "),
              })}
            </Banner>
          </div>
        ) : null}
      </div>
    </section>
  );
}

function OverviewTab({ bundle }: { bundle: ComponentBundle }) {
  const { t } = useTranslation();
  const { asset } = bundle;
  const rows: Array<[string, string]> = [
    [t("workbench.component.libraryId", { defaultValue: "Library id" }), asset.libraryId],
    [t("workbench.component.slot", { defaultValue: "Slot" }), asset.slot || "—"],
    [t("workbench.component.category", { defaultValue: "Category" }), asset.category || "—"],
    [t("workbench.component.kind", { defaultValue: "Kind" }), asset.catalogKind || "—"],
    [t("workbench.component.domain", { defaultValue: "Domain" }), asset.catalogDomain || "—"],
    [t("workbench.component.sourcePath", { defaultValue: "Source path" }), asset.sourcePath || "—"],
    [t("workbench.component.manifest", { defaultValue: "Manifest" }), asset.manifestPath || "—"],
    [
      t("workbench.component.directAdoptions", { defaultValue: "Direct adoptions" }),
      String(assetAdoptions(asset)),
    ],
    [
      t("workbench.component.dependencies", { defaultValue: "Dependencies" }),
      String(asset.dependencies.length),
    ],
  ];
  return (
    <div className="wb-drawer-grid">
      <div className="wb-stack tight">
        {rows.map(([label, value]) => (
          <div key={label} className="wb-row-between" style={{ alignItems: "baseline", gap: 16 }}>
            <span className="wb-caption">{label}</span>
            <span className="wb-mono wb-small" style={{ textAlign: "right", overflowWrap: "anywhere" }}>
              {value}
            </span>
          </div>
        ))}
      </div>
      <div className="wb-stack tight">
        <div>
          <span className="wb-caption">{t("workbench.component.tags", { defaultValue: "Tags" })}</span>
          <div className="wb-row" style={{ flexWrap: "wrap", marginTop: 6 }}>
            {asset.tags.length === 0 ? (
              <span className="wb-muted wb-small">{t("workbench.component.noTags", { defaultValue: "No tags" })}</span>
            ) : (
              asset.tags.map((tag) => <Chip key={tag}>{tag}</Chip>)
            )}
          </div>
        </div>
        <div>
          <span className="wb-caption">{t("workbench.component.affinities", { defaultValue: "Design affinities" })}</span>
          <div className="wb-row" style={{ flexWrap: "wrap", marginTop: 6 }}>
            {asset.designStyles.length === 0 ? (
              <span className="wb-muted wb-small">
                {t("workbench.component.noAffinities", { defaultValue: "Not declared" })}
              </span>
            ) : (
              asset.designStyles.map((style) => (
                <Chip key={style.styleId} tone="ice">
                  {style.styleId}
                </Chip>
              ))
            )}
          </div>
        </div>
        <div>
          <span className="wb-caption">{t("workbench.component.deps", { defaultValue: "Declared dependencies" })}</span>
          <div className="wb-stack tight" style={{ marginTop: 6 }}>
            {asset.dependencies.length === 0 ? (
              <span className="wb-muted wb-small">
                {t("workbench.component.noDeps", { defaultValue: "None declared" })}
              </span>
            ) : (
              asset.dependencies.map((dep) => (
                <div key={`${dep.libraryId}@${dep.version}`} className="wb-row-between">
                  <span className="wb-mono wb-small">{dep.libraryId}</span>
                  <span className="wb-tag">{dep.version}</span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function SourceTab({
  bundle,
  version,
  onVersion,
}: {
  bundle: ComponentBundle;
  version: string;
  onVersion: (version: string) => void;
}) {
  return <SourceWorkbench bundle={bundle} version={version} onVersion={onVersion} />;
}

function VersionsTab({
  bundle,
  selected,
  onSelect,
}: {
  bundle: ComponentBundle;
  selected: string;
  onSelect: (version: string) => void;
}) {
  const { t } = useTranslation();
  if (bundle.versions.length === 0) {
    return (
      <Empty title={t("workbench.component.noVersions", { defaultValue: "No version history recorded" })}>
        {t("workbench.component.noVersionsBody", {
          defaultValue:
            "The versions owner returned no rows. This is shown as unavailable history, not as a release with no changes.",
        })}
      </Empty>
    );
  }
  return (
    <div className="wb-list">
      {bundle.versions.map((row) => (
        <button
          key={row.id}
          type="button"
          className="wb-list-row"
          style={{ width: "100%", border: 0, background: "none", textAlign: "left" }}
          aria-pressed={row.version === selected}
          onClick={() => onSelect(row.version)}
        >
          <FileCode2 aria-hidden size={14} style={{ color: "var(--wb-muted-2)" }} />
          <span style={{ minWidth: 0, flex: 1 }}>
            <b>{row.version}</b>
            <small style={{ display: "block" }}>{row.sourcePath}</small>
          </span>
          <Chip
            tone={
              row.status === ComponentVersionStatus.RELEASED
                ? "green"
                : row.status === ComponentVersionStatus.DRAFT
                  ? "amber"
                  : "ice"
            }
          >
            {row.status === ComponentVersionStatus.RELEASED
              ? t("workbench.component.released", { defaultValue: "Released" })
              : row.status === ComponentVersionStatus.DRAFT
                ? t("workbench.component.draft", { defaultValue: "Draft" })
                : t("workbench.component.indexed", { defaultValue: "Indexed" })}
          </Chip>
          <span className="wb-small wb-muted-2">
            {t("workbench.component.fileCount", {
              count: row.files.length,
            })}
          </span>
          <span className="wb-small wb-muted-2">
            {formatRelative(timestampToDate(row.releasedAt) ?? timestampToDate(row.indexedAt))}
          </span>
        </button>
      ))}
    </div>
  );
}

function ExperienceTab({ bundle }: { bundle: ComponentBundle }) {
  const { t } = useTranslation();
  const experience = bundle.experience;
  if (!experience) {
    return (
      <Empty title={t("workbench.component.noExperience", { defaultValue: "No experience contract" })}>
        {t("workbench.component.noExperienceBody", {
          defaultValue: "This asset has no claim/evidence contract, so nothing is asserted about its states.",
        })}
      </Empty>
    );
  }
  const verdicts = experience.evidence.reduce<Record<string, number>>((counts, row) => {
    counts[row.verdict] = (counts[row.verdict] ?? 0) + 1;
    return counts;
  }, {});
  return (
    <div className="wb-stack">
      <div className="wb-row" style={{ flexWrap: "wrap" }}>
        <Chip tone={experience.evidenceStatus === "fresh" ? "green" : "amber"}>
          {experience.evidenceStatus || "unknown"}
        </Chip>
        <span className="wb-small wb-muted">{experience.evidenceMessage}</span>
      </div>
      <div className="wb-row" style={{ flexWrap: "wrap" }}>
        {Object.entries(verdicts).map(([verdict, count]) => (
          <Chip key={verdict} tone={verdict === "pass" ? "green" : verdict === "fail" ? "red" : "amber"}>
            {verdict} · {count}
          </Chip>
        ))}
      </div>
      <div className="wb-stack tight">
        {experience.claims.map((claim) => {
          const evidence = experience.evidence.filter((row) => row.claimId === claim.id);
          const verdict = evidence[0]?.verdict ?? "unmeasured";
          return (
            <div key={claim.id} className="wb-row-between" style={{ alignItems: "flex-start", gap: 12 }}>
              <span style={{ minWidth: 0 }}>
                <b className="wb-small">{claim.statement}</b>
                <small className="wb-muted-2 wb-small" style={{ display: "block" }}>
                  {claim.type} · {claim.tier} · {claim.states.join(", ") || "no states"}
                </small>
              </span>
              <Chip tone={verdict === "pass" ? "green" : verdict === "fail" ? "red" : "amber"}>{verdict}</Chip>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function TestsTab({ bundle, version }: { bundle: ComponentBundle; version: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const run = useMutation({
    mutationFn: () =>
      runComponentTest({ componentId: bundle.asset.id, version, includeClosure: false }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.component(bundle.asset.id) });
    },
  });

  return (
    <div className="wb-stack">
      <div className="wb-row-between">
        <span className="wb-caption">
          {t("workbench.component.testReports", { defaultValue: "Component test evidence" })}
        </span>
        <WbButton variant="ghost" size="sm" onClick={() => run.mutate()} disabled={run.isPending}>
          {run.isPending ? <RotateCw aria-hidden size={13} /> : <Play aria-hidden size={13} />}
          {run.isPending
            ? t("workbench.component.running", { defaultValue: "Running…" })
            : t("workbench.component.runTest", { defaultValue: "Run test" })}
        </WbButton>
      </div>
      {run.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
          {run.error instanceof Error ? run.error.message : String(run.error)}
        </Banner>
      ) : null}
      {bundle.testReports.length === 0 ? (
        <Empty title={t("workbench.component.noTests", { defaultValue: "No component test evidence yet" })}>
          {t("workbench.component.noTestsBody", {
            defaultValue:
              "A release without a current test report is not proven. Run the focused component test to produce one.",
          })}
        </Empty>
      ) : (
        bundle.testReports.map((report) => (
          <div key={report.id} className="wb-panel" style={{ padding: 12 }}>
            <div className="wb-row-between">
              <span className="wb-mono wb-small">{report.id.slice(0, 12)}</span>
              <Chip tone={report.verdict === "passed" ? "green" : report.verdict === "failed" ? "red" : "amber"}>
                {report.verdict}
              </Chip>
              <span className="wb-small wb-muted-2">
                {formatRelative(timestampToDate(report.createdAt))}
              </span>
            </div>
            <div className="wb-stack tight" style={{ marginTop: 8 }}>
              {report.results.slice(0, 8).map((result) => (
                <div key={`${result.stage}:${result.subject}`} className="wb-checkrow">
                  <span style={{ color: result.verdict === "passed" ? "var(--wb-green)" : "var(--wb-red)" }}>●</span>
                  <span>{result.stage}</span>
                  <b>{result.verdict}</b>
                </div>
              ))}
            </div>
          </div>
        ))
      )}
    </div>
  );
}

function RelationshipsTab({ bundle }: { bundle: ComponentBundle }) {
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
      <Empty title={t("workbench.component.noRelationships", { defaultValue: "No relationships recorded" })}>
        {t("workbench.component.noRelationshipsBody", {
          defaultValue: "This asset has no declared dependencies or consumers in the catalog graph.",
        })}
      </Empty>
    );
  }
  return (
    <div className="wb-drawer-grid">
      <div className="wb-stack tight">
        <span className="wb-caption">{t("workbench.component.dependsOn", { defaultValue: "Depends on" })}</span>
        {data.directDependencies.length === 0 ? (
          <span className="wb-muted wb-small">{t("workbench.component.none", { defaultValue: "None" })}</span>
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
          <span className="wb-muted wb-small">{t("workbench.component.none", { defaultValue: "None" })}</span>
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
