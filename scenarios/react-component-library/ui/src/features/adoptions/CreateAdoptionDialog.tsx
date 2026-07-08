import { useEffect, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { adoptionsClient, ResolveSource } from "../../api/adoptions";
import {
  depsClient,
  IssueKind,
  VerdictKind,
  type DepIssue,
  type ValidateAdoptionResponse,
} from "../../api/deps";
import { errorMessage } from "../../lib/errorMessage";

interface Props {
  open: boolean;
  onClose: () => void;
}

/**
 * CreateAdoptionDialog — DC-003 surface.
 *
 * On every (componentId, scenario) change, calls DepsService.ValidateAdoption
 * and renders the verdict (ok | warn | block). Confirm is disabled while
 * validating, when verdict is block, or when verdict is warn without the
 * acknowledgment checkbox checked.
 */
export function CreateAdoptionDialog({ open, onClose }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [componentId, setComponentId] = useState("");
  const [scenario, setScenario] = useState("");
  const [adoptedPath, setAdoptedPath] = useState("");
  const [pathUserEdited, setPathUserEdited] = useState(false);
  const [pathSource, setPathSource] = useState<ResolveSource>(ResolveSource.UNSPECIFIED);
  const [pathWarnings, setPathWarnings] = useState<string[]>([]);
  const [pathResolving, setPathResolving] = useState(false);
  const [adoptedVersion, setAdoptedVersion] = useState("");
  const [ack, setAck] = useState(false);
  const [verdict, setVerdict] = useState<ValidateAdoptionResponse | null>(null);
  const [validating, setValidating] = useState(false);
  const [overwriteRequired, setOverwriteRequired] = useState(false);

  useEffect(() => {
    if (!open) {
      setComponentId("");
      setScenario("");
      setAdoptedPath("");
      setPathUserEdited(false);
      setPathSource(ResolveSource.UNSPECIFIED);
      setPathWarnings([]);
      setPathResolving(false);
      setAdoptedVersion("");
      setAck(false);
      setVerdict(null);
      setValidating(false);
      setOverwriteRequired(false);
    }
  }, [open]);

  // ResolveAdoptionPath pre-fills the adopted-path input from the target
  // scenario's UI manifest. Skips when the user has hand-edited the input —
  // we don't clobber their typing.
  useEffect(() => {
    if (!open) return;
    if (pathUserEdited) return;
    const cid = componentId.trim();
    const sc = scenario.trim();
    if (!cid || !sc) {
      setPathSource(ResolveSource.UNSPECIFIED);
      setPathWarnings([]);
      return;
    }
    let cancelled = false;
    setPathResolving(true);
    adoptionsClient
      .resolveAdoptionPath({ componentId: cid, scenario: sc })
      .then((res) => {
        if (cancelled) return;
        setAdoptedPath(res.path);
        setPathSource(res.source);
        setPathWarnings(res.warnings ?? []);
      })
      .catch(() => {
        if (cancelled) return;
        setPathSource(ResolveSource.UNSPECIFIED);
        setPathWarnings([]);
      })
      .finally(() => {
        if (!cancelled) setPathResolving(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, componentId, scenario, pathUserEdited]);

  useEffect(() => {
    setOverwriteRequired(false);
  }, [componentId, scenario, adoptedPath, adoptedVersion]);

  useEffect(() => {
    if (!open) return;
    const cid = componentId.trim();
    const sc = scenario.trim();
    if (!cid || !sc) {
      setVerdict(null);
      return;
    }
    let cancelled = false;
    setValidating(true);
    setAck(false);
    depsClient
      .validateAdoption({ componentId: cid, scenario: sc, version: adoptedVersion.trim() })
      .then((res) => {
        if (!cancelled) setVerdict(res);
      })
      .catch(() => {
        if (!cancelled) setVerdict(null);
      })
      .finally(() => {
        if (!cancelled) setValidating(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, componentId, scenario, adoptedVersion]);

  const createMutation = useMutation({
    mutationFn: () =>
      adoptionsClient.applyAdoption({
        componentId: componentId.trim(),
        scenario: scenario.trim(),
        adoptedPath: adoptedPath.trim(),
        version: adoptedVersion.trim(),
        confirmOverwrite: overwriteRequired,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["adoptions"] });
      onClose();
    },
    onError: (err) => {
      if (String(err).includes("target file already exists")) {
        setOverwriteRequired(true);
      }
    },
  });

  const kind = verdict?.kind ?? VerdictKind.UNSPECIFIED;
  const proceedDisabled =
    !open ||
    validating ||
    createMutation.isPending ||
    !componentId.trim() ||
    !scenario.trim() ||
    !adoptedPath.trim() ||
    kind === VerdictKind.BLOCK ||
    (kind === VerdictKind.WARN && !ack);

  const verdictKindString = useMemo(() => {
    switch (kind) {
      case VerdictKind.OK:
        return "ok";
      case VerdictKind.WARN:
        return "warn";
      case VerdictKind.BLOCK:
        return "block";
      default:
        return "unspecified";
    }
  }, [kind]);

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="adoptions-create-title"
      data-testid={selectors.adoptions.createDialog}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
    >
      <div className="w-full max-w-md rounded-xl border border-white/10 bg-app-surface p-5 shadow-xl">
        <h3 id="adoptions-create-title" className="text-base font-semibold">
          {t(strings.adoptions.create.title)}
        </h3>
        <p className="mt-1 text-xs text-slate-400">
          {t(strings.adoptions.create.subtitle)}
        </p>

        <div className="mt-3 space-y-2">
          <label className="block text-xs text-slate-400">
            {t(strings.adoptions.create.componentIdLabel)}
            <Input
              data-testid={selectors.adoptions.createComponentId}
              value={componentId}
              onChange={(e) => setComponentId(e.target.value)}
              placeholder={t(strings.adoptions.create.componentIdPlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-slate-400">
            {t(strings.adoptions.create.scenarioLabel)}
            <Input
              data-testid={selectors.adoptions.createScenario}
              value={scenario}
              onChange={(e) => setScenario(e.target.value)}
              placeholder={t(strings.adoptions.create.scenarioPlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-slate-400">
            {t(strings.adoptions.create.adoptedPathLabel)}
            <Input
              data-testid={selectors.adoptions.createAdoptedPath}
              value={adoptedPath}
              onChange={(e) => {
                setAdoptedPath(e.target.value);
                setPathUserEdited(true);
                setPathSource(ResolveSource.EXPLICIT);
              }}
              placeholder={t(strings.adoptions.create.adoptedPathPlaceholder)}
              className="mt-1"
            />
            <PathSourceBadge
              resolving={pathResolving}
              source={pathSource}
              warnings={pathWarnings}
            />
          </label>
          <label className="block text-xs text-slate-400">
            {t(strings.adoptions.create.adoptedVersionLabel)}
            <Input
              data-testid={selectors.adoptions.createAdoptedVersion}
              value={adoptedVersion}
              onChange={(e) => setAdoptedVersion(e.target.value)}
              placeholder={t(strings.adoptions.create.adoptedVersionPlaceholder)}
              className="mt-1"
            />
          </label>
        </div>

        <VerdictBlock
          validating={validating}
          verdict={verdict}
          kind={kind}
          verdictKindString={verdictKindString}
          ack={ack}
          setAck={setAck}
        />

        {createMutation.error && (
          <p
            data-testid={selectors.adoptions.createError}
            className="mt-3 text-xs text-red-400"
          >
            {overwriteRequired
              ? t(strings.adoptions.create.overwriteRequired)
              : errorMessage(createMutation.error, t)}
          </p>
        )}

        <div className="mt-4 flex items-center justify-end gap-2">
          <Button
            variant="outline"
            data-testid={selectors.adoptions.createCancel}
            onClick={onClose}
            disabled={createMutation.isPending}
          >
            {t(strings.adoptions.create.cancelAction)}
          </Button>
          <Button
            data-testid={selectors.adoptions.createConfirm}
            onClick={() => createMutation.mutate()}
            disabled={proceedDisabled}
          >
            {createMutation.isPending
              ? t(strings.adoptions.creating)
              : overwriteRequired
                ? t(strings.adoptions.create.confirmOverwriteAction)
              : t(strings.adoptions.create.confirmAction)}
          </Button>
        </div>
      </div>
    </div>
  );
}

interface VerdictBlockProps {
  validating: boolean;
  verdict: ValidateAdoptionResponse | null;
  kind: VerdictKind;
  verdictKindString: string;
  ack: boolean;
  setAck: (v: boolean) => void;
}

function VerdictBlock({
  validating,
  verdict,
  kind,
  verdictKindString,
  ack,
  setAck,
}: VerdictBlockProps) {
  const { t } = useTranslation();

  if (validating) {
    return (
      <p className="mt-3 text-xs text-slate-400">
        {t(strings.adoptions.create.validating)}
      </p>
    );
  }
  if (!verdict || kind === VerdictKind.UNSPECIFIED) {
    return null;
  }

  const headline =
    kind === VerdictKind.OK
      ? t(strings.adoptions.create.verdictOk)
      : kind === VerdictKind.WARN
        ? t(strings.adoptions.create.verdictWarn)
        : t(strings.adoptions.create.verdictBlock);

  const tone =
    kind === VerdictKind.OK
      ? "border-emerald-700/40 bg-emerald-900/20 text-emerald-200"
      : kind === VerdictKind.WARN
        ? "border-amber-700/40 bg-amber-900/20 text-amber-200"
        : "border-red-700/40 bg-red-900/20 text-red-200";

  return (
    <div
      data-testid={selectors.adoptions.createVerdict}
      data-verdict-kind={verdictKindString}
      className={"mt-3 rounded-lg border p-3 text-xs " + tone}
    >
      <div
        data-testid={selectors.adoptions.createVerdictKind}
        className="font-medium"
      >
        {headline}
      </div>
      {verdict.issues.length > 0 && (
        <ul className="mt-2 space-y-1">
          {verdict.issues.map((issue, idx) => (
            <li
              key={`${issue.depName}-${idx}`}
              data-testid={selectors.adoptions.createVerdictIssue}
            >
              {formatIssue(issue, t)}
            </li>
          ))}
        </ul>
      )}
      {kind === VerdictKind.WARN && (
        <label className="mt-2 flex items-center gap-2 text-xs">
          <input
            type="checkbox"
            data-testid={selectors.adoptions.createVerdictAck}
            checked={ack}
            onChange={(e) => setAck(e.target.checked)}
          />
          {t(strings.adoptions.create.ackLabel)}
        </label>
      )}
    </div>
  );
}

interface PathSourceBadgeProps {
  resolving: boolean;
  source: ResolveSource;
  warnings: string[];
}

function PathSourceBadge({ resolving, source, warnings }: PathSourceBadgeProps) {
  const { t } = useTranslation();
  if (resolving) {
    return (
      <p
        data-testid={selectors.adoptions.createPathSource}
        data-path-source="resolving"
        className="mt-1 text-[11px] text-slate-400"
      >
        {t(strings.adoptions.create.pathResolving)}
      </p>
    );
  }
  if (source === ResolveSource.UNSPECIFIED) {
    return null;
  }
  const { label, tone, slug } = pathSourceMeta(source, t);
  return (
    <>
      <span
        data-testid={selectors.adoptions.createPathSource}
        data-path-source={slug}
        className={"mt-1 inline-block rounded-md border px-2 py-0.5 text-[11px] " + tone}
      >
        {label}
      </span>
      {warnings.length > 0 && (
        <ul className="mt-1 space-y-0.5 text-[11px] text-amber-300">
          {warnings.map((w, idx) => (
            <li
              key={idx}
              data-testid={selectors.adoptions.createPathWarning}
            >
              {w}
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

function pathSourceMeta(
  source: ResolveSource,
  t: ReturnType<typeof useTranslation>["t"],
): { label: string; tone: string; slug: string } {
  switch (source) {
    case ResolveSource.EXPLICIT:
      return {
        label: t(strings.adoptions.create.pathSourceExplicit),
        tone: "border-sky-700/40 bg-sky-900/20 text-sky-200",
        slug: "explicit",
      };
    case ResolveSource.TEMPLATE_MANIFEST:
      return {
        label: t(strings.adoptions.create.pathSourceTemplateManifest),
        tone: "border-emerald-700/40 bg-emerald-900/20 text-emerald-200",
        slug: "template-manifest",
      };
    case ResolveSource.HEURISTIC:
      return {
        label: t(strings.adoptions.create.pathSourceHeuristic),
        tone: "border-amber-700/40 bg-amber-900/20 text-amber-200",
        slug: "heuristic",
      };
    case ResolveSource.FALLBACK:
      return {
        label: t(strings.adoptions.create.pathSourceFallback),
        tone: "border-amber-700/40 bg-amber-900/20 text-amber-200",
        slug: "fallback",
      };
    default:
      return { label: "", tone: "", slug: "unspecified" };
  }
}

function formatIssue(
  issue: DepIssue,
  t: ReturnType<typeof useTranslation>["t"],
): string {
  const vars = {
    name: issue.depName,
    declared: issue.declaredRange,
    scenario: issue.scenarioVersion,
  };
  switch (issue.kind) {
    case IssueKind.MISSING_DEP:
      return t(strings.adoptions.create.issueMissingDep, vars);
    case IssueKind.RANGE_DOES_NOT_MATCH:
      return t(strings.adoptions.create.issueRangeDoesNotMatch, vars);
    case IssueKind.INCOMPATIBLE_MAJOR:
      return t(strings.adoptions.create.issueIncompatibleMajor, vars);
    case IssueKind.UNPARSEABLE_RANGE:
      return t(strings.adoptions.create.issueUnparseableRange, vars);
    case IssueKind.UNPARSEABLE_TARGET:
      return t(strings.adoptions.create.issueUnparseableTarget, vars);
    default:
      return t(strings.adoptions.create.issueUnknown, vars);
  }
}
