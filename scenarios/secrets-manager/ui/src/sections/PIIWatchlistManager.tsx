import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Plus, ShieldCheck, Trash2 } from "lucide-react";
import {
  createWatchlistEntry,
  deleteWatchlistEntry,
  fetchWatchlist,
  type CreateWatchlistEntryPayload,
  type WatchlistValueType
} from "../lib/api";
import { Button } from "../components/ui/button";
import { Skeleton } from "../components/ui/LoadingStates";
import { ConfirmDialog } from "../components/ui/ConfirmDialog";

const valueTypeOptions: Array<{ value: WatchlistValueType; label: string }> = [
  { value: "email", label: "Email" },
  { value: "phone", label: "Phone" },
  { value: "path", label: "Path / Username" },
  { value: "ssn", label: "SSN" },
  { value: "custom", label: "Custom" }
];

const initialForm: CreateWatchlistEntryPayload = {
  label: "",
  value: "",
  value_type: "email"
};

export function PIIWatchlistManager() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CreateWatchlistEntryPayload>(initialForm);
  const [formError, setFormError] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<{ id: string; label: string } | null>(null);

  const listQuery = useQuery({
    queryKey: ["watchlist"],
    queryFn: fetchWatchlist,
    refetchOnWindowFocus: false,
    retry: (failureCount, error: unknown) => {
      const msg = error instanceof Error ? error.message : "";
      if (msg.includes("(503)")) return false;
      return failureCount < 2;
    }
  });

  const createMutation = useMutation({
    mutationFn: (payload: CreateWatchlistEntryPayload) => createWatchlistEntry(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["watchlist"] });
      setForm(initialForm);
      setFormError(null);
    },
    onError: (err: unknown) => {
      setFormError(err instanceof Error ? err.message : "Failed to create entry");
    }
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteWatchlistEntry(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["watchlist"] });
      setPendingDelete(null);
    }
  });

  const listErrMsg = listQuery.error instanceof Error ? listQuery.error.message : "";
  const keyMissing = listErrMsg.includes("(503)");

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const label = form.label.trim();
    const value = form.value;
    if (!label || !value) {
      setFormError("Label and value are required");
      return;
    }
    createMutation.mutate({ ...form, label });
  };

  const entries = listQuery.data?.entries ?? [];

  return (
    <section id="anchor-watchlist" className="rounded-3xl border border-white/5 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-white">PII Watchlist</h2>
          <p className="mt-1 text-xs text-white/60">
            Values stored here are encrypted at rest and always flagged when found during a scan.
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-white/60">
          <ShieldCheck className="h-4 w-4" />
          {keyMissing ? "Encryption key not configured" : `${entries.length} entries`}
        </div>
      </div>

      {keyMissing ? (
        <div className="mt-4 flex items-start gap-3 rounded-2xl border border-amber-400/30 bg-amber-500/10 p-4 text-sm text-amber-100">
          <AlertTriangle className="mt-0.5 h-4 w-4 flex-none" />
          <div>
            <p className="font-semibold">Watchlist encryption key is not configured.</p>
            <p className="mt-1 text-amber-100/80">
              Set the <code className="rounded bg-black/30 px-1 py-0.5 font-mono text-xs">SECRETS_MANAGER_WATCHLIST_KEY</code>
              {" "}environment variable (32-byte hex) to enable watchlist management. Scans still run for secrets and PII.
            </p>
          </div>
        </div>
      ) : (
        <>
          <form onSubmit={handleSubmit} className="mt-4 grid gap-3 md:grid-cols-4">
            <div className="md:col-span-1">
              <label className="text-xs text-white/60">Label</label>
              <input
                type="text"
                className="mt-2 w-full rounded-2xl border border-white/10 bg-slate-800 px-3 py-2 text-sm text-white placeholder-white/30"
                placeholder="e.g. My work email"
                value={form.label}
                onChange={(e) => setForm({ ...form, label: e.target.value })}
              />
            </div>
            <div className="md:col-span-1">
              <label className="text-xs text-white/60">Type</label>
              <select
                className="mt-2 w-full rounded-2xl border border-white/10 bg-slate-800 px-3 py-2 text-sm text-white [&_option]:bg-slate-800 [&_option]:text-white"
                value={form.value_type}
                onChange={(e) => setForm({ ...form, value_type: e.target.value as WatchlistValueType })}
              >
                {valueTypeOptions.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </div>
            <div className="md:col-span-1">
              <label className="text-xs text-white/60">Value</label>
              <input
                type="password"
                autoComplete="off"
                className="mt-2 w-full rounded-2xl border border-white/10 bg-slate-800 px-3 py-2 text-sm text-white placeholder-white/30"
                placeholder="Encrypted on save"
                value={form.value}
                onChange={(e) => setForm({ ...form, value: e.target.value })}
              />
            </div>
            <div className="flex items-end md:col-span-1">
              <Button
                type="submit"
                className="w-full"
                size="sm"
                disabled={createMutation.isPending}
              >
                <Plus className="mr-2 h-4 w-4" />
                {createMutation.isPending ? "Adding..." : "Add entry"}
              </Button>
            </div>
            {formError ? (
              <p className="md:col-span-4 text-xs text-rose-300">{formError}</p>
            ) : null}
          </form>

          <div className="mt-6 space-y-2">
            {listQuery.isLoading ? (
              <>
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </>
            ) : entries.length === 0 ? (
              <div className="rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
                No watchlist entries yet. Add a value above to start flagging it in scans.
              </div>
            ) : (
              entries.map((entry) => (
                <div
                  key={entry.id}
                  className="flex items-center justify-between rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm"
                >
                  <div className="flex-1">
                    <div className="font-medium text-white">{entry.label}</div>
                    <div className="mt-0.5 text-xs text-white/50">
                      Type: <span className="font-mono">{entry.value_type}</span>
                      {" · "}
                      Added {new Date(entry.created_at).toLocaleDateString()}
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setPendingDelete({ id: entry.id, label: entry.label })}
                    aria-label={`Delete watchlist entry ${entry.label}`}
                    className="text-rose-300 hover:bg-rose-500/10"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))
            )}
          </div>
        </>
      )}

      <ConfirmDialog
        open={Boolean(pendingDelete)}
        title="Delete watchlist entry?"
        message={`"${pendingDelete?.label ?? ""}" will be permanently removed. The value cannot be recovered without the encryption key.`}
        confirmLabel={deleteMutation.isPending ? "Deleting..." : "Delete"}
        variant="danger"
        onCancel={() => setPendingDelete(null)}
        onConfirm={() => pendingDelete && deleteMutation.mutate(pendingDelete.id)}
      />
    </section>
  );
}
