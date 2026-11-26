import { CircleAlert, FileCode2 } from "lucide-react";

import { ComponentVersionStatus } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { useTranslation } from "../../i18n";
import {
  assetAdoptions,
  formatRelative,
  timestampToDate,
  type ComponentBundle,
} from "../data";
import { Banner, Chip, Empty, Tabs } from "../primitives";
import { evidenceStatusCopy, evidenceStatusFor, evidenceStatusTone } from "../experienceSummary";
import { asVerdict, verdictTone } from "../verdicts";
import { AssetRelationshipsTab } from "./AssetRelationshipsTab";
import { AssetTestEvidenceTab } from "./AssetTestEvidenceTab";
import { SourceWorkbench } from "./SourceWorkbench";

export type DrawerTab =
  | "overview"
  | "source"
  | "versions"
  | "experience"
  | "tests"
  | "relationships";

export function AssetDataDrawer({
  bundle,
  tab,
  onTab,
  version,
  onVersion,
  selectedTestReportID,
  onSelectTestReport,
  onSourceSaved,
}: {
  bundle: ComponentBundle;
  tab: DrawerTab;
  onTab: (tab: DrawerTab) => void;
  version: string;
  onVersion: (version: string) => void;
  selectedTestReportID?: string | null;
  onSelectTestReport?: (reportID: string | null) => void;
  onSourceSaved?: (version: string, filePath: string, sha256: string) => void;
}) {
  const { t } = useTranslation();
  const versionRows = bundle.versions;
  const selectedVersion = versionRows.find((row) => row.version === version) ?? versionRows[0];

  return (
    <section
      className="wb-drawer"
      aria-label={t("workbench.component.drawer", { defaultValue: "Asset data" })}
    >
      <div className="wb-drawer-tabs">
        <Tabs
          label={t("workbench.component.drawerTabs", { defaultValue: "Asset data sections" })}
          value={tab}
          onChange={onTab}
          items={[
            {
              id: "overview",
              label: t("workbench.component.overview", { defaultValue: "Overview" }),
            },
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
            {
              id: "relationships",
              label: t("workbench.component.relationships", { defaultValue: "Relationships" }),
            },
          ]}
        />
      </div>
      <div className="wb-drawer-body">
        {tab === "overview" ? <OverviewTab bundle={bundle} /> : null}
        {tab === "source" ? (
          <SourceTab
            bundle={bundle}
            version={selectedVersion?.version ?? version}
            onVersion={onVersion}
            onSaved={onSourceSaved}
          />
        ) : null}
        {tab === "versions" ? (
          <VersionsTab
            bundle={bundle}
            selected={selectedVersion?.version ?? version}
            onSelect={onVersion}
          />
        ) : null}
        {tab === "experience" ? <ExperienceTab bundle={bundle} /> : null}
        {tab === "tests" ? (
          <AssetTestEvidenceTab
            bundle={bundle}
            version={selectedVersion?.version ?? version}
            selectedReportID={selectedTestReportID ?? null}
            onSelectReport={onSelectTestReport}
          />
        ) : null}
        {tab === "relationships" ? <AssetRelationshipsTab bundle={bundle} /> : null}
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
            <span
              className="wb-mono wb-small"
              style={{ textAlign: "right", overflowWrap: "anywhere" }}
            >
              {value}
            </span>
          </div>
        ))}
      </div>
      <div className="wb-stack tight">
        <div>
          <span className="wb-caption">
            {t("workbench.component.tags", { defaultValue: "Tags" })}
          </span>
          <div className="wb-row" style={{ flexWrap: "wrap", marginTop: 6 }}>
            {asset.tags.length === 0 ? (
              <span className="wb-muted wb-small">
                {t("workbench.component.noTags", { defaultValue: "No tags" })}
              </span>
            ) : (
              asset.tags.map((tag) => <Chip key={tag}>{tag}</Chip>)
            )}
          </div>
        </div>
        <div>
          <span className="wb-caption">
            {t("workbench.component.affinities", { defaultValue: "Design affinities" })}
          </span>
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
          <span className="wb-caption">
            {t("workbench.component.deps", { defaultValue: "Declared dependencies" })}
          </span>
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
  onSaved,
}: {
  bundle: ComponentBundle;
  version: string;
  onVersion: (version: string) => void;
  onSaved?: (version: string, filePath: string, sha256: string) => void;
}) {
  return (
    <SourceWorkbench bundle={bundle} version={version} onVersion={onVersion} onSaved={onSaved} />
  );
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
      <Empty
        title={t("workbench.component.noVersions", { defaultValue: "No version history recorded" })}
      >
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
      <Empty
        title={t("workbench.component.noExperience", { defaultValue: "No experience contract" })}
      >
        {t("workbench.component.noExperienceBody", {
          defaultValue:
            "This asset has no claim/evidence contract, so nothing is asserted about its states.",
        })}
      </Empty>
    );
  }
  const status = evidenceStatusFor(experience);
  const statusCopy = evidenceStatusCopy(status);
  const verdicts = experience.evidence.reduce<Record<string, number>>((counts, row) => {
    const verdict = asVerdict(row.verdict);
    counts[verdict] = (counts[verdict] ?? 0) + 1;
    return counts;
  }, {});
  return (
    <div className="wb-stack">
      <div className="wb-row" style={{ flexWrap: "wrap" }}>
        <Chip tone={evidenceStatusTone(status)}>
          {t(statusCopy.key, { defaultValue: statusCopy.defaultValue })}
        </Chip>
        <span className="wb-small wb-muted">{experience.evidenceMessage}</span>
      </div>
      <div className="wb-row" style={{ flexWrap: "wrap" }}>
        {Object.entries(verdicts).map(([verdict, count]) => (
          <Chip key={verdict} tone={verdictTone(asVerdict(verdict))}>
            {verdict} · {count}
          </Chip>
        ))}
      </div>
      <div className="wb-stack tight">
        {experience.claims.map((claim) => {
          const evidence = experience.evidence.filter((row) => row.claimId === claim.id);
          const verdict = asVerdict(evidence[0]?.verdict);
          return (
            <div
              key={claim.id}
              className="wb-row-between"
              style={{ alignItems: "flex-start", gap: 12 }}
            >
              <span style={{ minWidth: 0 }}>
                <b className="wb-small">{claim.statement}</b>
                <small className="wb-muted-2 wb-small" style={{ display: "block" }}>
                  {claim.type} · {claim.tier} · {claim.states.join(", ") || "no states"}
                </small>
              </span>
              <Chip tone={verdictTone(verdict)}>{verdict}</Chip>
            </div>
          );
        })}
      </div>
    </div>
  );
}
