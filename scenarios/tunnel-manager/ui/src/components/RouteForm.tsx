import { useState, useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, X, Trash2, HelpCircle } from "lucide-react";
import { createRoute, updateRoute, deleteRoute, type Route, type RouteInput } from "../lib/api";
import { Button } from "./ui/button";
import { Tooltip } from "./ui/tooltip";
import { ConfirmDialog } from "./ui/confirm-dialog";

/** Tooltip help icon button used beside form field labels. */
function FieldHelp({ label, content }: { label: string; content: string }) {
  return (
    <Tooltip content={content}>
      <button type="button" className="cursor-help text-slate-300 hover:text-slate-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50 rounded" aria-label={`Help: ${label}`}>
        <HelpCircle className="h-3.5 w-3.5" aria-hidden="true" />
      </button>
    </Tooltip>
  );
}

interface RouteFormProps {
  editRoute?: Route;
  onClose: () => void;
}

function RouteFormDialog({ editRoute, onClose }: RouteFormProps) {
  const queryClient = useQueryClient();
  const isEdit = !!editRoute;
  const formRef = useRef<HTMLFormElement>(null);

  const [subdomain, setSubdomain] = useState(editRoute?.subdomain ?? "");
  const [scenarioName, setScenarioName] = useState(editRoute?.scenario_name ?? "");
  const [localPort, setLocalPort] = useState(editRoute?.local_port?.toString() ?? "");
  const [healthPath, setHealthPath] = useState(editRoute?.health_path ?? "/health");
  const [publicUrl, setPublicUrl] = useState(editRoute?.public_url ?? "");
  const [enabled, setEnabled] = useState(editRoute?.enabled ?? true);
  const [formError, setFormError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const mutation = useMutation({
    mutationFn: (input: RouteInput) =>
      isEdit ? updateRoute(editRoute!.id, input) : createRoute(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["routes"] });
      onClose();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteRoute(editRoute!.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["routes"] });
      onClose();
    },
    onError: (err: Error) => {
      setShowDeleteConfirm(false);
      setFormError(err.message);
    },
  });

  function validateField(name: string, value: string) {
    const errors = { ...fieldErrors };
    switch (name) {
      case "subdomain":
        errors.subdomain = value.trim() ? "" : "Subdomain is required";
        break;
      case "scenarioName":
        errors.scenarioName = value.trim() ? "" : "Scenario name is required";
        break;
      case "localPort": {
        const port = parseInt(value, 10);
        errors.localPort = !value.trim()
          ? "Port is required"
          : isNaN(port) || port < 1 || port > 65535
            ? "Port must be 1\u201365535"
            : "";
        break;
      }
    }
    setFieldErrors(errors);
  }

  /** Map field error keys to their input element IDs */
  const fieldInputIds: Record<string, string> = {
    subdomain: "route-subdomain",
    scenarioName: "route-scenario-name",
    localPort: "route-local-port",
  };

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setFormError("");

    const port = parseInt(localPort, 10);
    const errors: Record<string, string> = {};
    if (!subdomain.trim()) errors.subdomain = "Subdomain is required";
    if (!scenarioName.trim()) errors.scenarioName = "Scenario name is required";
    if (isNaN(port) || port < 1 || port > 65535) errors.localPort = "Port must be 1\u201365535";

    if (Object.values(errors).some(Boolean)) {
      setFieldErrors(errors);
      // Focus the first invalid field for accessibility
      const firstErrorKey = Object.keys(errors).find((k) => errors[k]);
      if (firstErrorKey && fieldInputIds[firstErrorKey]) {
        const el = document.getElementById(fieldInputIds[firstErrorKey]);
        el?.focus();
      }
      return;
    }

    mutation.mutate({
      subdomain: subdomain.trim(),
      scenario_name: scenarioName.trim(),
      local_port: port,
      health_path: healthPath.trim() || "/health",
      public_url: publicUrl.trim(),
      enabled,
    });
  }

  const inputClass = (field?: string) =>
    `w-full rounded-lg border ${
      field && fieldErrors[field] ? "border-red-500/50" : "border-white/10"
    } bg-black/20 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/30 transition-colors`;

  return (
    <div className="rounded-xl border border-white/10 bg-slate-900 p-4 sm:p-6" data-testid="route-form">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold">{isEdit ? "Edit Route" : "Add Route"}</h3>
        <Button variant="outline" size="sm" onClick={onClose} aria-label="Close form" data-testid="route-form-close">
          <X className="h-4 w-4" />
        </Button>
      </div>

      <form ref={formRef} onSubmit={handleSubmit} className="space-y-3">
        <div>
          <div className="flex items-center gap-1 mb-1">
            <label htmlFor="route-subdomain" className="text-xs text-slate-300">Subdomain <span className="text-red-400" aria-hidden="true">*</span></label>
            <FieldHelp label="Subdomain field" content="The subdomain that Cloudflare will route to this scenario (e.g., 'api' for api.yourdomain.com)" />
          </div>
          <input
            id="route-subdomain"
            className={inputClass("subdomain")}
            value={subdomain}
            onChange={(e) => setSubdomain(e.target.value)}
            onBlur={() => validateField("subdomain", subdomain)}
            placeholder="my-scenario"
            autoFocus
            required
            aria-required="true"
            aria-invalid={!!fieldErrors.subdomain || undefined}
            aria-describedby={fieldErrors.subdomain ? "route-subdomain-error" : undefined}
          />
          {fieldErrors.subdomain && <p id="route-subdomain-error" className="mt-1 text-xs text-red-400" role="alert">{fieldErrors.subdomain}</p>}
        </div>
        <div>
          <div className="flex items-center gap-1 mb-1">
            <label htmlFor="route-scenario-name" className="text-xs text-slate-300">Scenario Name <span className="text-red-400" aria-hidden="true">*</span></label>
            <FieldHelp label="Scenario Name field" content="The Vrooli scenario name this route serves" />
          </div>
          <input
            id="route-scenario-name"
            className={inputClass("scenarioName")}
            value={scenarioName}
            onChange={(e) => setScenarioName(e.target.value)}
            onBlur={() => validateField("scenarioName", scenarioName)}
            placeholder="my-scenario"
            required
            aria-required="true"
            aria-invalid={!!fieldErrors.scenarioName || undefined}
            aria-describedby={fieldErrors.scenarioName ? "route-scenario-error" : undefined}
          />
          {fieldErrors.scenarioName && <p id="route-scenario-error" className="mt-1 text-xs text-red-400" role="alert">{fieldErrors.scenarioName}</p>}
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <div className="flex items-center gap-1 mb-1">
              <label htmlFor="route-local-port" className="text-xs text-slate-300">Local Port <span className="text-red-400" aria-hidden="true">*</span></label>
              <FieldHelp label="Local Port field" content="The port the scenario listens on locally (1-65535)" />
            </div>
            <input
              id="route-local-port"
              className={inputClass("localPort")}
              type="number"
              value={localPort}
              onChange={(e) => setLocalPort(e.target.value)}
              onBlur={() => validateField("localPort", localPort)}
              placeholder="3000"
              min={1}
              max={65535}
              required
              aria-required="true"
              aria-invalid={!!fieldErrors.localPort || undefined}
              aria-describedby={fieldErrors.localPort ? "route-port-error" : undefined}
            />
            {fieldErrors.localPort && <p id="route-port-error" className="mt-1 text-xs text-red-400" role="alert">{fieldErrors.localPort}</p>}
          </div>
          <div>
            <div className="flex items-center gap-1 mb-1">
              <label htmlFor="route-health-path" className="text-xs text-slate-300">Health Path</label>
              <FieldHelp label="Health Path field" content="The HTTP path used for liveness probes (defaults to /health)" />
            </div>
            <input
              id="route-health-path"
              className={inputClass()}
              value={healthPath}
              onChange={(e) => setHealthPath(e.target.value)}
              placeholder="/health"
            />
          </div>
        </div>
        <div>
          <div className="flex items-center gap-1 mb-1">
            <label htmlFor="route-public-url" className="text-xs text-slate-300">Public URL</label>
            <FieldHelp label="Public URL field" content="The full public URL for this route (optional, for display purposes)" />
          </div>
          <input
            id="route-public-url"
            className={inputClass()}
            value={publicUrl}
            onChange={(e) => setPublicUrl(e.target.value)}
            placeholder="https://my-scenario.example.com"
          />
        </div>
        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="route-enabled"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="rounded border-white/10"
          />
          <label htmlFor="route-enabled" className="text-sm text-slate-300">
            Enabled
          </label>
        </div>

        {formError && (
          <div className="rounded-lg border border-red-500/20 bg-red-500/5 px-3 py-2" role="alert">
            <p className="text-sm text-red-400">{formError}</p>
          </div>
        )}

        <div className="flex flex-wrap gap-2 pt-2">
          <Button type="submit" disabled={mutation.isPending} data-testid="route-form-submit">
            {mutation.isPending ? "Saving..." : isEdit ? "Update" : "Create"}
          </Button>
          {isEdit && (
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowDeleteConfirm(true)}
              disabled={deleteMutation.isPending}
              className="text-red-400 border-red-400/30 hover:bg-red-500/10"
              data-testid="route-form-delete"
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </Button>
          )}
        </div>
      </form>

      <ConfirmDialog
        open={showDeleteConfirm}
        title="Delete Route"
        description={`Are you sure you want to delete the route "${editRoute?.subdomain ?? ""}"? This action cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Keep"
        variant="danger"
        isPending={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
        onCancel={() => setShowDeleteConfirm(false)}
      />
    </div>
  );
}

export function RouteManagement() {
  const [showForm, setShowForm] = useState(false);
  const [editRoute, setEditRoute] = useState<Route | undefined>();

  function openAdd() {
    setEditRoute(undefined);
    setShowForm(true);
  }

  function openEdit(route: Route) {
    setEditRoute(route);
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditRoute(undefined);
  }

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Add &amp; Edit Routes</h2>
        {!showForm && (
          <Button variant="outline" size="sm" onClick={openAdd} data-testid="route-add-button">
            <Plus className="h-4 w-4 mr-2" />
            Add Route
          </Button>
        )}
      </div>

      {showForm && <div className="mt-4"><RouteFormDialog editRoute={editRoute} onClose={closeForm} /></div>}

      {!showForm && (
        <p className="mt-4 text-sm text-slate-300">
          Click "Add Route" to create a new route entry, or click a route in the table above to edit it.
        </p>
      )}
    </div>
  );
}

export { RouteFormDialog };
export type { RouteFormProps };
