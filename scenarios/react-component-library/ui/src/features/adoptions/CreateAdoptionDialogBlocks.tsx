import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { ResolveSource } from "../../api/adoptions";
import {
  DesignAffinity,
  StyleFitVerdictKind,
  type ValidateStyleFitResponse,
} from "../../api/components";
import {
  IssueKind,
  VerdictKind,
  type DepIssue,
  type ValidateAdoptionResponse,
} from "../../api/deps";

export interface WarnAcknowledgementProps {
  ack: boolean;
  setAck: (v: boolean) => void;
}

export function WarnAcknowledgement({ ack, setAck }: WarnAcknowledgementProps) {
  const { t } = useTranslation();
  return (
    <label className="mt-space-2xs flex items-center gap-space-2xs text-xs text-app-warning">
      <input
        type="checkbox"
        data-testid={selectors.adoptions.createVerdictAck}
        checked={ack}
        onChange={(e) => setAck(e.target.checked)}
      />
      {t(strings.adoptions.create.ackLabel)}
    </label>
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

export function VerdictBlock({
  validating,
  verdict,
  kind,
  verdictKindString,
  ack,
  setAck,
}: VerdictBlockProps) {
  const { t } = useTranslation();
  if (validating)
    return (
      <p className="mt-space-xs text-xs text-app-muted-foreground">
        {t(strings.adoptions.create.validating)}
      </p>
    );
  if (!verdict || kind === VerdictKind.UNSPECIFIED) return null;
  const headline =
    kind === VerdictKind.OK
      ? t(strings.adoptions.create.verdictOk)
      : kind === VerdictKind.WARN
        ? t(strings.adoptions.create.verdictWarn)
        : t(strings.adoptions.create.verdictBlock);
  const tone =
    kind === VerdictKind.OK
      ? "border-app-success/40 bg-app-success/10 text-app-success"
      : kind === VerdictKind.WARN
        ? "border-app-warning/40 bg-app-warning/10 text-app-warning"
        : "border-app-danger/40 bg-app-danger/10 text-app-danger";
  const badgeTone = kind === VerdictKind.OK ? "success" : kind === VerdictKind.WARN ? "warning" : "danger";
  return (
    <div
      data-testid={selectors.adoptions.createVerdict}
      data-verdict-kind={verdictKindString}
      className={"mt-space-xs rounded-lg border p-space-xs text-xs " + tone}
    >
      <div data-testid={selectors.adoptions.createVerdictKind} className="font-medium">
        <StatusBadge tone={badgeTone}>{headline}</StatusBadge>
      </div>
      {verdict.issues.length > 0 && (
        <ul className="mt-space-2xs space-y-space-3xs">
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
      {(kind === VerdictKind.WARN || kind === VerdictKind.BLOCK) && (
        <WarnAcknowledgement ack={ack} setAck={setAck} />
      )}
    </div>
  );
}

export function StyleFitBlock({
  validating,
  verdict,
}: {
  validating: boolean;
  verdict: ValidateStyleFitResponse | null;
}) {
  const { t } = useTranslation();
  if (validating)
    return (
      <p className="mt-space-2xs text-xs text-app-muted-foreground">
        {t(strings.adoptions.create.styleValidating)}
      </p>
    );
  if (!verdict || verdict.kind === StyleFitVerdictKind.UNSPECIFIED) return null;
  const kindString = styleFitKindString(verdict.kind);
  const tone =
    verdict.kind === StyleFitVerdictKind.WARN
      ? "border-app-warning/40 bg-app-warning/10 text-app-warning"
      : verdict.kind === StyleFitVerdictKind.INFO
        ? "border-app-info/40 bg-app-info/10 text-app-info"
        : "border-app-success/40 bg-app-success/10 text-app-success";
  return (
    <div
      data-testid={selectors.adoptions.createStyleVerdict}
      data-verdict-kind={kindString}
      className={"mt-space-2xs rounded-lg border p-space-xs text-xs " + tone}
    >
      <div data-testid={selectors.adoptions.createStyleVerdictKind} className="font-medium">
        {t(strings.adoptions.create.styleVerdict, {
          kind: kindString,
          style: verdict.scenarioStyle || t(strings.adoptions.create.styleUnknown),
          affinity: designAffinityString(verdict.affinity),
        })}
      </div>
      {verdict.detail && (
        <p data-testid={selectors.adoptions.createStyleVerdictDetail} className="mt-space-3xs">
          {verdict.detail}
        </p>
      )}
    </div>
  );
}

function styleFitKindString(kind: StyleFitVerdictKind) {
  return kind === StyleFitVerdictKind.OK
    ? "ok"
    : kind === StyleFitVerdictKind.INFO
      ? "info"
      : kind === StyleFitVerdictKind.WARN
        ? "warn"
        : "unspecified";
}

function designAffinityString(affinity: DesignAffinity) {
  return affinity === DesignAffinity.NATIVE
    ? "native"
    : affinity === DesignAffinity.COMPATIBLE
      ? "compatible"
      : affinity === DesignAffinity.DISCOURAGED
        ? "discouraged"
        : "none";
}

export function PathSourceBadge({
  resolving,
  source,
  warnings,
}: {
  resolving: boolean;
  source: ResolveSource;
  warnings: string[];
}) {
  const { t } = useTranslation();
  if (resolving)
    return (
      <p
        data-testid={selectors.adoptions.createPathSource}
        data-path-source="resolving"
        className="mt-space-3xs text-[11px] text-app-muted-foreground"
      >
        {t(strings.adoptions.create.pathResolving)}
      </p>
    );
  if (source === ResolveSource.UNSPECIFIED) return null;
  const { label, tone, slug } = pathSourceMeta(source, t);
  return (
    <>
      <span
        data-testid={selectors.adoptions.createPathSource}
        data-path-source={slug}
        className={
          "mt-space-3xs inline-block rounded-md border px-space-2xs py-space-3xs text-[11px] " +
          tone
        }
      >
        {label}
      </span>
      {warnings.length > 0 && (
        <ul className="mt-space-3xs space-y-space-4xs text-[11px] text-app-warning">
          {warnings.map((w, idx) => (
            <li key={idx} data-testid={selectors.adoptions.createPathWarning}>
              {w}
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

function pathSourceMeta(source: ResolveSource, t: ReturnType<typeof useTranslation>["t"]) {
  switch (source) {
    case ResolveSource.EXPLICIT:
      return {
        label: t(strings.adoptions.create.pathSourceExplicit),
        tone: "border-app-info/40 bg-app-info/10 text-app-info",
        slug: "explicit",
      };
    case ResolveSource.TEMPLATE_MANIFEST:
      return {
        label: t(strings.adoptions.create.pathSourceTemplateManifest),
        tone: "border-app-success/40 bg-app-success/10 text-app-success",
        slug: "template-manifest",
      };
    case ResolveSource.HEURISTIC:
      return {
        label: t(strings.adoptions.create.pathSourceHeuristic),
        tone: "border-app-warning/40 bg-app-warning/10 text-app-warning",
        slug: "heuristic",
      };
    case ResolveSource.FALLBACK:
      return {
        label: t(strings.adoptions.create.pathSourceFallback),
        tone: "border-app-warning/40 bg-app-warning/10 text-app-warning",
        slug: "fallback",
      };
    default:
      return { label: "", tone: "", slug: "unspecified" };
  }
}

function formatIssue(issue: DepIssue, t: ReturnType<typeof useTranslation>["t"]): string {
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
