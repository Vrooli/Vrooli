/** @vrooliComponentSource overlays.dialog */
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Dialog } from "@vrooli/react-component-library/Dialog/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { adoptionsClient, ResolveSource } from "../../api/adoptions";
import {
  componentsClient,
  StyleFitVerdictKind,
  type ValidateStyleFitResponse,
} from "../../api/components";
import { depsClient, VerdictKind, type ValidateAdoptionResponse } from "../../api/deps";
import { errorMessage } from "../../lib/errorMessage";
import {
  PathSourceBadge,
  StyleFitBlock,
  VerdictBlock,
  WarnAcknowledgement,
} from "./CreateAdoptionDialogBlocks";

interface Props {
  open: boolean;
  onClose: () => void;
  initial?: { componentId: string; scenario: string } | null;
}

/**
 * CreateAdoptionDialog — DC-003 surface.
 *
 * On every (componentId, scenario) change, calls DepsService.ValidateAdoption
 * and renders the verdict (ok | warn | block). Confirm is disabled while
 * validating, or when a warning/block verdict has not been explicitly
 * acknowledged. A block acknowledgement is deliberately forwarded as the
 * server-side override_validation flag; direct RPC/CLI callers therefore get
 * the same protection as this UI.
 */
export function CreateAdoptionDialog({ open, onClose, initial }: Props) {
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
  const [styleVerdict, setStyleVerdict] = useState<ValidateStyleFitResponse | null>(null);
  const [validating, setValidating] = useState(false);
  const [styleValidating, setStyleValidating] = useState(false);
  const [overwriteRequired, setOverwriteRequired] = useState(false);
  const [replaceExisting, setReplaceExisting] = useState(false);

  useEffect(() => {
    if (!open) return;
    setComponentId(initial?.componentId ?? "");
    setScenario(initial?.scenario ?? "");
    setAdoptedPath("");
    setPathUserEdited(false);
    setPathSource(ResolveSource.UNSPECIFIED);
    setPathWarnings([]);
    setPathResolving(false);
    setAdoptedVersion("");
    setAck(false);
    setVerdict(null);
    setStyleVerdict(null);
    setValidating(false);
    setStyleValidating(false);
    setOverwriteRequired(false);
    setReplaceExisting(false);
  }, [open, initial]);

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
        setPathWarnings(res.warnings);
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

  useEffect(() => {
    if (!open) return;
    const cid = componentId.trim();
    const sc = scenario.trim();
    if (!cid || !sc) {
      setStyleVerdict(null);
      return;
    }
    let cancelled = false;
    setStyleValidating(true);
    setAck(false);
    componentsClient
      .validateStyleFit({ componentId: cid, scenario: sc, version: adoptedVersion.trim() })
      .then((res) => {
        if (!cancelled) setStyleVerdict(res);
      })
      .catch(() => {
        if (!cancelled) setStyleVerdict(null);
      })
      .finally(() => {
        if (!cancelled) setStyleValidating(false);
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
        ...(replaceExisting ? { replaceExisting: true } : {}),
        ...(kind === VerdictKind.BLOCK && ack ? { overrideValidation: true } : {}),
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
  const styleKind = styleVerdict?.kind ?? StyleFitVerdictKind.UNSPECIFIED;
  const proceedDisabled =
    !open ||
    validating ||
    styleValidating ||
    createMutation.isPending ||
    !componentId.trim() ||
    !scenario.trim() ||
    !adoptedPath.trim() ||
    ((kind === VerdictKind.BLOCK ||
      kind === VerdictKind.WARN ||
      styleKind === StyleFitVerdictKind.WARN) &&
      !ack);

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
    <Dialog
      open={open}
      title={t(strings.adoptions.create.title)}
      description={t(strings.adoptions.create.subtitle)}
      onClose={onClose}
      closeLabel={t(strings.adoptions.create.cancelAction)}
      className="max-w-md"
      footer={
        <div className="flex items-center justify-end gap-space-2xs">
          <Button
            variant="secondary"
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
      }
    >
      <div data-testid={selectors.adoptions.createDialog}>
        <div className="mt-space-xs space-y-space-2xs">
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.adoptions.create.componentIdLabel)}
            <Input
              data-testid={selectors.adoptions.createComponentId}
              value={componentId}
              onChange={(e) => setComponentId(e.target.value)}
              placeholder={t(strings.adoptions.create.componentIdPlaceholder)}
              className="mt-space-3xs"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.adoptions.create.scenarioLabel)}
            <Input
              data-testid={selectors.adoptions.createScenario}
              value={scenario}
              onChange={(e) => setScenario(e.target.value)}
              placeholder={t(strings.adoptions.create.scenarioPlaceholder)}
              className="mt-space-3xs"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
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
              className="mt-space-3xs"
            />
            <PathSourceBadge
              resolving={pathResolving}
              source={pathSource}
              warnings={pathWarnings}
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.adoptions.create.adoptedVersionLabel)}
            <Input
              data-testid={selectors.adoptions.createAdoptedVersion}
              value={adoptedVersion}
              onChange={(e) => setAdoptedVersion(e.target.value)}
              placeholder={t(strings.adoptions.create.adoptedVersionPlaceholder)}
              className="mt-space-3xs"
            />
          </label>
          <label className="flex items-start gap-space-2xs rounded-md border border-app-warning/30 bg-app-warning/10 p-space-2xs text-xs text-app-foreground">
            <input
              type="checkbox"
              aria-label={t(strings.adoptions.create.replaceExistingLabel)}
              checked={replaceExisting}
              onChange={(event) => setReplaceExisting(event.target.checked)}
            />
            <span>
              <span className="block font-medium">
                {t(strings.adoptions.create.replaceExistingLabel)}
              </span>
              <span className="text-app-muted-foreground">
                {t(strings.adoptions.create.replaceExistingHelp)}
              </span>
            </span>
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
        <StyleFitBlock validating={styleValidating} verdict={styleVerdict} />
        {kind !== VerdictKind.WARN && styleKind === StyleFitVerdictKind.WARN && (
          <WarnAcknowledgement ack={ack} setAck={setAck} />
        )}

        {createMutation.error && (
          <p
            data-testid={selectors.adoptions.createError}
            className="mt-space-xs text-xs text-app-danger"
          >
            {overwriteRequired
              ? t(strings.adoptions.create.overwriteRequired)
              : errorMessage(createMutation.error, t)}
          </p>
        )}
      </div>
    </Dialog>
  );
}
