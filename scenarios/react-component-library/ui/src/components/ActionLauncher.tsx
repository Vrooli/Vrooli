import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";

import { workflowsClient } from "../api/workflows";
import { useTranslation } from "../i18n";
import { Button } from "./ui/button";
import { Dialog } from "./ui/dialog";
import { Input } from "./ui/input";

export type LauncherAction = "menu" | "extract" | "adopt" | "create" | null;

export function ActionLauncher({ action, onActionChange, onCreate, initialAssetID = "", initialTarget = "" }: { action: LauncherAction; onActionChange: (action: LauncherAction) => void; onCreate: () => void; initialAssetID?: string; initialTarget?: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [sourceScenario, setSourceScenario] = useState("");
  const [sourcePath, setSourcePath] = useState("");
  const [assetID, setAssetID] = useState("");
  const [targets, setTargets] = useState("");
  useEffect(() => { if (action === "adopt") { setAssetID(initialAssetID); setTargets(initialTarget); } }, [action, initialAssetID, initialTarget]);
  const extract = useMutation({ mutationFn: () => workflowsClient.startWorkflow({ kind: 1, sourceScenario, sourcePath, idempotencyKey: `launcher-extract:${sourceScenario}:${sourcePath}:${Date.now()}` }), onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ["workflows"] }); onActionChange(null); } });
  const adopt = useMutation({ mutationFn: async () => Promise.all(targets.split(",").map((target) => target.trim()).filter(Boolean).map((target) => workflowsClient.startWorkflow({ kind: 2, assetId: assetID, targetScenario: target, idempotencyKey: `launcher-adopt:${assetID}:${target}:${Date.now()}` }))), onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ["workflows"] }); onActionChange(null); } });
  const close = () => onActionChange(null);
  const extractSubmit = (event: FormEvent<HTMLFormElement>) => { event.preventDefault(); extract.mutate(); };
  const adoptSubmit = (event: FormEvent<HTMLFormElement>) => { event.preventDefault(); adopt.mutate(); };

  return <>
    <Button type="button" data-testid="launcher-open" aria-label={t("launcher.open", { defaultValue: "Open main actions" })} onClick={() => onActionChange("menu")} className="fixed bottom-6 end-6 z-40 h-14 w-14 rounded-full p-0 shadow-lg"><Plus aria-hidden className="h-6 w-6" /></Button>
    <Dialog open={action === "menu"} onClose={close} closeLabel={t("common.close", { defaultValue: "Close" })} title={t("launcher.title", { defaultValue: "Main actions" })} description={t("launcher.description", { defaultValue: "Start a guided library workflow." })}>
      <div role="menu" data-testid="launcher-menu" className="grid gap-2"><Button data-testid="launcher-extract" role="menuitem" onClick={() => onActionChange("extract")}>{t("launcher.extract", { defaultValue: "Extract into library" })}</Button><Button data-testid="launcher-adopt" role="menuitem" variant="secondary" onClick={() => onActionChange("adopt")}>{t("launcher.adopt", { defaultValue: "Adopt into scenarios" })}</Button><Button data-testid="launcher-create" role="menuitem" variant="secondary" onClick={() => { close(); onCreate(); }}>{t("launcher.create", { defaultValue: "Create component" })}</Button></div>
    </Dialog>
    <Dialog open={action === "extract"} onClose={close} closeLabel={t("common.close", { defaultValue: "Close" })} title={t("launcher.extract", { defaultValue: "Extract into library" })} description={t("launcher.extractDescription", { defaultValue: "An extract-assist workflow will inspect this source and report its progress in Active work." })}>
      <form onSubmit={extractSubmit} className="grid gap-3"><label>{t("catalog.sourceScenario", { defaultValue: "Source scenario" })}<Input required value={sourceScenario} onChange={(event) => setSourceScenario(event.target.value)} /></label><label>{t("catalog.sourcePath", { defaultValue: "Source path" })}<Input required value={sourcePath} onChange={(event) => setSourcePath(event.target.value)} /></label>{extract.error && <p role="alert" className="text-sm text-app-danger">{t("launcher.startError", { defaultValue: "Could not start workflow." })}</p>}<Button type="submit" disabled={extract.isPending}>{extract.isPending ? t("launcher.starting", { defaultValue: "Starting…" }) : t("launcher.startExtract", { defaultValue: "Start extract-assist" })}</Button></form>
    </Dialog>
    <Dialog open={action === "adopt"} onClose={close} closeLabel={t("common.close", { defaultValue: "Close" })} title={t("launcher.adopt", { defaultValue: "Adopt into scenarios" })} description={t("launcher.adoptDescription", { defaultValue: "One adopt-assist workflow starts for each target scenario." })}>
      <form onSubmit={adoptSubmit} className="grid gap-3"><label>{t("launcher.asset", { defaultValue: "Library asset ID" })}<Input required value={assetID} onChange={(event) => setAssetID(event.target.value)} /></label><label>{t("launcher.targets", { defaultValue: "Target scenarios" })}<Input required value={targets} onChange={(event) => setTargets(event.target.value)} placeholder={t("launcher.targetsPlaceholder", { defaultValue: "scenario-a, scenario-b" })} /></label>{adopt.error && <p role="alert" className="text-sm text-app-danger">{t("launcher.startError", { defaultValue: "Could not start workflow." })}</p>}<Button type="submit" disabled={adopt.isPending}>{adopt.isPending ? t("launcher.starting", { defaultValue: "Starting…" }) : t("launcher.startAdopt", { defaultValue: "Start adopt-assist" })}</Button></form>
    </Dialog>
  </>;
}
