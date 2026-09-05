/** @vrooliComponentSource overlays.dialog */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";

import { workflowsClient } from "../api/workflows";
import { WorkflowKind } from "@vrooli/proto-types/react-component-library/v1/workflows/workflows_pb";
import { adoptionsClient } from "../api/adoptions";
import { listCatalogAssets } from "../api/components";
import { useTranslation } from "../i18n";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Dialog } from "@vrooli/react-component-library/Dialog/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";

export type LauncherAction = "menu" | "extract" | "adopt" | "create" | null;

export type PickerOption = { value: string; label: string };

/** A searchable, typed option picker used by launcher workflow forms. */
function SearchablePicker({
  label,
  options,
  value,
  onChange,
  multiple = false,
}: {
  label: string;
  options: PickerOption[];
  value: string | string[];
  onChange: (value: string | string[]) => void;
  multiple?: boolean;
}) {
  const [query, setQuery] = useState("");
  const normalizedQuery = query.trim().toLocaleLowerCase();
  const visible = options.filter(
    (option) =>
      !normalizedQuery ||
      option.label.toLocaleLowerCase().includes(normalizedQuery) ||
      option.value.toLocaleLowerCase().includes(normalizedQuery),
  );
  return (
    <div className="mt-space-3xs grid gap-space-3xs">
      <Input
        type="search"
        aria-label="Filter available choices"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={`Search ${label}`}
      />
      {/* Select, not a raw <select> with a local border/background class list:
          the shared control owns hover, :focus-visible, aria-invalid, the
          disabled treatment and the reduced-motion opt-out. The multi-select
          height comes from the native `size` attribute rather than a min-height
          utility, so the listbox shows whole rows at any font size. */}
      <Select
        aria-label={label}
        required
        multiple={multiple}
        size={multiple ? 6 : undefined}
        value={value}
        onChange={(event) =>
          onChange(
            multiple
              ? Array.from(event.target.selectedOptions, (option) => option.value)
              : event.target.value,
          )
        }
        placeholder={multiple ? undefined : "Select an option"}
        options={visible}
      />
    </div>
  );
}

