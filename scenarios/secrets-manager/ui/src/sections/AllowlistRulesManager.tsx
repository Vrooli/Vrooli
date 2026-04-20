import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Filter, Pencil, Plus, Trash2 } from "lucide-react";
import {
  createAllowlistRule,
  deleteAllowlistRule,
  fetchAllowlistRules,
  updateAllowlistRule,
  type AllowlistRule,
  type UpsertAllowlistRulePayload
} from "../lib/api";
import { Button } from "../components/ui/button";
import { Skeleton } from "../components/ui/LoadingStates";
import { ConfirmDialog } from "../components/ui/ConfirmDialog";

const emptyForm: UpsertAllowlistRulePayload = {
  path_pattern: "",
  excluded_types: ["*"],
  description: "",
  enabled: true
};

export function AllowlistRulesManager() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<UpsertAllowlistRulePayload>(emptyForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<AllowlistRule | null>(null);
  const [typesInput, setTypesInput] = useState<string>("*");

  const listQuery = useQuery({
    queryKey: ["allowlist-rules"],
    queryFn: fetchAllowlistRules,
    refetchOnWindowFocus: false
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["allowlist-rules"] });

  const createMutation = useMutation({
    mutationFn: (payload: UpsertAllowlistRulePayload) => createAllowlistRule(payload),
    onSuccess: () => {
      invalidate();
      resetForm();
    },
    onError: (err: unknown) => setFormError(err instanceof Error ? err.message : "Failed to create rule")
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpsertAllowlistRulePayload }) =>
      updateAllowlistRule(id, payload),
    onSuccess: () => {
      invalidate();
      resetForm();
    },
    onError: (err: unknown) => setFormError(err instanceof Error ? err.message : "Failed to update rule")
  });

  const toggleMutation = useMutation({
    mutationFn: ({ rule, enabled }: { rule: AllowlistRule; enabled: boolean }) =>
      updateAllowlistRule(rule.id, {
        path_pattern: rule.path_pattern,
        excluded_types: rule.excluded_types,
        description: rule.description ?? "",
        enabled
      }),
    onSuccess: invalidate
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteAllowlistRule(id),
    onSuccess: () => {
      invalidate();
      setPendingDelete(null);
    }
  });

  const resetForm = () => {
    setForm(emptyForm);
    setEditingId(null);
    setFormError(null);
    setTypesInput("*");
  };

  const beginEdit = (rule: AllowlistRule) => {
    setEditingId(rule.id);
    setForm({
      path_pattern: rule.path_pattern,
      excluded_types: rule.excluded_types,
      description: rule.description ?? "",
      enabled: rule.enabled
    });
    setTypesInput(rule.excluded_types.join(", "));
    setFormError(null);
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const pathPattern = form.path_pattern.trim();
    const types = typesInput
      .split(",")
      .map((t) => t.trim())
      .filter((t) => t.length > 0);
    if (!pathPattern) {
      setFormError("Path pattern is required");
      return;
    }
    if (types.length === 0) {
      setFormError("At least one excluded type is required (use '*' to exclude all)");
      return;
    }
    const payload: UpsertAllowlistRulePayload = {
      path_pattern: pathPattern,
      excluded_types: types,
      description: form.description?.trim() || undefined,
      enabled: form.enabled ?? true
    };
    if (editingId) {
      updateMutation.mutate({ id: editingId, payload });
    } else {
      createMutation.mutate(payload);
    }
  };

  const rules = listQuery.data?.rules ?? [];
  const submitting = createMutation.isPending || updateMutation.isPending;

  return (
    <section id="anchor-allowlist" className="rounded-3xl border border-white/5 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-white">Scan Allowlist Rules</h2>
          <p className="mt-1 text-xs text-white/60">
            Suppress findings by path glob × finding type. Use <code className="rounded bg-black/30 px-1 font-mono text-xs">*</code> in excluded types to cover all findings for a pattern.
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-white/60">
          <Filter className="h-4 w-4" />
          {`${rules.length} rules`}
        </div>
      </div>

      <form onSubmit={handleSubmit} className="mt-4 grid gap-3 md:grid-cols-6">
        <div className="md:col-span-2">
          <label className="text-xs text-white/60">Path pattern</label>
          <input
            type="text"
            className="mt-2 w-full rounded-2xl border border-white/10 bg-slate-800 px-3 py-2 font-mono text-sm text-white placeholder-white/30"
            placeholder="e.g. *_test.go"
            value={form.path_pattern}
            onChange={(e) => setForm({ ...form, path_pattern: e.target.value })}
          />
        </div>
        <div className="md:col-span-2">
          <label className="text-xs text-white/60">Excluded types (comma-separated, or *)</label>
          <input
            type="text"
            className="mt-2 w-full rounded-2xl border border-white/10 bg-slate-800 px-3 py-2 font-mono text-sm text-white placeholder-white/30"
            placeholder="pii_email, pii_phone_us"
            value={typesInput}
            onChange={(e) => setTypesInput(e.target.value)}
          />
        </div>
        <div className="md:col-span-1">
          <label className="text-xs text-white/60">Description</label>
          <input
            type="text"
            className="mt-2 w-full rounded-2xl border border-white/10 bg-slate-800 px-3 py-2 text-sm text-white placeholder-white/30"
            placeholder="(optional)"
            value={form.description ?? ""}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
          />
        </div>
        <div className="flex items-end gap-2 md:col-span-1">
          <Button
            type="submit"
            className="flex-1"
            size="sm"
            disabled={submitting}
          >
            {editingId ? (
              <>
                <Pencil className="mr-2 h-4 w-4" />
                {submitting ? "Saving..." : "Save"}
              </>
            ) : (
              <>
                <Plus className="mr-2 h-4 w-4" />
                {submitting ? "Adding..." : "Add"}
              </>
            )}
          </Button>
          {editingId ? (
            <Button type="button" variant="ghost" size="sm" onClick={resetForm}>
              Cancel
            </Button>
          ) : null}
        </div>
        {formError ? (
          <p className="md:col-span-6 text-xs text-rose-300">{formError}</p>
        ) : null}
      </form>

      <div className="mt-6 space-y-2">
        {listQuery.isLoading ? (
          <>
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-14 w-full" />
          </>
        ) : rules.length === 0 ? (
          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
            No allowlist rules configured.
          </div>
        ) : (
          rules.map((rule) => (
            <div
              key={rule.id}
              className={`flex flex-col gap-2 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 md:flex-row md:items-center md:justify-between ${
                rule.enabled ? "" : "opacity-60"
              }`}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <code className="truncate rounded bg-black/40 px-2 py-0.5 font-mono text-xs text-emerald-200">
                    {rule.path_pattern}
                  </code>
                  <span className="text-xs text-white/50">→</span>
                  <span className="truncate text-xs text-white/70">
                    {rule.excluded_types.join(", ")}
                  </span>
                </div>
                {rule.description ? (
                  <div className="mt-1 text-xs text-white/50">{rule.description}</div>
                ) : null}
              </div>
              <div className="flex items-center gap-2">
                <label className="flex items-center gap-1 text-xs text-white/60">
                  <input
                    type="checkbox"
                    checked={rule.enabled}
                    disabled={toggleMutation.isPending}
                    onChange={(e) => toggleMutation.mutate({ rule, enabled: e.target.checked })}
                    className="h-4 w-4 rounded border-white/20 bg-slate-800"
                  />
                  Enabled
                </label>
                <Button variant="ghost" size="sm" onClick={() => beginEdit(rule)}>
                  <Pencil className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setPendingDelete(rule)}
                  className="text-rose-300 hover:bg-rose-500/10"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))
        )}
      </div>

      <ConfirmDialog
        open={Boolean(pendingDelete)}
        title="Delete allowlist rule?"
        message={`Pattern "${pendingDelete?.path_pattern ?? ""}" will be removed. Findings it was suppressing will start appearing again.`}
        confirmLabel={deleteMutation.isPending ? "Deleting..." : "Delete"}
        variant="danger"
        onCancel={() => setPendingDelete(null)}
        onConfirm={() => pendingDelete && deleteMutation.mutate(pendingDelete.id)}
      />
    </section>
  );
}
