import { useEffect, useRef, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { create } from "@bufbuild/protobuf";
import { Link, Navigate, useLocation, useParams } from "react-router-dom";
import { Code, ConnectError } from "@connectrpc/connect";
import { Card, CardContent, CardHeader, CardTitle } from "../components/Card";
import { Button } from "../components/Button";
import { Input } from "../components/Input";
import { Select } from "../components/Select";
import { searchDesignAssets } from "../api/catalog";
import type {
  CandidateResponse,
  CaptureOperation,
  CritiqueEvidence,
  DesignIntent,
  Sketch,
} from "@vrooli/proto-types/react-component-library/v1/sketch/sketch_pb";
import {
  DesignConstraintsSchema,
  DesignIntentSchema,
} from "@vrooli/proto-types/react-component-library/v1/sketch/sketch_pb";
import { listComponentStories } from "../api/components";
import { API_BASE } from "../api/client";
import { generateReferenceAsset, sketchClient, uploadReferenceAsset } from "../api/sketch";
import { strings } from "../consts/strings";
import { designPath } from "../routes";
import { useTranslation } from "../i18n";

function Failure({
  error,
  retry,
  conflictText,
}: {
  error: unknown;
  retry?: () => void;
  conflictText?: string;
}) {
  const { t } = useTranslation();
  return (
    <div role="alert" className="rounded-control border border-app-border p-space-sm">
      <p>
        {ConnectError.from(error).code === Code.Aborted
          ? (conflictText ?? t("design.conflict"))
          : error instanceof Error
            ? error.message
            : t("design.failed")}
      </p>
      {retry && <Button onClick={retry}>{t("design.reload")}</Button>}
    </div>
  );
}

export function DesignPage() {
  const { scenario, page } = useParams();
  const location = useLocation();
  if (location.pathname.startsWith("/sketch/"))
    return <Navigate replace to={designPath(scenario, page)} />;
  return page && scenario ? (
    <PageWorkspace key={`${scenario}/${page}`} scenario={scenario} page={page} />
  ) : (
    <DesignOverview key={scenario ?? "all"} scenario={scenario} />
  );
}

function DesignOverview({ scenario }: { scenario?: string }) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState("");
  const [showEmpty, setShowEmpty] = useState(false);
  const inventory = useQuery({
    queryKey: ["design-inventory", scenario],
    queryFn: ({ signal }) => sketchClient.listDesignPages({ scenario: scenario ?? "" }, { signal }),
  });
  return (
    <section
      data-experience-surface="design-overview"
      className="grid gap-space-sm"
      aria-label={t("design.title")}
    >
      {scenario && (
        <nav aria-label={t("design.breadcrumb")}>
          <Link
            to={designPath()}
            className="inline-flex min-h-11 items-center rounded-control px-space-2xs text-sm text-app-primary hover:bg-app-surface-muted"
          >
            {t("design.title")}
          </Link>
          {scenario && <span> / {scenario}</span>}
        </nav>
      )}
      <h2 className="text-title font-semibold">{scenario ?? t("design.chooseScenario")}</h2>
      <p className="text-app-muted-foreground">{t("design.overviewHelp")}</p>
      <Input
        aria-label={t("design.filter")}
        placeholder={t("design.filter")}
        value={filter}
        onChange={(event) => setFilter(event.target.value)}
      />
      {inventory.isPending && <p role="status">{t("design.loading")}</p>}
      {inventory.error && (
        <Failure error={inventory.error} retry={() => void inventory.refetch()} />
      )}
      {inventory.data?.issues.map((issue) => (
        <p role="alert" key={issue}>
          {issue}
        </p>
      ))}
      {!scenario && (
        <label className="relative flex min-h-11 items-center gap-space-2xs text-label text-app-muted-foreground">
          <input
            className="peer absolute inset-0 h-full w-full cursor-pointer opacity-0"
            type="checkbox"
            checked={showEmpty}
            onChange={(event) => setShowEmpty(event.target.checked)}
          />
          <span
            aria-hidden="true"
            className="grid h-5 w-5 shrink-0 place-items-center rounded border border-app-border bg-app-surface text-xs peer-checked:bg-app-primary peer-checked:text-white"
          >
            {showEmpty ? "✓" : ""}
          </span>
          {t(strings.design.showEmptyScenarios)}
        </label>
      )}
      <div className="grid gap-space-2xs md:grid-cols-2 xl:grid-cols-3">
        {inventory.data?.scenarios
          .filter((item) => item.scenario.includes(filter.trim().toLowerCase()))
          .filter((item) => showEmpty || item.pageCount > 0 || item.issue || Boolean(filter.trim()))
          .sort((a, b) => b.pageCount - a.pageCount || a.scenario.localeCompare(b.scenario))
          .map((item) => (
            <Link
              key={item.scenario}
              to={designPath(item.scenario)}
              className="grid min-w-0 gap-space-3xs rounded-panel border border-app-border bg-app-surface p-space-xs hover:bg-app-surface-muted"
            >
              <span className="flex min-w-0 items-center justify-between gap-space-sm">
                <span className="min-w-0 break-words font-medium">{item.scenario}</span>
                <span className="shrink-0 text-label text-app-muted-foreground">
                  {t("design.pageCount", { count: item.pageCount })}
                </span>
              </span>
              {item.issue && (
                <span role="alert" className="text-label">
                  {item.issue}
                </span>
              )}
            </Link>
          ))}
        {inventory.data?.pages
          .filter((item) =>
            `${item.title} ${item.page}`.toLowerCase().includes(filter.toLowerCase()),
          )
          .map((item) => (
            <Card key={item.page}>
              <CardHeader>
                <CardTitle>
                  {item.contentHash ? (
                    <Link to={designPath(scenario, item.page)}>{item.title || item.page}</Link>
                  ) : (
                    item.title || item.page
                  )}
                </CardTitle>
              </CardHeader>
              <CardContent className="grid gap-space-2xs">
                <p>{item.routes.join(" · ") || item.route}</p>
                <p>{t(`design.pageStatus.${item.status}`, { defaultValue: item.status })}</p>
                <p>{t("design.regionCount", { count: item.regionCount })}</p>
                {!item.registered && <p>{t("design.unregistered")}</p>}
                {item.issue && <p role="alert">{item.issue}</p>}
              </CardContent>
            </Card>
          ))}
      </div>
      {inventory.data && scenario && inventory.data.pages.length === 0 && (
        <p>{t("design.noPages")}</p>
      )}
    </section>
  );
}