export function ActionLauncher({
  action,
  onActionChange,
  onCreate,
  showTrigger = true,
  initialAssetID = "",
  initialTarget = "",
}: {
  action: LauncherAction;
  onActionChange: (action: LauncherAction) => void;
  onCreate: () => void;
  showTrigger?: boolean;
  initialAssetID?: string;
  initialTarget?: string;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [sourceScenario, setSourceScenario] = useState("");
  const [sourcePath, setSourcePath] = useState("");
  const [assetID, setAssetID] = useState("");
  const [targets, setTargets] = useState<string[]>([]);
  const scenarios = useQuery({
    queryKey: ["adoptions", "scenarios"],
    queryFn: () => adoptionsClient.listScenarios({}),
    staleTime: 30_000,
  });
  const assets = useQuery({
    queryKey: ["catalog", "assets", "launcher"],
    queryFn: async () => {
      const [components, hooks] = await Promise.all([
        listCatalogAssets({ limit: 200, assetKind: 1 }),
        listCatalogAssets({ limit: 200, assetKind: 2 }),
      ]);
      return {
        components: Array.from(
          new Map(
            [...components.components, ...hooks.components].map((asset) => [asset.id, asset]),
          ).values(),
        ),
      };
    },
    staleTime: 30_000,
  });
  const catalogAssets = assets.data?.components ?? [];
  const scenarioOptions = scenarios.data?.scenarios ?? [];
  const assetIsListed = catalogAssets.some((asset) => asset.id === assetID);
  const selectedTargetsNotListed = targets.filter(
    (target) => !scenarioOptions.some((scenario) => scenario.name === target),
  );
  useEffect(() => {
    if (action === "adopt") {
      setAssetID(initialAssetID);
      setTargets(initialTarget ? [initialTarget] : []);
    }
  }, [action, initialAssetID, initialTarget]);
  const extract = useMutation({
    mutationFn: () =>
      workflowsClient.startWorkflow({
        kind: WorkflowKind.EXTRACT,
        sourceScenario,
        sourcePath,
        idempotencyKey: `launcher-extract:${sourceScenario}:${sourcePath}:${Date.now()}`,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["workflows"] });
      onActionChange(null);
    },
  });
  const adopt = useMutation({
    mutationFn: async () =>
      Promise.all(
        targets.map((target) =>
          workflowsClient.startWorkflow({
            kind: WorkflowKind.ADOPT,
            assetId: assetID,
            targetScenario: target,
            idempotencyKey: `launcher-adopt:${assetID}:${target}:${Date.now()}`,
          }),
        ),
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["workflows"] });
      onActionChange(null);
    },
  });
  const close = () => onActionChange(null);
  const extractSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    extract.mutate();
  };
  const adoptSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    adopt.mutate();
  };

  return (
    <>
      {showTrigger && (
        <Button
          type="button"
          data-testid="launcher-open"
          aria-label={t("launcher.open", { defaultValue: "Open main actions" })}
          onClick={() => onActionChange("menu")}
          size="icon"
          shape="pill"
          className="fixed bottom-6 end-6 z-40 h-control-2xl w-control-2xl rounded-full p-0 shadow-lg"
          // Justified inline style: the floating trigger has to clear the device
          // safe area, and env() inside calc() is a runtime value no utility class
          // or design token can carry. Both operands are still token-backed.
          style={{
            insetBlockEnd: "calc(var(--space-md) + env(safe-area-inset-bottom, 0px))",
          }}
        >
          <Plus aria-hidden className="h-icon-lg w-icon-lg" />
        </Button>
      )}
      <Dialog
        open={action === "menu"}
        onClose={close}
        closeLabel={t("common.close", { defaultValue: "Close" })}
        title={t("launcher.title", { defaultValue: "Main actions" })}
        description={t("launcher.description", {
          defaultValue: "Start a guided library workflow.",
        })}
      >
        <div role="menu" data-testid="launcher-menu" className="grid gap-space-2xs">
          <Button
            data-testid="launcher-extract"
            role="menuitem"
            onClick={() => onActionChange("extract")}
          >
            {t("launcher.extract", { defaultValue: "Extract into library" })}
          </Button>
          <Button
            data-testid="launcher-adopt"
            role="menuitem"
            variant="secondary"
            onClick={() => onActionChange("adopt")}
          >
            {t("launcher.adopt", { defaultValue: "Adopt into scenarios" })}
          </Button>
          <Button
            data-testid="launcher-create"
            role="menuitem"
            variant="secondary"
            onClick={() => {
              close();
              onCreate();
            }}
          >
            {t("launcher.create", { defaultValue: "Create component" })}
          </Button>
        </div>
      </Dialog>
      <Dialog
        open={action === "extract"}
        onClose={close}
        closeLabel={t("common.close", { defaultValue: "Close" })}
        title={t("launcher.extract", { defaultValue: "Extract into library" })}
        description={t("launcher.extractDescription", {
          defaultValue:
            "An extract-assist workflow will inspect this source and report its progress in Active work.",
        })}
      >
        <form onSubmit={extractSubmit} className="grid gap-space-xs">
          <div>
            <span>{t("catalog.sourceScenario", { defaultValue: "Source scenario" })}</span>
            <SearchablePicker
              label={t("catalog.sourceScenario", { defaultValue: "Source scenario" })}
              value={sourceScenario}
              onChange={(value) => setSourceScenario(value as string)}
              options={[
                ...scenarioOptions.map((scenario) => ({
                  value: scenario.name,
                  label: scenario.displayName,
                })),
                ...(sourceScenario &&
                !scenarioOptions.some((scenario) => scenario.name === sourceScenario)
                  ? [{ value: sourceScenario, label: sourceScenario }]
                  : []),
              ]}
            />
          </div>
          <label>
            {t("catalog.sourcePath", { defaultValue: "Source path" })}
            <Input
              aria-label={t("catalog.sourcePath", { defaultValue: "Source path" })}
              required
              value={sourcePath}
              onChange={(event) => setSourcePath(event.target.value)}
            />
          </label>
          {extract.error && (
            <p role="alert" className="text-sm text-app-danger">
              {t("launcher.startError", { defaultValue: "Could not start workflow." })}
            </p>
          )}
          <Button type="submit" disabled={extract.isPending}>
            {extract.isPending
              ? t("launcher.starting", { defaultValue: "Starting…" })
              : t("launcher.startExtract", { defaultValue: "Start extract-assist" })}
          </Button>
        </form>
      </Dialog>
      <Dialog
        open={action === "adopt"}
        onClose={close}
        closeLabel={t("common.close", { defaultValue: "Close" })}
        title={t("launcher.adopt", { defaultValue: "Adopt into scenarios" })}
        description={t("launcher.adoptDescription", {
          defaultValue: "One adopt-assist workflow starts for each target scenario.",
        })}
      >
        <form onSubmit={adoptSubmit} className="grid gap-space-xs">
          <div>
            <span>{t("launcher.asset", { defaultValue: "Library asset" })}</span>
            <SearchablePicker
              label={t("launcher.asset", { defaultValue: "Library asset" })}
              value={assetID}
              onChange={(value) => setAssetID(value as string)}
              options={[
                ...catalogAssets.map((asset) => ({
                  value: asset.id,
                  label: asset.displayName || asset.libraryId,
                })),
                ...(assetID && !assetIsListed ? [{ value: assetID, label: assetID }] : []),
              ]}
            />
          </div>
          <div>
            <span>{t("launcher.targets", { defaultValue: "Target scenarios" })}</span>
            <SearchablePicker
              label={t("launcher.targets", { defaultValue: "Target scenarios" })}
              multiple
              value={targets}
              onChange={(value) => setTargets(value as string[])}
              options={[
                ...scenarioOptions.map((scenario) => ({
                  value: scenario.name,
                  label: scenario.displayName,
                })),
                ...selectedTargetsNotListed.map((target) => ({ value: target, label: target })),
              ]}
            />
          </div>
          {adopt.error && (
            <p role="alert" className="text-sm text-app-danger">
              {t("launcher.startError", { defaultValue: "Could not start workflow." })}
            </p>
          )}
          <Button type="submit" disabled={adopt.isPending || targets.length === 0}>
            {adopt.isPending
              ? t("launcher.starting", { defaultValue: "Starting…" })
              : t("launcher.startAdopt", { defaultValue: "Start adopt-assist" })}
          </Button>
        </form>
      </Dialog>
    </>
  );
}
