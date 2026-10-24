/** @vrooliComponentSource overlays.dialog */
import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "../../i18n";

import { adoptionsClient, ResolveSource } from "../../api/adoptions";
import {
  componentsClient,
  StyleFitVerdictKind,
  type ValidateStyleFitResponse,
} from "../../api/components";
import { depsClient, VerdictKind, type ValidateAdoptionResponse } from "../../api/deps";
import { renderCreateAdoptionDialog } from "./CreateAdoptionDialogView";

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
  const queryClient = useQueryClient();
  const { t } = useTranslation();

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
  const componentIdInputRef = useRef<HTMLInputElement>(null);
  const componentIdRef = useRef(componentId);
  componentIdRef.current = componentId;
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
    mutationFn: () => {
      return adoptionsClient.applyAdoption({
        componentId: componentIdRef.current.trim(),
        scenario: scenario.trim(),
        adoptedPath: adoptedPath.trim(),
        version: adoptedVersion.trim(),
        confirmOverwrite: overwriteRequired,
        ...(replaceExisting ? { replaceExisting: true } : {}),
        ...(kind === VerdictKind.BLOCK && ack ? { overrideValidation: true } : {}),
      });
    },
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
  return renderCreateAdoptionDialog({
    t,
    ...{
      open,
      onClose,
      componentId,
      setComponentId,
      scenario,
      setScenario,
      adoptedPath,
      setAdoptedPath,
      setPathUserEdited,
      setPathSource,
      pathResolving,
      pathSource,
      pathWarnings,
      adoptedVersion,
      setAdoptedVersion,
      replaceExisting,
      setReplaceExisting,
      validating,
      verdict,
      kind,
      verdictKindString,
      ack,
      setAck,
      styleValidating,
      styleVerdict,
      styleKind,
      createMutation,
      proceedDisabled,
      overwriteRequired,
      componentIdInputRef,
    },
  });
}