function PageWorkspace({ scenario, page }: { scenario: string; page: string }) {
  const { t } = useTranslation();
  useEffect(() => {
    document.querySelector<HTMLElement>("main")?.scrollTo({ top: 0, behavior: "auto" });
  }, [scenario, page]);
  const cache = useQueryClient();
  const target = { scenario, page };
  const key = ["design-sketch", scenario, page];
  const current = useQuery({
    queryKey: key,
    queryFn: ({ signal }) => sketchClient.getSketch({ target }, { signal }),
  });
  useEffect(() => {
    if (!current.data) return;
    const resetScroll = () => {
      const main = document.querySelector<HTMLElement>("main");
      if (main) main.scrollTop = 0;
    };
    resetScroll();
    const frame = window.requestAnimationFrame(resetScroll);
    return () => window.cancelAnimationFrame(frame);
  }, [current.data]);
  const history = useQuery({
    queryKey: ["design-history", scenario, page],
    queryFn: ({ signal }) => sketchClient.getHistory({ target }, { signal }),
  });
  const [revision, setRevision] = useState("");
  const [selected, setSelected] = useState("");
  const [note, setNote] = useState("");
  const [panel, setPanel] = useState("structure");
  const selectedRevision = history.data?.revisions.find((item) => item.contentHash === revision);
  const doc = revision ? selectedRevision?.sketch : current.data?.sketch;
  const hash = revision || current.data?.contentHash || "";
  const readOnly = Boolean(revision && revision !== current.data?.contentHash);
  const declaredRegions =
    (revision ? selectedRevision?.declaredRegions : current.data?.declaredRegions) ?? [];
  const regions = [
    ...new Set([
      ...declaredRegions.map((item) => item.id),
      ...(doc?.regions.map((item) => item.id) ?? []),
      ...(doc?.placements.map((item) => item.region) ?? []),
    ]),
  ];
  const referenceRegions = regions.filter((item) => {
    const placement = doc?.placements.find((candidate) => candidate.region === item);
    return !placement || Boolean(placement.fills?.placeholder);
  });
  const active = selected && regions.includes(selected) ? selected : (regions[0] ?? "");
  const placement = doc?.placements.find((item) => item.region === active);
  const region =
    doc?.regions.find((item) => item.id === active) ??
    declaredRegions.find((item) => item.id === active);
  const refresh = async () => {
    await Promise.all([
      cache.invalidateQueries({ queryKey: key }),
      cache.invalidateQueries({ queryKey: ["design-history", scenario, page] }),
    ]);
  };
  const saveNote = useMutation({
    mutationFn: () =>
      sketchClient.addNote({
        target,
        scope: active || "page",
        text: note.trim(),
        expectedContentHash: current.data?.contentHash ?? "",
      }),
    onSuccess: async () => {
      setNote("");
      await refresh();
    },
  });
  const toggleRegionLock = useMutation({
    mutationFn: async () => {
      if (!doc || !active) throw new Error(t("design.failed"));
      const nextRegions = doc.regions.filter((item) => item.id !== active);
      nextRegions.push({
        ...region,
        id: active,
        locked: !region?.locked,
      } as (typeof doc.regions)[number]);
      return sketchClient.putSketch({
        target,
        expectedContentHash: current.data?.contentHash ?? "",
        sketch: { ...doc, regions: nextRegions },
      });
    },
    onSuccess: refresh,
  });
  const importPage = useMutation({
    mutationFn: (write: boolean) =>
      sketchClient.importPage({
        target,
        write,
        expectedContentHash: current.data?.contentHash ?? "",
      }),
    onSuccess: async (result) => {
      if (result.written) await refresh();
    },
  });
  const verify = useMutation({
    mutationFn: async () => ({
      hash: current.data?.contentHash,
      result: await sketchClient.verifySketch({ target }),
    }),
  });
  const evidence = verify.data?.hash === hash ? verify.data.result : undefined;
  const staleEvidence = Boolean(verify.data && !evidence);
  const workflowSteps = [
    {
      id: "brief",
      label: "Brief",
      detail: "Intent & references",
      anchor: "design-brief",
      complete: Boolean(
        doc?.intent?.intent.trim() &&
          doc.intent.users.length > 0 &&
          doc.intent.primaryTasks.length > 0,
      ),
    },
    {
      id: "compose",
      label: "Compose",
      detail: "Canvas & assets",
      anchor: "design-canvas",
      complete: Boolean(doc?.placements.length),
    },
    {
      id: "review",
      label: "Review",
      detail: "Evidence & critique",
      anchor: "design-review",
      complete: Boolean(evidence),
    },
    {
      id: "adopt",
      label: "Adopt",
      detail: "Handoff & obligations",
      anchor: "design-adoption",
      complete: Boolean(evidence?.passes),
    },
  ];
  const nextStep = workflowSteps.find((step) => !step.complete) ?? workflowSteps.at(-1)!;
  return (
    <section
      data-experience-surface="design-workspace"
      className="grid min-w-0 grid-cols-1 gap-space-md"
      aria-label={t("design.workspace")}
    >
      <nav aria-label={t("design.breadcrumb")} className="flex flex-wrap gap-space-2xs">
        <Link
          to={designPath()}
          className="inline-flex min-h-11 items-center rounded-control px-space-2xs text-sm text-app-primary hover:bg-app-surface-muted"
        >
          {t("design.title")}
        </Link>
        <span>/</span>
        <Link
          to={designPath(scenario)}
          className="inline-flex min-h-11 items-center rounded-control px-space-2xs text-sm text-app-primary hover:bg-app-surface-muted"
        >
          {scenario}
        </Link>
        <span>/ {page}</span>
      </nav>
      <header
        className="grid gap-space-sm rounded-panel border border-app-border bg-app-surface p-space-sm shadow-sm lg:sticky lg:top-0 lg:z-20"
        data-testid="design-workspace-header"
      >
        <div className="flex flex-wrap items-start justify-between gap-space-sm">
          <div className="min-w-0">
            <p className="text-label font-semibold uppercase tracking-[0.14em] text-app-primary">
              Design studio
            </p>
            <h1 className="mt-1 break-words text-title font-semibold">{page}</h1>
            <p className="mt-1 max-w-2xl text-sm text-app-muted-foreground">
              Turn a brief into a reviewable, evidence-backed candidate. Technical details stay
              available when you need them.
            </p>
            <p className="mt-2 break-all text-label text-app-muted-foreground">
              {scenario} ·{" "}
              {current.data
                ? `revision ${current.data.contentHash.slice(0, 12)} · ${readOnly ? t("design.readOnly") : t("design.saved")}`
                : t("design.loading")}
            </p>
          </div>
          <div className="flex items-center gap-space-xs">
            <span className="rounded-full border border-app-border bg-app-surface-muted px-space-xs py-space-2xs text-label text-app-muted-foreground">
              {readOnly ? "Historical revision" : "Working draft"}
            </span>
            <Button
              className="min-h-11"
              disabled={!current.data || verify.isPending || readOnly}
              onClick={() => verify.mutate()}
            >
              {verify.isPending ? t("design.verifying") : t("design.verify")}
            </Button>
          </div>
        </div>
        <nav aria-label="Design workflow" className="grid gap-2 sm:grid-cols-4">
          {workflowSteps.map((step, index) => (
            <a
              key={step.id}
              href={`${designPath(scenario, page)}#${step.anchor}`}
              className={`relative min-h-touch rounded-control border p-space-xs transition-colors hover:border-app-primary/60 ${step.complete ? "border-emerald-500/40 bg-emerald-500/5" : step.id === nextStep.id ? "border-app-primary/50 bg-app-primary/5" : "border-app-border bg-app-surface-muted"}`}
            >
              <div className="flex items-center gap-2">
                <span className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-app-surface text-label font-semibold text-app-muted-foreground">
                  {index + 1}
                </span>
                <span className="inline-flex min-h-11 min-w-0 items-center text-sm font-semibold">
                  {step.label}
                </span>
                {step.complete && (
                  <span className="ml-auto text-label text-emerald-600">Ready</span>
                )}
              </div>
              <p className="mt-1 pl-8 text-label text-app-muted-foreground">{step.detail}</p>
            </a>
          ))}
        </nav>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 border-t border-app-border pt-space-xs text-sm">
          <span className="font-medium">Next best action</span>
          <a
            href={`${designPath(scenario, page)}#${nextStep.anchor}`}
            className="font-semibold text-app-primary underline-offset-2 hover:underline"
          >
            {nextStep.label}
          </a>
          <span className="text-app-muted-foreground">— {nextStep.detail}</span>
        </div>
      </header>
      {current.isPending && <p role="status">{t("design.loading")}</p>}
      {current.error && <Failure error={current.error} retry={() => void refresh()} />}
      {current.data && (
        <>
          <div>
            <IntentProposals
              target={target}
              hash={hash}
              sketch={doc}
              readOnly={readOnly}
              onSaved={refresh}
            />
          </div>
          {!readOnly && (
            <SavedCandidates
              key={`${target.scenario}/${target.page}`}
              target={target}
              onSelected={refresh}
            />
          )}
          <DesignCanvas
            target={target}
            hash={hash}
            configured={Boolean(doc?.render)}
            readOnly={readOnly}
          />
          <AssetDecisionGuide
            target={target}
            hash={current.data.contentHash}
            regions={regions}
            readOnly={readOnly}
            onSaved={refresh}
          />
          <ReferencePlaceholderEditor
            target={target}
            hash={current.data.contentHash}
            regions={referenceRegions}
            readOnly={readOnly}
            onSaved={refresh}
          />
          <ReferenceRevisionReview
            revisions={history.data?.revisions ?? []}
            currentHash={current.data.contentHash}
          />
          <TemplateEditor
            target={target}
            hash={current.data.contentHash}
            regions={[
              ...(doc?.regions.map((r) => r.id) ?? []),
              ...(doc?.placements
                .filter((p) => !doc.regions.some((r) => r.id === p.region))
                .map((p) => p.region) ?? []),
            ]}
            readOnly={readOnly}
            onSaved={refresh}
          />
          <div className="flex flex-wrap items-center gap-space-sm">
            <label htmlFor="design-revision">{t("design.revision")}</label>
            <Select
              id="design-revision"
              value={revision}
              options={[
                { value: "", label: t("design.current") },
                ...(history.data?.revisions
                  .filter((item) => !item.current)
                  .map((item) => ({
                    value: item.contentHash,
                    label: `${item.createdAt} · ${item.contentHash.slice(0, 12)}`,
                  })) ?? []),
              ]}
              onChange={(event) => setRevision(event.target.value)}
              className="min-h-touch max-w-full rounded-control border border-app-border bg-app-surface px-space-xs"
            />
            <span className="break-all text-sm text-app-muted-foreground">
              {hash.slice(0, 12)} · {readOnly ? t("design.readOnly") : t("design.saved")}
            </span>
          </div>
          {history.error && <Failure error={history.error} retry={() => void history.refetch()} />}
          <div
            className="flex flex-wrap gap-space-2xs md:hidden"
            role="group"
            aria-label={t("design.panels")}
          >
            {(["structure", "findings", "changes"] as const).map((name) => (
              <Button key={name} aria-pressed={panel === name} onClick={() => setPanel(name)}>
                {t(`design.${name}`)}
              </Button>
            ))}
          </div>
          <div className="grid min-w-0 grid-cols-1 gap-space-md md:grid-cols-2">
            <Card className={panel !== "structure" ? "hidden md:block" : ""}>
              <CardHeader>
                <CardTitle>{t("design.structure")}</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-space-sm">
                {doc?.template && (
                  <p>
                    {t("design.template")}: {doc.template.asset} {doc.template.version}
                  </p>
                )}
                {regions.length === 0 && <p>{t("design.noRegions")}</p>}
                <div
                  className="flex flex-wrap gap-space-2xs"
                  role="group"
                  aria-label={t("design.regions")}
                >
                  {regions.map((id) => (
                    <Button key={id} aria-pressed={active === id} onClick={() => setSelected(id)}>
                      {id}
                    </Button>
                  ))}
                </div>
                {active && (
                  <div className="grid gap-space-2xs border-t border-app-border pt-space-sm">
                    <h2 className="font-semibold">{active}</h2>
                    <Button
                      disabled={readOnly || toggleRegionLock.isPending}
                      onClick={() => toggleRegionLock.mutate()}
                    >
                      {t(region?.locked ? "design.unlockRegion" : "design.lockRegion")}
                    </Button>
                    {region?.locked && <p>{t("design.regionLockedHelp")}</p>}
                    {toggleRegionLock.error && <Failure error={toggleRegionLock.error} />}

                    <p>{region?.note || placement?.intent || placement?.fills?.intent}</p>
                    <p>
                      {placement?.fills?.asset ||
                        placement?.fills?.placeholder ||
                        t("design.unfilled")}
                    </p>
                    <p>{placement?.fills?.version}</p>
                    {region?.elements.length ? (
                      <p>
                        {t("design.elements")}: {region.elements.join(", ")}
                      </p>
                    ) : null}
                  </div>
                )}
                {doc?.unplaced.length ? (
                  <div>
                    <h2 className="font-semibold">{t("design.unplaced")}</h2>
                    {doc.unplaced.map((item, index) => (
                      <p key={`${item.was}-${index}`}>
                        {item.was}: {item.reason}
                      </p>
                    ))}
                  </div>
                ) : null}
              </CardContent>
            </Card>
            <div id="design-review" className="scroll-mt-24">
              <Card className={panel !== "findings" ? "hidden md:block" : ""}>
                <CardHeader>
                  <CardTitle>{t("design.findings")}</CardTitle>
                </CardHeader>
                <CardContent className="grid gap-space-sm">
                  {verify.error && <Failure error={verify.error} />}
                  {staleEvidence && <p role="status">{t("design.staleEvidence")}</p>}
                  {!evidence && <p>{t("design.verifyHelp")}</p>}
                  {evidence && (
                    <>
                      <p role="status">
                        {evidence.passes
                          ? t("design.verificationPassed")
                          : t("design.verificationNeedsWork")}
                      </p>
                      <p>
                        {evidence.coverage?.status === "not_applicable"
                          ? t("design.coverageNA")
                          : t("design.coverage", {
                              built: evidence.coverage?.built,
                              total: evidence.coverage?.total,
                            })}
                      </p>
                      <p>
                        {t(strings.design.sourceCoverage, {
                          resolved: evidence.coverage?.resolved ?? 0,
                          total: evidence.coverage?.total ?? 0,
                          custom: evidence.coverage?.resolvedLocal ?? 0,
                        })}
                      </p>
                      <p>
                        {t(strings.design.declaredCoverage, {
                          library: evidence.coverage?.libraryBacked ?? 0,
                          local: evidence.coverage?.local ?? 0,
                        })}
                      </p>
                      <ul className="grid gap-space-sm">
                        {evidence.regions.map((item, index) => (
                          <li key={`${item.region}-${index}`}>
                            <strong>{item.region || item.filePath}</strong>
                            <p>
                              {t(`design.verdict.${item.verdict}`, { defaultValue: item.verdict })}:{" "}
                              {item.reason}
                            </p>
                          </li>
                        ))}
                      </ul>
                    </>
                  )}
                </CardContent>
              </Card>
            </div>
            <Card className={`md:col-span-2 ${panel !== "changes" ? "hidden md:block" : ""}`}>
              <CardHeader>
                <CardTitle>{t("design.changes")}</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-space-sm">
                <ul>
                  {doc?.notes.map((item, index) => (
                    <li key={`${item.scope}-${index}`}>
                      <strong>{item.scope}</strong>: {item.text}
                    </li>
                  ))}
                </ul>
                <form
                  className="grid gap-space-sm"
                  onSubmit={(event) => {
                    event.preventDefault();
                    saveNote.mutate();
                  }}
                >
                  <label htmlFor="design-note">
                    {t("design.note", { region: active || t("design.wholePage") })}
                  </label>
                  <Input
                    id="design-note"
                    value={note}
                    onChange={(event) => setNote(event.target.value)}
                    disabled={readOnly || saveNote.isPending}
                  />
                  <Button type="submit" disabled={!note.trim() || readOnly || saveNote.isPending}>
                    {saveNote.isPending ? t("design.saving") : t("design.saveNote")}
                  </Button>
                </form>
                {saveNote.error && <Failure error={saveNote.error} retry={() => void refresh()} />}
              </CardContent>
            </Card>
          </div>
          <div id="design-adoption" className="scroll-mt-24 grid gap-space-sm">
            <Button
              size="sm"
              variant="secondary"
              className="justify-self-start"
              disabled={readOnly || importPage.isPending}
              onClick={() => importPage.mutate(false)}
            >
              {importPage.isPending ? t("design.importing") : t("design.previewImport")}
            </Button>
            {importPage.error && <Failure error={importPage.error} retry={() => void refresh()} />}
            {importPage.data && (
              <Card>
                <CardHeader>
                  <CardTitle>{t("design.importReview")}</CardTitle>
                </CardHeader>
                <CardContent className="grid gap-space-sm">
                  <p>{t("design.importHelp")}</p>
                  <ul>
                    {importPage.data.items.map((item) => (
                      <li key={item.component}>
                        <strong>{item.component}</strong>:{" "}
                        {t(`design.importState.${item.state}`, { defaultValue: item.state })}
                        {item.checks.map((check) => (
                          <p key={check.tier}>
                            {t("design.reuseCheck", {
                              tier: check.tier,
                              matches: check.matches.join(", ") || t("design.noMatches"),
                            })}
                          </p>
                        ))}
                      </li>
                    ))}
                  </ul>
                  <Button
                    disabled={
                      readOnly ||
                      importPage.data.written ||
                      importPage.data.contentHash !== current.data.contentHash
                    }
                    onClick={() => importPage.mutate(true)}
                  >
                    {importPage.data.written ? t("design.imported") : t("design.applyImport")}
                  </Button>
                </CardContent>
              </Card>
            )}
          </div>
        </>
      )}
    </section>
  );
}

