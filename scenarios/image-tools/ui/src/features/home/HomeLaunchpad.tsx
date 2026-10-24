import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { NavLink } from "react-router-dom";
import { Camera, ImageUp, ScanText, Sparkles, Upload } from "lucide-react";

import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { LumeMark } from "../../components/ui/lume-mark";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { blobUrl } from "../../api/client";
import { jobsClient } from "../../api/jobs";
import { useTranslation } from "../../i18n";
import { HealthCard } from "../health/HealthCard";
import { AI_CATALOG } from "../workspace/aiCatalog";
import { CREATE_CATALOG } from "../workspace/createCatalog";
import { OP_CATALOG } from "../workspace/opCatalog";
import type { WorkspaceMode } from "../workspace/useWorkspace";
import { imageOutputs } from "../library/outputs";
import { useReopenOutput } from "../library/useReopenOutput";
import { useOpenInWorkspace } from "./useOpenInWorkspace";
import { DEFAULT_SAMPLE, SAMPLES, loadSampleFile, type SampleImage } from "./samples";

const JOBS_QUERY_KEY = ["jobs"] as const;
const FIRST_RUN_KEY = "lume.home.seen";

const QUICK_EDIT_OPS = ["crop", "resize", "convert", "compress", "rotate", "adjust"] as const;
const ENHANCE_OPS = ["background_removal", "upscale", "denoise"] as const;

const readFirstRun = (): boolean => {
  try {
    return typeof window !== "undefined" && !window.localStorage.getItem(FIRST_RUN_KEY);
  } catch {
    return false;
  }
};
const markSeen = (): void => {
  try {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(FIRST_RUN_KEY, "1");
    }
  } catch {
    /* private mode / disabled storage — first-run hint is non-essential. */
  }
};

interface IntentTileProps {
  name: string;
  label: string;
  Icon: typeof Sparkles;
  needsModel?: boolean;
  onClick: () => void;
}

/** One intent tile — icon + friendly label + an honest "needs model" badge. */
function IntentTile({ name, label, Icon, needsModel, onClick }: IntentTileProps) {
  const { t } = useTranslation();
  return (
    <button
      type="button"
      data-testid={selectors.home.tile({ name })}
      onClick={onClick}
      className="group flex min-h-touch items-center gap-3 rounded-panel border border-app-border bg-app-surface px-3 py-3 text-left transition-colors hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
    >
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-app-surface-muted text-app-brand">
        <Icon aria-hidden="true" className="h-5 w-5" />
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="truncate text-sm font-medium text-app-foreground">{label}</span>
        {needsModel ? (
          <span className="text-[11px] text-app-muted-foreground">{t(strings.home.needsModel)}</span>
        ) : null}
      </span>
    </button>
  );
}

interface TileGroupProps {
  heading: string;
  children: React.ReactNode;
}
function TileGroup({ heading, children }: TileGroupProps) {
  return (
    <section className="flex flex-col gap-2">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{heading}</h3>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">{children}</div>
    </section>
  );
}

/**
 * Home — the launchpad. A universal entry (drop / paste / camera / file / try a
 * sample) drops the user into the Workspace pre-set to a task; intent tiles map
 * to the four modes; a recent rail reopens prior outputs. First-run nudges a
 * sample with a gold pulse. Replaces the old placeholder dashboard.
 */