function TemplateEditor({
  target,
  hash,
  regions,
  readOnly,
  onSaved,
}: {
  target: { scenario: string; page: string };
  hash: string;
  regions: string[];
  readOnly: boolean;
  onSaved: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [asset, setAsset] = useState("");
  const [mapping, setMapping] = useState<Record<string, string>>({});
  const [confirmed, setConfirmed] = useState(false);
  const catalog = useQuery({
    queryKey: ["design-templates"],
    queryFn: ({ signal }) => searchDesignAssets("", "page-template", signal),
    enabled: open,
  });
  const selected = catalog.data?.results.find((item) => item.catalogId === asset);
  const remap = Object.entries(mapping)
    .filter(([, to]) => to !== "")
    .map(([from, to]) => ({ from, to }));
  const request = {
    target,
    asset,
    version: selected?.implemented ? selected.version : "",
    remap,
    expectedContentHash: hash,
  };
  const signature = JSON.stringify(request);
  const mutation = useMutation({
    mutationFn: async (preview: boolean) => ({
      response: await sketchClient.setTemplate({
        ...request,
        preview,
        confirmUnplaced: preview ? false : confirmed,
      }),
      signature,
    }),
    onSuccess: async (result) => {
      if (result.response.changed) await onSaved();
    },
  });
  const proposal = mutation.data?.signature === signature ? mutation.data.response : undefined;
  return (
    <details
      onToggle={(event) => setOpen(event.currentTarget.open)}
      className="rounded-control border border-app-border p-space-sm"
    >
      <summary className="cursor-pointer font-semibold">{t("design.changeTemplate")}</summary>
      <div className="grid gap-space-sm pt-space-sm">
        {catalog.isPending && open && <p role="status">{t("design.loading")}</p>}
        {catalog.error && <Failure error={catalog.error} retry={() => void catalog.refetch()} />}
        <label htmlFor="design-template">{t("design.template")}</label>
        <Select
          id="design-template"
          className="min-h-touch min-w-0 max-w-full rounded-control border border-app-border bg-app-surface"
          value={asset}
          options={[
            { value: "", label: t("design.chooseTemplate") },
            ...(catalog.data?.results.map((item) => ({
              value: item.catalogId,
              label: `${item.name} · ${item.implemented ? t("design.published") : t("design.declarationOnly")}`,
            })) ?? []),
          ]}
          disabled={readOnly || mutation.isPending}
          onChange={(event) => {
            setAsset(event.target.value);
            setMapping({});
            setConfirmed(false);
          }}
        />
        {selected && (
          <>
            <p>{selected.description}</p>
            {regions.map((from, index) => (
              <div key={from} className="grid min-w-0 gap-space-2xs sm:grid-cols-2">
                <label htmlFor={`template-remap-${index}`}>{from}</label>
                <Select
                  id={`template-remap-${index}`}
                  className="min-h-touch min-w-0 max-w-full rounded-control border border-app-border bg-app-surface"
                  value={mapping[from] ?? ""}
                  options={[
                    {
                      value: "",
                      label: selected.regions.includes(from)
                        ? t("design.keepRegion")
                        : t("design.leaveUnmapped"),
                    },
                    ...selected.regions.map((to) => ({ value: to, label: to })),
                  ]}
                  disabled={readOnly || mutation.isPending}
                  onChange={(event) => {
                    setMapping({ ...mapping, [from]: event.target.value });
                    setConfirmed(false);
                  }}
                />
              </div>
            ))}
            <Button disabled={readOnly || mutation.isPending} onClick={() => mutation.mutate(true)}>
              {t("design.previewRemap")}
            </Button>
          </>
        )}
        {mutation.error && <Failure error={mutation.error} />}
        {proposal && (
          <>
            <ul>
              {proposal.newlyUnplaced.map((item, index) => (
                <li key={`${item.region}-${index}`}>
                  <strong>{item.region}</strong>: {item.was} — {item.reason}
                </li>
              ))}
            </ul>
            {proposal.newlyUnplaced.length > 0 && (
              <label className="flex items-start gap-space-2xs">
                <input
                  type="checkbox"
                  checked={confirmed}
                  onChange={(event) => setConfirmed(event.target.checked)}
                />
                {t("design.confirmUnplaced", { count: proposal.newlyUnplaced.length })}
              </label>
            )}
            <Button
              disabled={
                readOnly || proposal.changed || (proposal.newlyUnplaced.length > 0 && !confirmed)
              }
              onClick={() => mutation.mutate(false)}
            >
              {t("design.applyTemplate")}
            </Button>
          </>
        )}
      </div>
    </details>
  );
}

function AssetDecisionGuide({
  target,
  hash,
  regions,
  readOnly,
  onSaved,
}: {
  target: { scenario: string; page: string };
  hash: string;
  regions: string[];
  readOnly: boolean;
  onSaved: () => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const [region, setRegion] = useState(regions[0] ?? "");
  const [decision, setDecision] = useState("adopt");
  const [reason, setReason] = useState("");
  const [savedDecision, setSavedDecision] = useState("");
  const activeRegion = regions.includes(region) ? region : (regions[0] ?? "");
  const decisions = [
    {
      id: "adopt",
      label: "Adopt",
      detail: "The library asset fits the concept and its existing contract.",
    },
    {
      id: "repair",
      label: "Repair / version",
      detail: "The concept fits, but the shared asset has a defect worth fixing centrally.",
    },
    {
      id: "extend",
      label: "Extend",
      detail: "The asset fits, but a reusable variant or hook contract is missing.",
    },
    {
      id: "local",
      label: "Compose locally",
      detail: "The composition is product-specific and should stay outside the library.",
    },
    {
      id: "placeholder",
      label: "Record placeholder",
      detail: "The visual is not representable truthfully yet; preserve a replacement obligation.",
    },
  ];
  const record = useMutation({
    mutationFn: () =>
      sketchClient.addNote({
        target,
        scope: `asset-decision:${activeRegion || "page"}`,
        text: `decision=${decision}; reason=${reason.trim()}`,
        expectedContentHash: hash,
      }),
    onSuccess: async () => {
      setSavedDecision(`${activeRegion || "page"}:${decision}`);
      setReason("");
      await onSaved();
    },
  });
  return (
    <details
      open={open}
      onToggle={(event) => setOpen(event.currentTarget.open)}
      className="rounded-control border border-app-border bg-app-surface p-space-sm"
    >
      <summary className="cursor-pointer list-none font-semibold marker:hidden">
        <span className="flex flex-wrap items-center justify-between gap-space-xs">
          <span>Asset decision</span>
          <span className="text-label font-normal text-app-muted-foreground">
            Choose the smallest truthful reuse boundary
          </span>
        </span>
      </summary>
      {open && (
        <div className="grid gap-space-sm pt-space-sm">
          <p className="max-w-3xl text-sm text-app-muted-foreground">
            Use the evidence order from the campaign contract: conceptual fit, contract quality,
            tokens and defaults, accessibility, responsive behavior, tests, stories, hook semantics,
            and consumer impact. Record the decision before moving to review.
          </p>
          <label className="grid gap-space-2xs">
            Region
            <Select
              className="min-h-touch min-w-0 rounded-control border border-app-border bg-app-surface"
              value={activeRegion}
              options={regions.map((item) => ({ value: item, label: item }))}
              disabled={readOnly || regions.length === 0}
              onChange={(event) => setRegion(event.target.value)}
            />
          </label>
          <div className="grid gap-space-xs sm:grid-cols-2 lg:grid-cols-5">
            {decisions.map((item) => (
              <button
                key={item.id}
                type="button"
                aria-pressed={decision === item.id}
                disabled={readOnly}
                onClick={() => setDecision(item.id)}
                className={`min-h-touch rounded-control border p-space-xs text-left transition-colors ${decision === item.id ? "border-app-primary bg-app-primary/5" : "border-app-border bg-app-surface-muted hover:bg-app-surface"}`}
              >
                <span className="block text-sm font-semibold">{item.label}</span>
                <span className="mt-1 block text-label text-app-muted-foreground">
                  {item.detail}
                </span>
              </button>
            ))}
          </div>
          <label className="grid gap-space-2xs">
            Decision evidence / reason
            <textarea
              className="min-h-20 rounded-control border border-app-border bg-app-surface px-space-xs py-space-2xs text-sm"
              value={reason}
              disabled={readOnly}
              placeholder="Name the evidence that made this boundary truthful…"
              onChange={(event) => setReason(event.target.value)}
            />
          </label>
          <div className="flex flex-wrap items-center gap-space-sm">
            <Button
              disabled={readOnly || record.isPending || !activeRegion || reason.trim().length < 12}
              onClick={() => record.mutate()}
            >
              {record.isPending ? "Recording decision…" : "Record asset decision"}
            </Button>
            {savedDecision && (
              <span role="status" className="text-sm text-emerald-700 dark:text-emerald-300">
                Recorded {savedDecision.replace(":", " · ")}.
              </span>
            )}
          </div>
          {record.error && <Failure error={record.error} />}
        </div>
      )}
    </details>
  );
}

function ReferencePlaceholderEditor({
  target,
  hash,
  regions,
  readOnly,
  onSaved,
}: {
  target: { scenario: string; page: string };
  hash: string;
  regions: string[];
  readOnly: boolean;
  onSaved: () => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const [region, setRegion] = useState("");
  const [slug, setSlug] = useState("");
  const [intent, setIntent] = useState("");
  const [source, setSource] = useState("generated");
  const [provenance, setProvenance] = useState("");
  const [visualContract, setVisualContract] = useState("");
  const [replacement, setReplacement] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [uploaded, setUploaded] = useState<{ id: string; url: string } | null>(null);
  const [generated, setGenerated] = useState<{
    id: string;
    url: string;
    jobId: string;
    modelId: string;
    tier: string;
    warnings: string[];
  } | null>(null);
  const activeRegion = regions.includes(region) ? region : (regions[0] ?? "");
  const generation = useMutation({
    mutationFn: () => generateReferenceAsset(target, intent.trim()),
    onSuccess: (result) => {
      setGenerated(result);
      setProvenance(
        `image-tools text_to_image; job=${result.jobId}; model=${result.modelId}; tier=${result.tier}${result.warnings.length ? `; warnings=${result.warnings.join(" | ")}` : ""}`,
      );
    },
  });
  const effectiveProvenance =
    provenance.trim() ||
    (source === "generated"
      ? "generated reference not requested yet"
      : source === "uploaded"
        ? "uploaded reference; upload receipt recorded with the revision"
        : "existing approved asset");
  const note = [
    `reference-source=${source}`,
    `provenance=${effectiveProvenance}`,
    visualContract.trim() ? `visual-contract=${visualContract.trim()}` : "",
    replacement.trim() ? `replacement-owner=${replacement.trim()}` : "",
  ]
    .filter(Boolean)
    .join("; ");
  const placeholder = useMutation({
    mutationFn: async () => {
      const asset = source === "generated" && generated
        ? generated
        : file
          ? await uploadReferenceAsset(target, file)
          : null;
      const assetNote = asset ? `; reference-asset=${asset.id}; asset-url=${asset.url}` : "";
      const result = await sketchClient.placeholder({
        target,
        expectedContentHash: hash,
        region: activeRegion,
        placeholder: slug.trim(),
        intent: intent.trim(),
        note: note + assetNote,
      });
      return { result, asset };
    },
    onSuccess: async ({ asset }) => {
      setSlug("");
      setIntent("");
      setProvenance("");
      setVisualContract("");
      setReplacement("");
      setUploaded(asset ? { id: asset.id, url: asset.url } : null);
      setFile(null);
      setGenerated(null);
      await onSaved();
    },
  });

  return (
    <details
      open={open}
      onToggle={(event) => setOpen(event.currentTarget.open)}
      className="rounded-control border border-app-border bg-app-surface p-space-sm"
    >
      <summary className="cursor-pointer list-none font-semibold marker:hidden">
        <span className="flex flex-wrap items-center justify-between gap-space-xs">
          <span>Reference placeholder</span>
          <span className="rounded-full border border-amber-500/40 bg-amber-500/10 px-space-xs py-space-2xs text-label font-normal text-amber-700 dark:text-amber-300">
            Illustrative · replacement required
          </span>
        </span>
      </summary>
      {open && (
        <div className="grid gap-space-sm pt-space-sm">
          <p className="max-w-3xl text-sm text-app-muted-foreground">
            Use this for a complex visual region that cannot yet be represented truthfully by a
            library asset. Generate in image-tools / AI Gateway or choose an existing reference,
            then attach the source here. The bytes are retained in the experience workspace and the
            Sketch revision records provenance, limitations, and the replacement owner.
          </p>
          {regions.length === 0 ? (
            <p
              role="status"
              className="rounded-control border border-app-border bg-app-surface-muted p-space-sm text-sm text-app-muted-foreground"
            >
              Every current region already has an asset. Add a new empty region or revise an
              existing placement before recording a reference placeholder.
            </p>
          ) : (
            <>
              <fieldset className="grid gap-space-sm rounded-control border border-app-border bg-app-surface-muted p-space-sm sm:grid-cols-2">
                <legend className="px-space-2xs text-sm font-semibold">Source and target</legend>
                <label className="grid gap-space-2xs">
                  Region
                  <Select
                    className="min-h-touch min-w-0 rounded-control border border-app-border bg-app-surface"
                    value={activeRegion}
                    options={regions.map((item) => ({ value: item, label: item }))}
                    disabled={readOnly || regions.length === 0}
                    onChange={(event) => setRegion(event.target.value)}
                  />
                </label>
                <label className="grid gap-space-2xs">
                  Reference source
                  <Select
                    className="min-h-touch min-w-0 rounded-control border border-app-border bg-app-surface"
                    value={source}
                    disabled={readOnly}
                    options={[
                      { value: "generated", label: "Image tools / AI gateway" },
                      { value: "uploaded", label: "Uploaded reference" },
                      { value: "existing", label: "Existing approved asset" },
                    ]}
                    onChange={(event) => setSource(event.target.value)}
                  />
                </label>
              </fieldset>
              <fieldset className="grid gap-space-sm rounded-control border border-app-border bg-app-surface-muted p-space-sm">
                <legend className="px-space-2xs text-sm font-semibold">Visual brief</legend>
                <label className="grid gap-space-2xs">
                  Stable placeholder id
                  <Input
                    value={slug}
                    disabled={readOnly}
                    placeholder="operations-map-reference"
                    onChange={(event) => setSlug(event.target.value)}
                  />
                </label>
                <label className="grid gap-space-2xs">
                  What must the final region do?
                  <textarea
                    className="min-h-20 rounded-control border border-app-border bg-app-surface px-space-xs py-space-2xs text-sm"
                    value={intent}
                    disabled={readOnly}
                    placeholder="Describe the behavior and content the real renderer must provide…"
                    onChange={(event) => setIntent(event.target.value)}
                  />
                </label>
              </fieldset>
              <details className="rounded-control border border-app-border bg-app-surface-muted p-space-sm">
                <summary className="cursor-pointer font-semibold">
                  Handoff evidence and replacement obligation
                </summary>
                <div className="grid gap-space-sm pt-space-sm">
                  <p className="text-sm text-app-muted-foreground">
                    Add this when the source is ready to hand off. The save remains honest if
                    generation is unavailable, but the final implementation still needs an owner and
                    visual contract.
                  </p>
                  <label className="grid gap-space-2xs">
                    Provider receipt / upload id / unavailable reason
                    <Input
                      value={provenance}
                      disabled={readOnly}
                      placeholder="model/provider receipt, upload id, or why generation is unavailable"
                      onChange={(event) => setProvenance(event.target.value)}
                    />
                  </label>
                  {source === "generated" && !generated && !provenance.trim() && (
                    <p role="status" className="text-label text-amber-700 dark:text-amber-300">
                      Generate a reference here when image-tools is available, or record an honest
                      unavailable boundary instead of implying a model result.
                    </p>
                  )}
                  {source === "generated" && generated && (
                    <p role="status" className="text-label text-emerald-700 dark:text-emerald-300">
                      Generated with {generated.modelId} on {generated.tier}; job {generated.jobId}.
                      {generated.warnings.length ? ` ${generated.warnings.join(" ")}` : ""}
                    </p>
                  )}
                  <div className="grid gap-space-sm sm:grid-cols-2">
                    <label className="grid gap-space-2xs">
                      Replacement owner
                      <Input
                        value={replacement}
                        disabled={readOnly}
                        placeholder="renderer team or implementation path"
                        onChange={(event) => setReplacement(event.target.value)}
                      />
                    </label>
                    <label className="grid gap-space-2xs">
                      Visual contract
                      <Input
                        value={visualContract}
                        disabled={readOnly}
                        placeholder="crop, aspect ratio, palette, density, and known inaccuracies"
                        onChange={(event) => setVisualContract(event.target.value)}
                      />
                    </label>
                  </div>
                  <label className="grid gap-space-2xs">
                    Reference bytes (upload or attach an image-tools result)
                    <input
                      className="min-h-touch rounded-control border border-app-border bg-app-surface px-space-xs py-space-2xs text-sm"
                      type="file"
                      accept="image/*"
                      disabled={readOnly || placeholder.isPending}
                      onChange={(event) => setFile(event.target.files?.[0] ?? null)}
                    />
                    {file && (
                      <span className="text-label text-app-muted-foreground">
                        Ready: {file.name} · {Math.ceil(file.size / 1024)} KiB
                      </span>
                    )}
                  </label>
                </div>
              </details>
              {source === "generated" && (
                <div className="flex flex-wrap items-center gap-space-sm">
                  <Button
                    variant="secondary"
                    disabled={readOnly || generation.isPending || intent.trim().length < 20}
                    onClick={() => generation.mutate()}
                  >
                    {generation.isPending ? "Generating reference…" : generated ? "Regenerate reference" : "Generate reference with image-tools"}
                  </Button>
                  {generation.error && <Failure error={generation.error} />}
                </div>
              )}
              <Button
                disabled={
                  readOnly ||
                  placeholder.isPending ||
                  !activeRegion ||
                  !slug.trim() ||
                  intent.trim().length < 20 ||
                  (source === "generated" && !generated && !provenance.trim())
                }
                onClick={() => placeholder.mutate()}
              >
                {placeholder.isPending
                  ? file
                    ? "Uploading and recording…"
                    : "Recording placeholder…"
                  : "Save reference revision"}
              </Button>
              {placeholder.error && <Failure error={placeholder.error} />}
              {placeholder.isSuccess && (
                <p role="status">
                  Reference revision recorded with provenance, source bytes, and replacement
                  metadata. Later revisions remain in Sketch history.
                </p>
              )}
              {uploaded && (
                <img
                  src={uploaded.url}
                  alt="Recently uploaded reference revision"
                  className="max-h-48 max-w-full rounded-control border border-app-border object-contain"
                />
              )}
            </>
          )}
        </div>
      )}
    </details>
  );
}

type ReferenceRevision = {
  id: string;
  hash: string;
  createdAt: string;
  region: string;
  url?: string;
  source: string;
  contract: string;
  note: string;
};

function referenceRevisions(
  revisions: Array<{
    contentHash: string;
    createdAt: string;
    sketch?: {
      placements: Array<{ region: string; fills?: { placeholder?: string }; note: string }>;
    };
  }>,
): ReferenceRevision[] {
  const output: ReferenceRevision[] = [];
  const seen = new Set<string>();
  for (const revision of revisions) {
    for (const placement of revision.sketch?.placements ?? []) {
      if (!placement.fills?.placeholder) continue;
      const url = placement.note.match(/(?:^|; )asset-url=([^;]+)/)?.[1];
      const id = `${revision.contentHash}:${placement.region}:${placement.fills.placeholder}`;
      if (seen.has(id)) continue;
      seen.add(id);
      output.push({
        id,
        hash: revision.contentHash,
        createdAt: revision.createdAt,
        region: placement.region,
        url,
        source: placement.note.match(/(?:^|; )reference-source=([^;]+)/)?.[1] ?? "unknown",
        contract:
          placement.note.match(/(?:^|; )visual-contract=([^;]+)/)?.[1] ??
          "No visual contract recorded",
        note: placement.note,
      });
    }
  }
  return output;
}

function ReferenceRevisionReview({
  revisions,
  currentHash,
}: {
  revisions: Array<{
    contentHash: string;
    createdAt: string;
    sketch?: {
      placements: Array<{ region: string; fills?: { placeholder?: string }; note: string }>;
    };
  }>;
  currentHash: string;
}) {
  const [selected, setSelected] = useState<string[]>([]);
  const items = referenceRevisions(revisions);
  if (items.length === 0) return null;
  const toggle = (id: string) =>
    setSelected((previous) =>
      previous.includes(id)
        ? previous.filter((item) => item !== id)
        : previous.length < 2
          ? [...previous, id]
          : [previous[1]!, id],
    );
  const compared = items.filter((item) => selected.includes(item.id));
  const sourceUrl = (url: string) =>
    url.startsWith("http") ? url : `${API_BASE.replace(/\/$/, "")}${url}`;
  return (
    <section
      className="grid gap-space-sm rounded-control border border-app-border bg-app-surface p-space-sm"
      aria-label="Reference revision review"
    >
      <div className="flex flex-wrap items-start justify-between gap-space-xs">
        <div>
          <h2 className="font-semibold">Reference revisions</h2>
          <p className="text-sm text-app-muted-foreground">
            Inspect limitations before approval. Select up to two revisions to compare; Sketch
            history keeps every prior source.
          </p>
        </div>
        <span className="rounded-full border border-amber-500/40 bg-amber-500/10 px-space-xs py-space-2xs text-label text-amber-700 dark:text-amber-300">
          Illustrative · not production fidelity
        </span>
      </div>
      <div className="grid gap-space-xs sm:grid-cols-2 lg:grid-cols-3">
        {items.map((item) => (
          <button
            key={item.id}
            type="button"
            aria-pressed={selected.includes(item.id)}
            onClick={() => toggle(item.id)}
            className={`grid gap-space-2xs rounded-control border p-space-xs text-left ${selected.includes(item.id) ? "border-app-primary bg-app-primary/5" : "border-app-border bg-app-surface-muted"}`}
          >
            {item.url ? (
              <img
                src={sourceUrl(item.url)}
                alt={`Reference for ${item.region}`}
                className="aspect-video w-full rounded-control border border-app-border object-contain bg-black/5"
              />
            ) : (
              <div className="grid aspect-video place-items-center rounded-control border border-dashed border-amber-500/40 bg-amber-500/5 p-space-sm text-center text-label text-amber-700 dark:text-amber-300">
                No reference bytes attached. Review the recorded brief and provenance.
              </div>
            )}
            <span className="text-sm font-medium">
              {item.region} · {item.source}
            </span>
            <span className="text-label text-app-muted-foreground">
              {item.createdAt} · {item.hash.slice(0, 12)}
              {item.hash === currentHash ? " · current" : ""}
            </span>
            <span className="text-label text-amber-700 dark:text-amber-300">
              Limitations: {item.contract}
            </span>
          </button>
        ))}
      </div>
      {compared.length > 1 && (
        <div
          className="grid gap-space-sm border-t border-app-border pt-space-sm sm:grid-cols-2"
          aria-label="Reference comparison"
        >
          {compared.map((item) => (
            <figure key={item.id} className="grid gap-space-2xs">
              {item.url ? (
                <img
                  src={sourceUrl(item.url)}
                  alt={`Compared reference for ${item.region}`}
                  className="max-h-72 w-full rounded-control border border-app-border object-contain bg-black/5"
                />
              ) : (
                <div className="grid min-h-40 place-items-center rounded-control border border-dashed border-amber-500/40 bg-amber-500/5 p-space-sm text-center text-label text-amber-700 dark:text-amber-300">
                  Brief-only revision; no image bytes were available for this source.
                </div>
              )}
              <figcaption className="text-label text-app-muted-foreground">
                {item.createdAt} · {item.contract}
              </figcaption>
            </figure>
          ))}
        </div>
      )}
    </section>
  );
}

function DesignCanvas({
  target,
  hash,
  configured,
  readOnly,
  candidate,
  previewStates = [],
  unmappedRegions = [],
}: {
  target: { scenario: string; page: string };
  hash: string;
  configured: boolean;
  readOnly: boolean;
  candidate?: { scenario: string; designId: string; hash: string };
  previewStates?: string[];
  unmappedRegions?: string[];
}) {
  const { t, i18n } = useTranslation();
  const [width, setWidth] = useState("1440px");
  const [theme, setTheme] = useState("light");
  const [previewState, setPreviewState] = useState("");
  const direction = i18n.dir();
  const signature = JSON.stringify([hash, theme, i18n.language, direction, previewState]);
  const render = useMutation({
    mutationFn: async () => {
      const appearance = {
        missingLabel: t("design.canvasMissing"),
        failedLabel: t("design.canvasFailed"),
        theme,
        direction,
        previewState,
      };
      const result = candidate
        ? await sketchClient.renderCandidate({ candidate, ...appearance })
        : await sketchClient.renderSketch({ target, expectedContentHash: hash, ...appearance });
      // srcdoc inherits the UI base; runtime assets belong to the API origin.
      const document = new DOMParser().parseFromString(result.html, "text/html");
      const base = document.createElement("base");
      base.href = new URL(API_BASE, window.location.href).href.replace(/\/?$/, "/");
      document.head.prepend(base);
      return {
        signature,
        result,
        appearance,
        html: "<!doctype html>" + document.documentElement.outerHTML,
      };
    },
  });
  const current = render.data?.signature === signature ? render.data : undefined;
  const autoRenderSignature = useRef("");
  useEffect(() => {
    if (
      !configured ||
      readOnly ||
      current ||
      render.isPending ||
      autoRenderSignature.current === signature
    )
      return;
    autoRenderSignature.current = signature;
    render.mutate();
  }, [candidate, configured, current, readOnly, render, signature]);
  const frame = useRef<HTMLIFrameElement>(null);
  const viewport = useRef<HTMLDivElement>(null);
  const [previewScale, setPreviewScale] = useState(1);
  const viewportWidth = width === "390px" ? 390 : 1440;
  const viewportHeight = width === "390px" ? 844 : 900;
  useEffect(() => {
    const container = viewport.current;
    if (!container) return;
    const measure = () => {
      if (container.clientWidth > 0)
        setPreviewScale(Math.min(1, container.clientWidth / viewportWidth));
    };
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(measure);
    observer.observe(container);
    return () => observer.disconnect();
  }, [current?.result.renderHash, viewportWidth]);

  const [runtime, setRuntime] = useState<{ hash: string; error?: string }>();
  const renderHash = current?.result.renderHash;
  useEffect(() => {
    if (!renderHash) return;
    const receive = (event: MessageEvent) => {
      if (
        event.source !== frame.current?.contentWindow ||
        !event.data ||
        typeof event.data !== "object"
      )
        return;
      const data = event.data as { type?: string; sha256?: string; message?: string };
      if (data.sha256 !== renderHash) return;
      if (data.type === "preview-ready") setRuntime({ hash: renderHash });
      if (data.type === "preview-error")
        setRuntime({ hash: renderHash, error: data.message || t("design.canvasFailed") });
    };
    window.addEventListener("message", receive);
    return () => window.removeEventListener("message", receive);
  }, [renderHash, t]);
  return (
    <div id="design-canvas">
      <Card>
        <CardHeader>
          <CardTitle>{t("design.canvas")}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-space-sm">
          {!configured && (
            <div className="grid gap-space-2xs rounded-panel border border-dashed border-app-primary/40 bg-app-primary/5 p-space-sm">
              <p className="font-semibold">
                {candidate && unmappedRegions.length > 0
                  ? "Candidate needs port mapping"
                  : candidate
                    ? "Candidate needs a render contract"
                    : t("design.canvasEmptyTitle")}
              </p>
              <p className="text-sm text-app-muted-foreground">
                {candidate && unmappedRegions.length > 0
                  ? "Map " +
                    unmappedRegions.length +
                    " semantic region" +
                    (unmappedRegions.length === 1 ? "" : "s") +
                    " to published template ports before rendering."
                  : candidate
                    ? "Map this candidate to a published template, then render it here to inspect the real composition."
                    : t("design.canvasEmptyHelp")}
              </p>
              {candidate && unmappedRegions.length > 0 && (
                <a
                  href={`${designPath(target.scenario, target.page)}#candidate-port-mapping`}
                  className="mt-1 inline-flex min-h-touch w-fit items-center rounded-control border border-app-primary px-space-sm py-space-xs text-sm font-semibold text-app-primary hover:bg-app-primary/5"
                >
                  Review port mapping
                </a>
              )}
              {!candidate && (
                <a
                  href={`${designPath(target.scenario, target.page)}#design-brief`}
                  className="mt-1 inline-flex min-h-touch w-fit items-center rounded-control bg-app-primary px-space-sm py-space-xs text-sm font-semibold text-white hover:opacity-90"
                >
                  Create a candidate from intent
                </a>
              )}
            </div>
          )}
          {configured && !current && !render.isPending && (
            <p role="status">The current candidate is ready for a live composition preview.</p>
          )}
          <div className="flex flex-wrap items-center gap-space-sm">
            <label>
              {t("design.canvasWidth")}{" "}
              <Select
                id="design-canvas-width"
                aria-label={t("design.canvasWidth")}
                value={width}
                onChange={(e) => setWidth(e.target.value)}
                options={[
                  { value: "1440px", label: t("design.canvasDesktop") },
                  { value: "390px", label: t("design.canvasPhone") },
                ]}
              />
            </label>
            <label>
              {t("design.canvasTheme")}{" "}
              <Select
                id="design-canvas-theme"
                aria-label={t("design.canvasTheme")}
                value={theme}
                onChange={(e) => setTheme(e.target.value)}
                options={[
                  { value: "light", label: t("design.canvasLight") },
                  { value: "dark", label: t("design.canvasDark") },
                ]}
              />
            </label>
            {previewStates.length > 0 && (
              <label>
                {t("design.canvasState")}{" "}
                <Select
                  id="design-canvas-state"
                  aria-label={t("design.canvasState")}
                  value={previewState}
                  onChange={(e) => setPreviewState(e.target.value)}
                  options={[
                    { value: "", label: t("design.canvasInitialState") },
                    ...previewStates.map((state) => ({ value: state, label: state })),
                  ]}
                />
              </label>
            )}
            <Button
              data-action="render-design"
              disabled={!configured || readOnly || render.isPending}
              onClick={() => render.mutate()}
            >
              {render.isPending ? t("design.canvasRendering") : t("design.canvasRender")}
            </Button>
          </div>
          {render.error && <Failure error={render.error} />}
          {render.error &&
            /component .* not found|template .* not found|published template/i.test(
              render.error.message,
            ) && (
              <p
                role="status"
                className="rounded-control border border-amber-500/40 bg-amber-500/5 p-space-sm text-sm text-amber-700 dark:text-amber-300"
              >
                This revision cannot be rendered because its published template is not available in
                the current library registry. Keep the candidate as a reference or placeholder until
                the template owner publishes the exact version; it is not ready for implementation
                verification.
              </p>
            )}
          {render.data && !current && <p role="status">{t("design.canvasStale")}</p>}
          {current && (
            <>
              {runtime && runtime.hash === renderHash ? (
                runtime.error ? (
                  <div
                    role="alert"
                    className="grid gap-space-2xs rounded-control border border-amber-500/40 bg-amber-500/5 p-space-sm text-sm text-amber-800 dark:text-amber-200"
                  >
                    <p>{runtime.error}</p>
                    {/no template slot mapping/i.test(runtime.error) && (
                      <p>
                        This candidate needs a region-to-template-port decision before it can
                        render.
                        <a
                          href={`${designPath(target.scenario, target.page)}#candidate-port-mapping`}
                          className="ml-1 font-semibold underline underline-offset-2"
                        >
                          Review port mapping
                        </a>
                        .
                      </p>
                    )}
                  </div>
                ) : (
                  <p>{t("design.canvasEvidence")}</p>
                )
              ) : (
                <p role="status">{t("design.canvasRendering")}</p>
              )}
              {current.result.bundle?.gaps.map((gap) => (
                <p role="status" key={`${gap.region}:${gap.code}`}>
                  {gap.region}: {gap.message}
                </p>
              ))}
              {candidate && current.result.target && (
                <CandidateCapture
                  key={`${current.result.renderHash}:${width}`}
                  candidate={candidate}
                  renderHash={current.result.renderHash}
                  appearance={current.appearance}
                  width={width === "390px" ? 390 : 1440}
                  height={width === "390px" ? 844 : 900}
                />
              )}
              <div
                ref={viewport}
                className="relative min-w-0 max-w-full overflow-hidden"
                style={{ height: viewportHeight * previewScale }}
              >
                <iframe
                  ref={frame}
                  title={t("design.canvas")}
                  sandbox="allow-scripts allow-forms"
                  srcDoc={current.html}
                  style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: viewportWidth,
                    height: viewportHeight,
                    border: 0,
                    transform: `scale(${previewScale})`,
                    transformOrigin: "top left",
                  }}
                  data-viewport-width={viewportWidth}
                  data-viewport-height={viewportHeight}
                  data-render-hash={current.result.renderHash}
                  data-runtime-state={
                    runtime && runtime.hash === renderHash
                      ? runtime.error
                        ? "failed"
                        : "ready"
                      : "loading"
                  }
                />
              </div>
              {candidate && (
                <CandidateCritiques
                  key={`${candidate.hash}:${current.result.renderHash}`}
                  candidate={candidate}
                  renderHash={current.result.renderHash}
                  appearance={current.appearance}
                />
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function IntentProposals({
  target,
  hash,
  sketch,
  readOnly,
  onSaved,
}: {
  target: { scenario: string; page: string };
  hash: string;
  sketch?: Sketch;
  readOnly: boolean;
  onSaved: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const [intent, setIntent] = useState("");
  const [users, setUsers] = useState("");
  const [tasks, setTasks] = useState("");
  const [designSource, setDesignSource] = useState("");
  const [preserveRoutes, setPreserveRoutes] = useState(true);
  const [preserveBusinessBehavior, setPreserveBusinessBehavior] = useState(true);
  const [savedSignature, setSavedSignature] = useState("");
  const initialIntent = sketch?.intent as DesignIntent | undefined;
  useEffect(() => {
    setIntent(initialIntent?.intent ?? "");
    setUsers(initialIntent?.users.join("\n") ?? "");
    setTasks(initialIntent?.primaryTasks.join("\n") ?? "");
    setDesignSource(initialIntent?.constraints?.designSource ?? "");
    setPreserveRoutes(initialIntent?.constraints?.preserveRoutes ?? true);
    setPreserveBusinessBehavior(initialIntent?.constraints?.preserveBusinessBehavior ?? true);
    setSavedSignature(
      JSON.stringify([
        hash,
        initialIntent?.intent ?? "",
        initialIntent?.users.join("\n") ?? "",
        initialIntent?.primaryTasks.join("\n") ?? "",
        initialIntent?.constraints?.designSource ?? "",
        initialIntent?.constraints?.preserveRoutes ?? true,
        initialIntent?.constraints?.preserveBusinessBehavior ?? true,
      ]),
    );
  }, [hash, initialIntent]);
  const signature = JSON.stringify([
    hash,
    intent,
    users,
    tasks,
    designSource,
    preserveRoutes,
    preserveBusinessBehavior,
  ]);
  const formIntent = (): DesignIntent =>
    create(DesignIntentSchema, {
      intent: intent.trim(),
      users: users
        .split("\n")
        .map((v) => v.trim())
        .filter(Boolean),
      primaryTasks: tasks
        .split("\n")
        .map((v) => v.trim())
        .filter(Boolean),
      target: "react-vite",
      kit: "vrooli-default",
      viewports: ["phone", "desktop"],
      constraints: create(DesignConstraintsSchema, {
        designSource: designSource.trim(),
        preserveRoutes,
        preserveBusinessBehavior,
      }),
    });
  const saveBrief = useMutation({
    mutationFn: () =>
      sketchClient.putSketch({
        target,
        expectedContentHash: hash,
        sketch: {
          ...sketch,
          intent: formIntent(),
        } as Sketch,
      }),
    onSuccess: async () => {
      setSavedSignature(signature);
      await onSaved();
    },
  });
  const propose = useMutation({
    mutationFn: async () => ({
      signature,
      result: await sketchClient.proposeSketch({
        target,
        expectedContentHash: hash,
        candidateLimit: 3,
        intent: formIntent(),
      }),
    }),
  });
  const proposals = propose.data?.signature === signature ? propose.data.result : undefined;
  const preview = useMutation({
    mutationFn: async (index: number) => {
      const selected = proposals?.candidates[index];
      if (!selected?.sketch || !proposals) throw new Error(t("design.conflict"));
      const saved = await sketchClient.saveCandidate({
        target,
        designId: `${target.page}-intent`,
        expectedContentHash: proposals.contentHash,
        sketch: selected.sketch,
      });
      return { saved, signature };
    },
  });
  const choose = useMutation({
    mutationFn: async (index: number) => {
      const candidate = proposals?.candidates[index];
      if (!candidate?.sketch || !proposals) throw new Error(t("design.conflict"));
      const saved = await sketchClient.saveCandidate({
        target,
        designId: `${target.page}-intent`,
        expectedContentHash: proposals.contentHash,
        sketch: candidate.sketch,
      });
      if (!saved.sketch || saved.baseHash !== proposals.contentHash)
        throw new Error(t("design.conflict"));
      return sketchClient.putSketch({
        target,
        expectedContentHash: saved.baseHash,
        sketch: saved.sketch,
      });
    },
    onSuccess: onSaved,
  });
  return (
    <details
      id="design-brief"
      open
      className="scroll-mt-24 rounded-control border border-app-border p-space-sm"
    >
      <summary>{t("design.proposeTitle")}</summary>
      <div className="grid gap-space-sm pt-space-sm">
        <p>{t("design.proposeHelp")}</p>
        <p className="rounded-control border border-app-border bg-app-surface-muted p-space-xs text-sm text-app-muted-foreground">
          Complete the three required fields below before generating proposals. Existing canvas
          content can be reviewed independently, but it does not replace this brief.
        </p>
        <label>
          {t("design.proposeIntent")} <span aria-hidden="true">(required)</span>
          <textarea
            className="block w-full"
            aria-label={t("design.proposeIntent")}
            aria-required="true"
            placeholder="What should this experience help someone accomplish?"
            value={intent}
            onChange={(e) => setIntent(e.target.value)}
          />
        </label>
        <label>
          {t("design.proposeUsers")} <span aria-hidden="true">(required)</span>
          <textarea
            className="block w-full"
            aria-label={t("design.proposeUsers")}
            aria-required="true"
            placeholder="One audience per line"
            value={users}
            onChange={(e) => setUsers(e.target.value)}
          />
        </label>
        <label>
          {t("design.proposeTasks")} <span aria-hidden="true">(required)</span>
          <textarea
            className="block w-full"
            aria-label={t("design.proposeTasks")}
            aria-required="true"
            placeholder="One primary task per line"
            value={tasks}
            onChange={(e) => setTasks(e.target.value)}
          />
        </label>
        <label>
          {t("design.proposeDesignSource")}
          <input
            className="block w-full"
            value={designSource}
            onChange={(e) => setDesignSource(e.target.value)}
          />
        </label>
        <p>{t("design.proposeSourceHelp")}</p>
        <label>
          <input
            type="checkbox"
            checked={preserveRoutes}
            onChange={(e) => setPreserveRoutes(e.target.checked)}
          />{" "}
          {t("design.proposePreserveRoutes")}
        </label>
        <label>
          <input
            type="checkbox"
            checked={preserveBusinessBehavior}
            onChange={(e) => setPreserveBusinessBehavior(e.target.checked)}
          />{" "}
          {t("design.proposePreserveBehavior")}
        </label>
        <Button
          variant="secondary"
          disabled={
            readOnly ||
            saveBrief.isPending ||
            !intent.trim() ||
            !users.trim() ||
            !tasks.trim() ||
            signature === savedSignature
          }
          onClick={() => saveBrief.mutate()}
        >
          {saveBrief.isPending ? "Saving brief…" : "Save brief revision"}
        </Button>
        <Button
          disabled={
            readOnly || !intent.trim() || !users.trim() || !tasks.trim() || propose.isPending
          }
          onClick={() => propose.mutate()}
        >
          {propose.isPending ? t("design.loading") : t("design.proposeAction")}
        </Button>
        {saveBrief.error && <Failure error={saveBrief.error} />}
        {saveBrief.isSuccess && signature === savedSignature && (
          <p role="status" className="text-sm text-emerald-700 dark:text-emerald-300">
            Brief revision saved. It will be restored when this workspace is reopened.
          </p>
        )}
        {!readOnly && !intent.trim() && !users.trim() && !tasks.trim() && (
          <p role="status" className="text-sm text-app-muted-foreground">
            Add an intent, at least one audience, and at least one primary task to enable proposal
            generation.
          </p>
        )}
        {propose.error && <Failure error={propose.error} />}
        {choose.error && <Failure error={choose.error} />}
        {preview.error && <Failure error={preview.error} />}
        {preview.data?.signature === signature && preview.data.saved.candidate && (
          <CandidatePreview
            key={preview.data.saved.candidate.hash}
            target={target}
            initial={preview.data.saved}
            onSelected={onSaved}
          />
        )}
        {proposals && (
          <>
            <p>{t("design.proposeLexical")}</p>
            {proposals.designSource && (
              <details>
                <summary>
                  {t("design.proposeSourceSnapshot", { path: proposals.designSource.path })}
                </summary>
                <p className="break-all">
                  {t("design.proposeSourceHash", { hash: proposals.designSource.contentHash })}
                </p>
                <pre className="max-h-96 overflow-auto whitespace-pre-wrap">
                  {proposals.designSource.content}
                </pre>
              </details>
            )}
            {proposals.diagnostics.map((message) => (
              <p role="status" key={message}>
                {message}
              </p>
            ))}
            {proposals.candidates.map((candidate, index) => (
              <Card key={`${candidate.sketch?.template?.asset}:${index}`}>
                <CardHeader>
                  <CardTitle>{candidate.title}</CardTitle>
                </CardHeader>
                <CardContent className="grid gap-space-sm">
                  <p>
                    {candidate.sketch?.template?.asset} · {candidate.sketch?.template?.version}
                  </p>
                  <ul>
                    {candidate.sketch?.regions.map((region) => (
                      <li key={region.id}>{region.id}</li>
                    ))}
                  </ul>
                  <ul>
                    {candidate.obligations.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                  {candidate.sketch?.unplaced.map((item) => (
                    <p key={`${item.region}:${item.was}`}>
                      {item.was}: {item.reason}
                    </p>
                  ))}
                  <Button
                    disabled={readOnly || preview.isPending}
                    onClick={() => preview.mutate(index)}
                  >
                    {t("design.proposePreview")}
                  </Button>
                  {!candidate.sketch?.render && (
                    <p role="status" className="text-sm text-app-muted-foreground">
                      Preview will open the candidate review first. Configure its template ports
                      before a live composition can render.
                    </p>
                  )}
                  <Button
                    disabled={readOnly || choose.isPending}
                    onClick={() => choose.mutate(index)}
                  >
                    {t("design.proposeChoose")}
                  </Button>
                </CardContent>
              </Card>
            ))}
          </>
        )}
      </div>
    </details>
  );
}

function CandidatePreview({
  target,
  initial,
  onSelected,
}: {
  target: { scenario: string; page: string };
  initial: CandidateResponse;
  onSelected: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const [current, setCurrent] = useState(initial);
  const [selections, setSelections] = useState<Record<string, string>>({});
  const ports = current.sketch?.render?.regions ?? [];
  // Only top-level render regions are remappable template ports. Nested
  // regions inherit their parent's slot and must stay out of this control.
  const topLevelPorts = ports.filter((port) => !port.parent);
  const authoredIds = new Set([
    ...current.declaredRegions.map((region) => region.id),
    ...(current.sketch?.regions
      .filter((region) => region.origin !== "template")
      .map((region) => region.id) ?? []),
  ]);
  const authored = [...authoredIds];
  const select = useMutation({
    mutationFn: () =>
      sketchClient.putSketch({
        target,
        expectedContentHash: current.baseHash,
        sketch: current.sketch,
      }),
    onSuccess: onSelected,
  });
  const value = (region: string) => {
    if (selections[region] !== undefined) return selections[region];
    const port = topLevelPorts.find((candidate) => candidate.id === region);
    return port?.templateRegion || port?.id || "";
  };
  const unmappedRegions = ports.length > 0 ? authored.filter((region) => !value(region)) : [];
  const map = useMutation({
    mutationFn: () =>
      sketchClient.mapCandidateRegions({
        candidate: current.candidate,
        mappings: authored
          .filter(
            (region) =>
              value(region) && (selections[region] !== undefined || value(region) !== region),
          )
          .map((region) => ({ region, templateRegion: value(region) })),
      }),
    onSuccess: setCurrent,
  });
  return (
    <div
      className="grid gap-space-sm"
      data-candidate-hash={current.candidate?.hash}
      data-candidate-parent={current.parentHash || undefined}
    >
      {current.refinement && (
        <div>
          <p>
            {t("design.refinementRound", {
              round: current.refinement.round,
              budget: current.refinement.budget,
            })}
          </p>
          <p>{current.refinement.reason}</p>
          <p>
            {t("design.refinementScope", { regions: current.refinement.changedRegions.join(", ") })}
          </p>
          {current.refinement.broaderChangeReason && (
            <p>{current.refinement.broaderChangeReason}</p>
          )}
        </div>
      )}

      {authored.length > 0 && (
        <section
          id="candidate-port-mapping"
          className="grid gap-space-sm rounded-control border border-app-border bg-app-surface-muted p-space-sm"
        >
          <p>{t("design.portMappingHelp")}</p>
          {authored.map((region) => (
            <label key={region}>
              {t("design.portMappingRegion", { region })}
              <Select
                className="block w-full"
                value={value(region)}
                options={[
                  { value: "", label: t("design.portMappingChoose") },
                  ...topLevelPorts.map((port) => ({
                    value: port.templateRegion || port.id,
                    label: port.templateRegion || port.id,
                  })),
                ]}
                onChange={(event) => setSelections({ ...selections, [region]: event.target.value })}
              />
            </label>
          ))}
          <Button
            disabled={map.isPending || !authored.some((region) => value(region))}
            onClick={() => map.mutate()}
          >
            {t("design.portMappingApply")}
          </Button>
          {map.error && <Failure error={map.error} />}
        </section>
      )}
      <CandidateAssetPicker current={current} onSelected={setCurrent} />
      {current.candidate && (
        <DesignCanvas
          key={current.candidate.hash}
          target={target}
          hash={current.candidate.hash}
          candidate={current.candidate}
          previewStates={declaredPreviewStates(current)}
          configured={Boolean(current.sketch?.render) && unmappedRegions.length === 0}
          unmappedRegions={unmappedRegions}
          readOnly={false}
        />
      )}
      <Button disabled={select.isPending || !current.sketch} onClick={() => select.mutate()}>
        {t("design.proposeChoose")}
      </Button>
      {select.error && <Failure error={select.error} />}
    </div>
  );
}

function declaredPreviewStates(current: CandidateResponse): string[] {
  const graph = current.sketch?.render?.bindings?.$preview;
  if (!graph || typeof graph !== "object" || Array.isArray(graph)) return [];
  const states = graph.states;
  return states && typeof states === "object" && !Array.isArray(states) ? Object.keys(states) : [];
}

function CandidateAssetPicker({
  current,
  onSelected,
}: {
  current: CandidateResponse;
  onSelected: (next: CandidateResponse) => void;
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [region, setRegion] = useState("");
  const [assetId, setAssetId] = useState("");
  const [storyId, setStoryId] = useState("");
  const emptyRegions = (current.sketch?.render?.regions ?? []).filter(
    (r) => !current.sketch?.placements.some((p) => p.region === r.id),
  );
  const activeRegion = emptyRegions.some((r) => r.id === region)
    ? region
    : (emptyRegions[0]?.id ?? "");
  const search = useQuery({
    queryKey: ["candidate-filler-search", query],
    enabled: query.trim().length > 1,
    queryFn: ({ signal }) => searchDesignAssets(query, "", signal),
  });
  const assets =
    search.data?.results.filter(
      (asset) =>
        asset.availabilityState === "built" && asset.version && asset.kind !== "page-template",
    ) ?? [];
  const asset = assets.find((item) => item.catalogId === assetId);
  const stories = useQuery({
    queryKey: ["candidate-filler-stories", asset?.catalogId, asset?.version],
    enabled: Boolean(asset),
    queryFn: async () => {
      if (!asset) return [];
      const result = await listComponentStories({
        componentId: asset.catalogId,
        version: asset.version,
      });
      return result.stories.flatMap((row) => {
        const values: unknown = JSON.parse(row.storiesJson);
        if (!Array.isArray(values)) throw new Error(t("design.failed"));
        const items: unknown[] = values;
        return items.filter((value): value is { id: string; name: string } =>
          Boolean(
            value &&
              typeof value === "object" &&
              "id" in value &&
              "name" in value &&
              typeof value.id === "string" &&
              typeof value.name === "string",
          ),
        );
      });
    },
  });
  const place = useMutation({
    mutationFn: () =>
      sketchClient.placeCandidateAsset({
        candidate: current.candidate,
        region: activeRegion,
        asset: { asset: asset?.catalogId ?? "", version: asset?.version ?? "" },
        story: storyId,
      }),
    onSuccess: (next) => {
      onSelected(next);
      setAssetId("");
      setStoryId("");
    },
  });
  if (!emptyRegions.length) return null;
  return (
    <details className="rounded-control border border-app-border p-space-sm">
      <summary>{t("design.fillerTitle")}</summary>
      <div className="grid gap-space-sm pt-space-sm">
        <label>
          {t("design.fillerRegion")}
          <Select
            className="block w-full"
            value={activeRegion}
            options={emptyRegions.map((r) => ({ value: r.id, label: r.id }))}
            onChange={(e) => setRegion(e.target.value)}
          />
        </label>
        <label>
          {t("design.fillerSearch")}
          <Input
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setAssetId("");
              setStoryId("");
            }}
          />
        </label>
        <label>
          {t("design.fillerAsset")}
          <Select
            className="block w-full"
            value={assetId}
            options={[
              { value: "", label: t("design.fillerChoose") },
              ...assets.map((item) => ({
                value: item.catalogId,
                label: `${item.name} · ${item.version}`,
              })),
            ]}
            onChange={(e) => {
              setAssetId(e.target.value);
              setStoryId("");
            }}
          />
        </label>
        <label>
          {t("design.fillerStory")}
          <Select
            className="block w-full"
            value={storyId}
            options={[
              { value: "", label: t("design.fillerChoose") },
              ...(stories.data?.map((story) => ({ value: story.id, label: story.name })) ?? []),
            ]}
            onChange={(e) => setStoryId(e.target.value)}
          />
        </label>
        <Button
          disabled={
            !asset ||
            !activeRegion ||
            !stories.data?.some((story) => story.id === storyId) ||
            place.isPending
          }
          onClick={() => place.mutate()}
        >
          {t("design.fillerPlace")}
        </Button>
        {search.error && <Failure error={search.error} />}
        {stories.error && <Failure error={stories.error} />}
        {place.error && <Failure error={place.error} />}
      </div>
    </details>
  );
}

function SavedCandidates({
  target,
  onSelected,
}: {
  target: { scenario: string; page: string };
  onSelected: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<CandidateResponse>();
  const [compared, setCompared] = useState<CandidateResponse[]>([]);
  const compare = useMutation({
    mutationFn: async (candidate: { scenario: string; designId: string; hash: string }) => {
      const result = await sketchClient.getCandidate(candidate);
      if (
        result.candidate?.scenario !== candidate.scenario ||
        result.candidate.designId !== candidate.designId ||
        result.candidate.hash !== candidate.hash ||
        result.page !== target.page
      )
        throw new Error(t("design.compareMismatch"));
      return result;
    },
    onSuccess: (result) =>
      setCompared((previous) =>
        previous.some((item) => item.candidate?.hash === result.candidate?.hash) ||
        previous.length >= 2
          ? previous
          : [...previous, result],
      ),
  });

  const inventory = useQuery({
    queryKey: ["saved-design-candidates", target.scenario, target.page],
    queryFn: () => sketchClient.listCandidates(target),
  });
  const open = useMutation({
    mutationFn: (candidate: NonNullable<CandidateResponse["candidate"]>) =>
      sketchClient.getCandidate(candidate),
    onSuccess: setSelected,
  });
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("design.savedCandidates")}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-space-sm">
        <Button disabled={inventory.isFetching} onClick={() => void inventory.refetch()}>
          {t("design.reload")}
        </Button>
        {inventory.isPending && <p role="status">{t("design.loading")}</p>}
        {inventory.error && <Failure error={inventory.error} />}
        {inventory.data?.candidates.length === 0 && <p>{t("design.noSavedCandidates")}</p>}
        {inventory.data?.candidates.map(
          (row) =>
            row.candidate && (
              <div key={`${row.candidate.designId}/${row.candidate.hash}`}>
                <Button
                  data-open-candidate={row.candidate.hash}
                  disabled={open.isPending}
                  onClick={() => {
                    if (row.candidate) open.mutate(row.candidate);
                  }}
                >
                  {t("design.openCandidate", {
                    design: row.candidate.designId,
                    revision: row.candidate.hash.slice(0, 12),
                  })}
                </Button>
                <Button
                  data-compare-candidate={row.candidate.hash}
                  aria-pressed={compared.some(
                    (item) => item.candidate?.hash === row.candidate?.hash,
                  )}
                  disabled={
                    compare.isPending ||
                    (compared.length >= 2 &&
                      !compared.some((item) => item.candidate?.hash === row.candidate?.hash))
                  }
                  onClick={() => {
                    const candidate = row.candidate;
                    if (!candidate) return;
                    if (compared.some((item) => item.candidate?.hash === candidate.hash))
                      setCompared((items) =>
                        items.filter((item) => item.candidate?.hash !== candidate.hash),
                      );
                    else compare.mutate(candidate);
                  }}
                >
                  {t("design.compareCandidate", { revision: row.candidate.hash.slice(0, 12) })}
                </Button>
                <p>
                  {row.templateAsset} {row.templateVersion}
                </p>
                {row.parentHash && (
                  <p>{t("design.candidateParent", { revision: row.parentHash.slice(0, 12) })}</p>
                )}
              </div>
            ),
        )}
        {compare.error && <Failure error={compare.error} />}
        {compared.length > 0 && (
          <section aria-label={t("design.compareTitle")} className="grid min-w-0 gap-space-sm">
            <h3>{t("design.compareTitle")}</h3>
            <p>
              {t(
                compared.length === 1
                  ? "design.compareChooseSecond"
                  : "design.comparePreservesSelection",
              )}
            </p>
            <div className="grid min-w-0 gap-space-sm lg:grid-cols-2">
              {compared.map(
                (item) =>
                  item.candidate && (
                    <article
                      key={item.candidate.hash}
                      data-comparison-candidate={item.candidate.hash}
                      className="min-w-0"
                    >
                      <h4>
                        {item.candidate.designId} · {item.candidate.hash.slice(0, 12)}
                      </h4>
                      <DesignCanvas
                        target={target}
                        hash={item.candidate.hash}
                        candidate={item.candidate}
                        configured={Boolean(item.sketch?.render)}
                        previewStates={declaredPreviewStates(item)}
                        readOnly={false}
                      />
                    </article>
                  ),
              )}
            </div>
          </section>
        )}
        {open.error && <Failure error={open.error} />}
        {selected && (
          <CandidatePreview
            key={`${target.scenario}/${target.page}/${selected.candidate?.hash}`}
            target={target}
            initial={selected}
            onSelected={onSelected}
          />
        )}
      </CardContent>
    </Card>
  );
}

type CaptureIntent = { key: string; id?: string; previousId?: string };

function CandidateCapture({
  candidate,
  renderHash,
  appearance,
  width,
  height,
}: {
  candidate: { scenario: string; designId: string; hash: string };
  renderHash: string;
  appearance: {
    missingLabel: string;
    failedLabel: string;
    theme: string;
    direction: string;
    previewState?: string;
  };
  width: number;
  height: number;
}) {
  const { t } = useTranslation();
  const storageKey = `rcl:capture:${renderHash}:${width}x${height}`;
  const [identity, setIdentity] = useState<CaptureIntent>(() => {
    try {
      const stored = sessionStorage.getItem(storageKey);
      if (stored) {
        const value = JSON.parse(stored) as { key?: unknown; id?: unknown; previousId?: unknown };
        if (typeof value.key === "string" && value.key.length > 0 && value.key.length <= 200) {
          return {
            key: value.key,
            id: typeof value.id === "string" ? value.id : undefined,
            previousId: typeof value.previousId === "string" ? value.previousId : undefined,
          };
        }
      }
    } catch {
      /* Capture submission reports unavailable browser storage before dispatch. */
    }
    return { key: crypto.randomUUID() };
  });
  const [operation, setOperation] = useState<CaptureOperation>();
  const restored = useQuery({
    queryKey: ["capture-operation", identity.id],
    enabled: Boolean(identity.id),
    queryFn: () => sketchClient.getCapture({ id: identity.id ?? "" }),
    retry: false,
  });
  const current = operation ?? restored.data;
  const captured = (next: CaptureOperation, intent: CaptureIntent = identity) => {
    setOperation(next);
    const saved = { ...intent, id: next.id };
    sessionStorage.setItem(storageKey, JSON.stringify(saved));
    setIdentity(saved);
  };
  const start = useMutation({
    mutationFn: (intent: CaptureIntent) => {
      // Persist the new intent before dispatch, including its previous operation.
      sessionStorage.setItem(storageKey, JSON.stringify(intent));
      setIdentity(intent);
      setOperation(undefined);
      if (intent.previousId)
        return sketchClient.retryCapture({ id: intent.previousId, idempotencyKey: intent.key });
      return sketchClient.captureCandidate({
        render: { candidate, ...appearance },
        expectedRenderHash: renderHash,
        idempotencyKey: intent.key,
        width,
        height,
      });
    },
    onSuccess: (next, intent) => captured(next, intent),
  });
  const attach = useMutation({
    mutationFn: () => sketchClient.attachCapture({ id: current?.id ?? "" }),
    onSuccess: (next) => captured(next),
  });
  const cancel = useMutation({
    mutationFn: () => sketchClient.cancelCapture({ id: current?.id ?? "" }),
    onSuccess: (next) => captured(next),
    onError: () => {
      void restored.refetch().then((result) => {
        if (result.data) setOperation(result.data);
      });
    },
  });
  const stateLabel = (() => {
    switch (current?.state) {
      case "completed":
        return t("design.captureCompleted");
      case "failed":
        return t("design.captureFailed");
      case "cancelled":
        return t("design.captureCancelled");
      case "dispatch_unknown":
        return t("design.captureUnknown");
      case "cancel_requested":
        return t("design.captureCancelRequested");
      default:
        return t("design.captureRunning");
    }
  })();
  return (
    <section className="grid gap-space-sm" aria-label={t("design.captureTitle")}>
      <p>{t("design.captureViewport", { width, height })}</p>
      {!current && (
        <Button
          disabled={start.isPending || restored.isFetching}
          onClick={() => start.mutate(identity)}
        >
          {t("design.captureStart")}
        </Button>
      )}
      {start.isPending && <p role="status">{t("design.captureStarting")}</p>}
      {start.error && <Failure error={start.error} conflictText={t("design.captureStale")} />}
      {restored.error && <Failure error={restored.error} />}
      {current && (
        <>
          <p role="status">{stateLabel}</p>
          {current.detail && <p>{current.detail}</p>}
          {["running", "cancel_requested"].includes(current.state) && (
            <Button disabled={attach.isPending || cancel.isPending} onClick={() => attach.mutate()}>
              {t("design.captureAttach")}
            </Button>
          )}
          {["running", "cancel_requested"].includes(current.state) && (
            <Button disabled={cancel.isPending || attach.isPending} onClick={() => cancel.mutate()}>
              {t(
                current.state === "cancel_requested"
                  ? "design.captureCancelRetry"
                  : "design.captureCancel",
              )}
            </Button>
          )}
          {cancel.error && <Failure error={cancel.error} />}
          {attach.error && <Failure error={attach.error} />}
          {["completed", "failed", "cancelled"].includes(current.state) && (
            <Button
              disabled={start.isPending}
              onClick={() => start.mutate({ key: crypto.randomUUID(), previousId: current.id })}
            >
              {t("design.captureAgain")}
            </Button>
          )}
          {current.previousId && <PreviousCaptureReceipt id={current.previousId} />}
          {current.state === "completed" && <p>{t("design.captureEvidenceOnly")}</p>}
          <details className="break-all">
            <summary>{t("design.captureReceipt")}</summary>
            <p>{t("design.captureOperation", { id: current.id })}</p>
            {current.producerId && <p>{t("design.captureProducer", { id: current.producerId })}</p>}
            {current.artifacts.map((artifact) => (
              <div key={artifact.reference}>
                <code>{artifact.reference}</code>
                {artifact.kind === "screenshot" && current.state === "completed" && (
                  <CaptureScreenshotView
                    operationId={current.id}
                    reference={artifact.reference}
                    viewportWidth={current.width}
                    viewportHeight={current.height}
                  />
                )}
                {artifact.evidence && (
                  <p>{t("design.captureRegions", { count: artifact.evidence.regions.length })}</p>
                )}
              </div>
            ))}
          </details>
        </>
      )}
    </section>
  );
}

function CaptureScreenshotView({
  operationId,
  reference,
  viewportWidth,
  viewportHeight,
}: {
  operationId: string;
  reference: string;
  viewportWidth: number;
  viewportHeight: number;
}) {
  const { t } = useTranslation();
  const [imageState, setImageState] = useState<"loading" | "ready" | "failed">("loading");
  const [attempt, setAttempt] = useState(0);
  const screenshot = useMutation({
    mutationFn: () => sketchClient.getCaptureScreenshot({ id: operationId, reference }),
  });
  return (
    <div className="grid gap-space-sm">
      <Button
        disabled={screenshot.isPending}
        onClick={() => {
          setImageState("loading");
          setAttempt((value) => value + 1);
          screenshot.mutate();
        }}
      >
        {t("design.captureViewScreenshot")}
      </Button>
      {screenshot.error && <Failure error={screenshot.error} />}
      {screenshot.data && (
        <figure>
          <img
            key={attempt}
            src={screenshot.data.url}
            width={viewportWidth}
            height={viewportHeight}
            alt={t("design.captureScreenshotAlt")}
            className="h-auto max-w-full"
            referrerPolicy="no-referrer"
            onLoad={() => setImageState("ready")}
            onError={() => setImageState("failed")}
          />
          {imageState === "loading" && <p role="status">{t("design.captureImageLoading")}</p>}
          {imageState === "failed" && <p role="alert">{t("design.captureImageUnavailable")}</p>}
          <figcaption>{t("design.captureImageOwner")}</figcaption>
        </figure>
      )}
    </div>
  );
}

function PreviousCaptureReceipt({ id }: { id: string }) {
  const { t } = useTranslation();
  const previous = useMutation({ mutationFn: () => sketchClient.getCapture({ id }) });
  return (
    <details className="break-all">
      <summary>{t("design.capturePrevious", { id })}</summary>
      <Button disabled={previous.isPending} onClick={() => previous.mutate()}>
        {t("design.capturePreviousRead")}
      </Button>
      {previous.error && <Failure error={previous.error} />}
      {previous.data && (
        <>
          <p>{t("design.captureOperation", { id: previous.data.id })}</p>
          <p>{t("design.captureProducer", { id: previous.data.producerId })}</p>
          {previous.data.detail && <p>{previous.data.detail}</p>}
          {previous.data.artifacts.map((artifact) => (
            <div key={artifact.reference}>
              <code>{artifact.reference}</code>
              {artifact.kind === "screenshot" && (
                <CaptureScreenshotView
                  operationId={previous.data.id}
                  reference={artifact.reference}
                  viewportWidth={previous.data.width}
                  viewportHeight={previous.data.height}
                />
              )}
            </div>
          ))}
        </>
      )}
    </details>
  );
}

const critiqueDimensionKeys = {
  task_clarity: "design.reviewTaskClarity",
  hierarchy: "design.reviewHierarchy",
  navigation_continuity: "design.reviewNavigation",
  responsive_layout: "design.reviewResponsive",
  state_recovery: "design.reviewRecovery",
  accessibility: "design.reviewAccessibility",
  terminology_i18n: "design.reviewLanguage",
  visual_consistency: "design.reviewConsistency",
} as const;

function CandidateCritiques({
  candidate,
  renderHash,
  appearance,
}: {
  candidate: { scenario: string; designId: string; hash: string };
  renderHash: string;
  appearance: {
    missingLabel: string;
    failedLabel: string;
    theme: string;
    direction: string;
    previewState: string;
  };
}) {
  const { t } = useTranslation();
  const target = {
    scenario: candidate.scenario,
    designId: candidate.designId,
    revision: candidate.hash,
    renderHash,
  };
  const [selected, setSelected] = useState<{ id: string; hash: string }>();
  const rubric = useQuery({
    queryKey: ["critique-rubric"],
    queryFn: () => sketchClient.getCritiqueRubric({}),
    retry: false,
  });
  const reviews = useInfiniteQuery({
    queryKey: ["candidate-critiques", target],
    initialPageParam: "",
    queryFn: ({ pageParam }) => sketchClient.listCritiques({ target, beforeId: pageParam }),
    getNextPageParam: (page) => page.nextBeforeId || undefined,
    retry: false,
  });
  const detail = useQuery({
    queryKey: ["critique-record", selected?.id, selected?.hash, target],
    enabled: Boolean(selected),
    retry: false,
    queryFn: async () => {
      const record = await sketchClient.getCritique({ id: selected?.id ?? "" });
      const actual = record.review?.target;
      if (
        record.hash !== selected?.hash ||
        actual?.scenario !== target.scenario ||
        actual.designId !== target.designId ||
        actual.revision !== target.revision ||
        actual.renderHash !== target.renderHash
      )
        throw new Error(t("design.reviewMismatch"));
      return record;
    },
  });
  const severityLabel = (value: string) => {
    const keys = {
      critical: "design.reviewCritical",
      major: "design.reviewMajor",
      minor: "design.reviewMinor",
      informational: "design.reviewInformational",
    } as const;
    return value in keys ? t(keys[value as keyof typeof keys]) : value;
  };
  const dimensionLabel = (value: string) =>
    value in critiqueDimensionKeys
      ? t(critiqueDimensionKeys[value as keyof typeof critiqueDimensionKeys])
      : value;
  const records = reviews.data?.pages.flatMap((page) => page.reviews) ?? [];
  return (
    <section className="grid gap-space-sm" aria-label={t("design.reviewTitle")}>
      <h3>{t("design.reviewTitle")}</h3>
      <p>{t("design.reviewNotAcceptance")}</p>
      <CandidateAcceptanceCheck
        key={selected?.id ?? "none"}
        candidate={candidate}
        renderHash={renderHash}
        appearance={appearance}
        critiqueId={selected?.id}
      />
      {rubric.data && !rubric.data.calibrated && <p>{t("design.reviewUncalibrated")}</p>}
      {rubric.error && (
        <Failure
          error={rubric.error}
          retry={() => {
            void rubric.refetch();
          }}
        />
      )}
      <Button
        disabled={reviews.isFetching}
        onClick={() => {
          void reviews.refetch();
        }}
      >
        {t("design.reviewRefresh")}
      </Button>
      {reviews.error && (
        <Failure
          error={reviews.error}
          retry={() => {
            void reviews.refetch();
          }}
        />
      )}
      {reviews.isPending && <p role="status">{t("design.reviewLoading")}</p>}
      {reviews.data && records.length === 0 && <p>{t("design.reviewEmpty")}</p>}
      {records.map((record) => (
        <div key={record.id} className="grid gap-space-sm">
          <Button
            data-review-id={record.id}
            onClick={() => setSelected({ id: record.id, hash: record.hash })}
          >
            {t("design.reviewOpen", { critic: record.critic?.id ?? "", date: record.recordedAt })}
          </Button>
          <p>
            {t(
              record.assessment?.visualFloorMet
                ? "design.reviewFloorMet"
                : "design.reviewFloorFailed",
            )}
          </p>
        </div>
      ))}
      {reviews.hasNextPage && (
        <Button
          disabled={reviews.isFetchingNextPage}
          onClick={() => {
            void reviews.fetchNextPage();
          }}
        >
          {t("design.reviewMore")}
        </Button>
      )}
      {selected && detail.isPending && <p role="status">{t("design.reviewLoading")}</p>}
      {detail.error && (
        <Failure
          error={detail.error}
          retry={() => {
            void detail.refetch();
          }}
        />
      )}
      {detail.data?.review && (
        <article className="grid gap-space-sm" data-critique-id={detail.data.id}>
          <p>{t("design.reviewMinimum", { score: detail.data.assessment?.minimumScore ?? 0 })}</p>
          <p>
            {t("design.reviewVersions", {
              rubric: detail.data.review.rubricVersion,
              policy: detail.data.review.policyVersion,
            })}
          </p>
          <p>
            {t("design.reviewCritic", {
              critic: detail.data.review.critic?.id ?? "",
              version: detail.data.review.critic?.version ?? "",
            })}
          </p>
          <p>
            {detail.data.review.critic?.model} {detail.data.review.critic?.profile}
          </p>
          {detail.data.review.ratings.map((rating) => (
            <div key={rating.dimension} className="grid gap-space-sm">
              <h4>
                {dimensionLabel(rating.dimension)}: {rating.score} / 4
              </h4>
              <p>{rating.rationale}</p>
              <details>
                <summary>{t("design.reviewEvidence")}</summary>
                {rating.evidence.map((evidence, index) => (
                  <CritiqueEvidenceView key={index} evidence={evidence} />
                ))}
              </details>
            </div>
          ))}
          <h4>{t("design.reviewFindings")}</h4>
          {detail.data.review.findings.map((finding, index) => (
            <div key={index} className="grid gap-space-sm">
              <p>
                {dimensionLabel(finding.dimension)} · {severityLabel(finding.severity)}
              </p>
              <p>{finding.rationale}</p>
              <p>{t("design.reviewCorrection", { correction: finding.correction })}</p>
              {finding.evidence && (
                <details>
                  <summary>{t("design.reviewEvidence")}</summary>
                  <CritiqueEvidenceView evidence={finding.evidence} />
                </details>
              )}
            </div>
          ))}
          <details className="break-all">
            <summary>{t("design.reviewProvenance")}</summary>
            <p>{detail.data.id}</p>
            <p>{detail.data.hash}</p>
            <p>{target.revision}</p>
            <p>{target.renderHash}</p>
          </details>
        </article>
      )}
    </section>
  );
}
function CritiqueEvidenceView({ evidence }: { evidence: CritiqueEvidence }) {
  const { t } = useTranslation();
  return (
    <div className="grid gap-space-sm break-all">
      <p>
        {t("design.reviewEvidenceContext", {
          region: evidence.region === "$page" ? t("design.reviewWholePage") : evidence.region,
          state: evidence.state,
          width: evidence.width,
          height: evidence.height,
        })}
      </p>
      <code>{evidence.artifact}</code>
      {evidence.renderHash && <code>{evidence.renderHash}</code>}
      <CaptureScreenshotView
        operationId={evidence.captureId}
        reference={evidence.artifact}
        viewportWidth={evidence.width}
        viewportHeight={evidence.height}
      />
    </div>
  );
}

function CandidateAcceptanceCheck({
  candidate,
  renderHash,
  appearance,
  critiqueId,
}: {
  candidate: { scenario: string; designId: string; hash: string };
  renderHash: string;
  appearance: {
    missingLabel: string;
    failedLabel: string;
    theme: string;
    direction: string;
    previewState: string;
  };
  critiqueId?: string;
}) {
  const { t } = useTranslation();
  const check = useMutation({
    mutationFn: () =>
      sketchClient.checkCandidateAcceptance({
        render: { candidate, ...appearance },
        expectedRenderHash: renderHash,
        critiqueIds: critiqueId ? [critiqueId] : [],
      }),
  });
  const [actor, setActor] = useState("");
  const record = useMutation({
    mutationFn: async (newAttempt: boolean) => {
      const request = {
        check: {
          render: { candidate, ...appearance },
          expectedRenderHash: renderHash,
          critiqueIds: critiqueId ? [critiqueId] : [],
        },
        actor: actor.trim(),
      };
      const storageKey = `rcl:acceptance:${JSON.stringify(request)}`;
      const key = (!newAttempt && sessionStorage.getItem(storageKey)) || crypto.randomUUID();
      // Persist intent before dispatch so a lost response can use the same key.
      sessionStorage.setItem(storageKey, key);
      return sketchClient.acceptCandidate({ ...request, idempotencyKey: key });
    },
  });
  return (
    <section className="grid gap-space-sm" aria-label={t("design.acceptanceTitle")}>
      <h4>{t("design.acceptanceTitle")}</h4>
      <p>{t("design.acceptanceExplanation")}</p>
      <Button disabled={check.isPending} onClick={() => check.mutate()}>
        {t("design.acceptanceCheck")}
      </Button>
      {check.isPending && <p role="status">{t("design.acceptanceChecking")}</p>}
      {check.error && <Failure error={check.error} />}
      {check.data && (
        <div className="grid gap-space-sm">
          <p
            role="status"
            className={
              check.data.acceptanceEstablished || check.data.ready
                ? "text-emerald-700 dark:text-emerald-300"
                : "text-amber-700 dark:text-amber-300"
            }
          >
            {check.data.acceptanceEstablished
              ? t("design.acceptanceAlreadyEstablished")
              : check.data.ready
                ? t("design.acceptanceReady")
                : t("design.acceptanceNotAccepted")}
          </p>
          <p className="text-sm text-app-muted-foreground">
            {t("design.acceptanceRequirementSummary", {
              passed: check.data.requirements.filter((row) => row.status === "passed").length,
              total: check.data.requirements.length,
            })}
          </p>
          <ul className="grid gap-space-sm">
            {check.data.requirements.map((row, index) => (
              <li key={`${row.code}:${index}`}>
                <strong>
                  {t(`design.acceptanceRequirement_${row.code}`, { defaultValue: row.code })} ·{" "}
                  {t(`design.acceptanceStatus_${row.status}`, { defaultValue: row.status })}
                </strong>
                <p>{row.detail}</p>
              </li>
            ))}
          </ul>
          <label className="grid gap-space-xs">
            {t("design.acceptanceActor")}
            <input
              value={actor}
              maxLength={200}
              disabled={record.isPending}
              onChange={(event) => {
                setActor(event.target.value);
                record.reset();
              }}
            />
          </label>
          <p>{t("design.acceptanceActorExplanation")}</p>
          <Button
            disabled={record.isPending || !actor.trim() || record.data?.state === "accepted"}
            onClick={() => record.mutate(Boolean(record.data && record.data.state !== "prepared"))}
          >
            {t(
              record.error
                ? "design.acceptanceRetry"
                : record.data
                  ? "design.acceptanceNewAttempt"
                  : "design.acceptanceRecord",
            )}
          </Button>
          {record.isPending && <p role="status">{t("design.acceptanceRecording")}</p>}
          {record.error && <Failure error={record.error} />}
          {record.data && (
            <div className="grid gap-space-sm">
              <p role="status">
                {t(`design.acceptanceDecision_${record.data.state}`, {
                  defaultValue: record.data.state,
                })}
              </p>
              <p>{t("design.acceptanceNoApply")}</p>
              <details className="break-all">
                <summary>{t("design.acceptanceReceipt")}</summary>
                <p>{record.data.id}</p>
                <p>{record.data.hash}</p>
                <p>{record.data.intent?.candidateHash}</p>
                <p>{record.data.intent?.expectedRenderHash}</p>
                {record.data.detail && <p>{record.data.detail}</p>}
              </details>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