export function HomeLaunchpad() {
  const { t } = useTranslation();
  const openInWorkspace = useOpenInWorkspace();
  const reopen = useReopenOutput();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const cameraInputRef = useRef<HTMLInputElement>(null);
  const [dragActive, setDragActive] = useState(false);
  const [firstRun, setFirstRun] = useState(readFirstRun);

  const jobsQuery = useQuery({
    queryKey: JOBS_QUERY_KEY,
    queryFn: () => jobsClient.listJobs({ limit: 24 }),
  });
  const recent = useMemo(
    () => imageOutputs(jobsQuery.data?.jobs ?? []).slice(0, 8),
    [jobsQuery.data],
  );

  const openFile = useCallback(
    (file: File | null, mode: WorkspaceMode = "edit") => {
      if (!file) {
        return;
      }
      markSeen();
      setFirstRun(false);
      openInWorkspace({ file, mode });
    },
    [openInWorkspace],
  );

  // Desktop clipboard paste — anywhere on Home, paste an image to open it.
  useEffect(() => {
    const onPaste = (event: ClipboardEvent) => {
      const file = Array.from(event.clipboardData?.files ?? []).find((f) =>
        f.type.startsWith("image/"),
      );
      if (file) {
        openFile(file);
      }
    };
    window.addEventListener("paste", onPaste);
    return () => window.removeEventListener("paste", onPaste);
  }, [openFile]);

  const onSample = useCallback(
    (sample: SampleImage) => {
      void loadSampleFile(sample).then((file) => openFile(file, sample.mode));
    },
    [openFile],
  );

  const onDrop = (event: React.DragEvent<HTMLButtonElement>) => {
    event.preventDefault();
    setDragActive(false);
    openFile(event.dataTransfer.files[0] ?? null);
  };

  const showFirstRun = firstRun && recent.length === 0;
  const generate = CREATE_CATALOG.text_to_image;

  return (
    <div data-testid={selectors.home.root} className="flex flex-col gap-8">
      {/* Hero */}
      <header data-testid={selectors.home.hero} className="flex flex-col items-start gap-2">
        <div className="flex items-center gap-3">
          <LumeMark size={40} />
          <div className="flex flex-col">
            <span className="text-2xl font-semibold tracking-tight text-app-foreground">
              {t(strings.app.title)}
            </span>
            <span className="text-sm text-app-muted-foreground">{t(strings.app.tagline)}</span>
          </div>
        </div>
        <h2 className="mt-2 text-xl font-medium text-app-foreground">{t(strings.home.greeting)}</h2>
      </header>

      {/* Universal entry — the dropzone is a native button (drop + click +
          keyboard open the file picker); the explicit actions are siblings so
          there are no nested interactive elements. */}
      <div className="flex flex-col gap-3">
        <button
          type="button"
          data-testid={selectors.home.dropzone}
          aria-label={t(strings.home.entry.title)}
          onClick={() => fileInputRef.current?.click()}
          onDrop={onDrop}
          onDragOver={(e) => {
            e.preventDefault();
            setDragActive(true);
          }}
          onDragLeave={() => setDragActive(false)}
          className={[
            "flex w-full flex-col items-center gap-3 rounded-sheet border-2 border-dashed p-8 text-center transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
            dragActive
              ? "border-app-primary bg-app-primary/5"
              : "border-app-border bg-app-surface hover:border-app-primary",
          ].join(" ")}
        >
          <span className="flex h-12 w-12 items-center justify-center rounded-full bg-app-surface-muted text-app-brand">
            <ImageUp aria-hidden="true" className="h-6 w-6" />
          </span>
          <span className="flex flex-col gap-1">
            <span className="text-base font-medium text-app-foreground">
              {dragActive ? t(strings.home.entry.dropActive) : t(strings.home.entry.title)}
            </span>
            <span className="text-xs text-app-muted-foreground">{t(strings.home.entry.hint)}</span>
          </span>
        </button>
        <div className="flex flex-wrap items-center justify-center gap-2">
          <Button data-testid={selectors.home.chooseButton} onClick={() => fileInputRef.current?.click()}>
            <Upload aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.home.entry.choose)}
          </Button>
          <Button
            variant="outline"
            data-testid={selectors.home.cameraButton}
            onClick={() => cameraInputRef.current?.click()}
          >
            <Camera aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.home.entry.camera)}
          </Button>
          <Button
            variant="outline"
            data-testid={selectors.home.sampleButton}
            onClick={() => onSample(DEFAULT_SAMPLE)}
            className={showFirstRun ? "animate-pulse-gold" : undefined}
          >
            <Sparkles aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.home.entry.sample)}
          </Button>
        </div>
        {showFirstRun ? (
          <p data-testid={selectors.home.firstRun} className="text-center text-xs text-app-brand-strong">
            {t(strings.home.firstRun)}
          </p>
        ) : null}
        <input
          ref={fileInputRef}
          data-testid={selectors.home.fileInput}
          type="file"
          accept="image/*"
          aria-label={t(strings.home.entry.fileLabel)}
          onChange={(e) => openFile(e.target.files?.[0] ?? null)}
          className="sr-only"
        />
        <input
          ref={cameraInputRef}
          data-testid={selectors.home.cameraInput}
          type="file"
          accept="image/*"
          capture="environment"
          aria-label={t(strings.home.entry.camera)}
          onChange={(e) => openFile(e.target.files?.[0] ?? null)}
          className="sr-only"
        />
      </div>

      {/* Intent tiles */}
      <div data-testid={selectors.home.groups} className="grid gap-6 md:grid-cols-2">
        <TileGroup heading={t(strings.home.group.quickEdits)}>
          {QUICK_EDIT_OPS.map((name) => {
            const pres = OP_CATALOG[name];
            if (!pres) {
              return null;
            }
            return (
              <IntentTile
                key={name}
                name={name}
                label={t(pres.labelKey)}
                Icon={pres.Icon}
                onClick={() => openInWorkspace({ mode: "edit", operation: name })}
              />
            );
          })}
        </TileGroup>

        <TileGroup heading={t(strings.home.group.enhance)}>
          {ENHANCE_OPS.map((name) => {
            const pres = AI_CATALOG[name];
            if (!pres) {
              return null;
            }
            return (
              <IntentTile
                key={name}
                name={name}
                label={t(pres.labelKey)}
                Icon={pres.Icon}
                needsModel
                onClick={() => openInWorkspace({ mode: "enhance", operation: name })}
              />
            );
          })}
        </TileGroup>

        <TileGroup heading={t(strings.home.group.create)}>
          {generate ? (
            <IntentTile
              name="text_to_image"
              label={t(generate.labelKey)}
              Icon={generate.Icon}
              needsModel
              onClick={() => openInWorkspace({ mode: "create", operation: "text_to_image" })}
            />
          ) : null}
        </TileGroup>

        <TileGroup heading={t(strings.home.group.analyze)}>
          <IntentTile
            name="analyze"
            label={t(strings.home.analyzeTile)}
            Icon={ScanText}
            onClick={() => openInWorkspace({ mode: "analyze" })}
          />
        </TileGroup>
      </div>

      {/* Samples */}
      <section data-testid={selectors.home.samples} className="flex flex-col gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
          {t(strings.home.samples.heading)}
        </h3>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {SAMPLES.map((sample) => (
            <button
              key={sample.key}
              type="button"
              data-testid={selectors.home.sample({ key: sample.key })}
              onClick={() => onSample(sample)}
              className="group flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-2 text-left transition-colors hover:border-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
            >
              <img
                src={sample.url}
                alt=""
                loading="lazy"
                className="h-20 w-full rounded-control object-cover"
              />
              <span className="truncate text-xs font-medium text-app-foreground">
                {t(sample.labelKey)}
              </span>
            </button>
          ))}
        </div>
      </section>

      {/* Recent rail */}
      <section data-testid={selectors.home.recent} className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
            {t(strings.home.recent.heading)}
          </h3>
          <NavLink
            data-testid={selectors.home.viewLibrary}
            to="/library"
            className="text-xs font-medium text-app-primary hover:underline"
          >
            {t(strings.home.recent.viewAll)}
          </NavLink>
        </div>
        {recent.length === 0 ? (
          <EmptyState
            testId={selectors.home.recentEmpty}
            Icon={Sparkles}
            title={t(strings.home.recent.empty)}
            action={
              <Button variant="outline" onClick={() => onSample(DEFAULT_SAMPLE)}>
                {t(strings.home.entry.sample)}
              </Button>
            }
          />
        ) : (
          <ul data-testid={selectors.home.recentList} className="grid grid-cols-4 gap-2 sm:grid-cols-8">
            {recent.map((item, index) => (
              <li key={item.jobId}>
                <button
                  type="button"
                  data-testid={selectors.home.recentItem({ index: index + 1 })}
                  onClick={() => void reopen(item)}
                  className="block w-full overflow-hidden rounded-control border border-app-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
                >
                  <img
                    src={blobUrl(item.resultRef)}
                    alt={t(strings.home.recent.alt, { operation: item.operation })}
                    loading="lazy"
                    className="aspect-square w-full object-cover"
                    onError={(e) => {
                      e.currentTarget.closest("li")?.setAttribute("hidden", "true");
                    }}
                  />
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <HealthCard />
    </div>
  );
}
