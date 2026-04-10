// REQ-UI-007: Policy management — rule editor
import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ShieldCheck, ArrowLeft } from "lucide-react";
import { Button } from "../components/ui/button";
import { ErrorAlert } from "../components/ErrorAlert";
import { Spinner } from "../components/Spinner";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { fetchPolicy, updatePolicy, type PolicyRule } from "../lib/api";
import { INPUT_CLASS } from "../lib/constants";

type FormState = Omit<PolicyRule, "id" | "created_at" | "updated_at">;

export function PolicyEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const policyId = Number(id);

  const { data, isLoading, error } = useQuery({
    queryKey: ["policy", policyId],
    queryFn: () => fetchPolicy(policyId),
    enabled: !Number.isNaN(policyId),
  });

  const [form, setForm] = useState<FormState | null>(null);

  useEffect(() => {
    if (data && !form) {
      const { id: _id, created_at: _c, updated_at: _u, ...rest } = data;
      setForm(rest);
    }
  }, [data, form]);

  const saveMutation = useMutation({
    mutationFn: (rule: Partial<PolicyRule>) => updatePolicy(policyId, rule),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] });
      queryClient.invalidateQueries({ queryKey: ["policy", policyId] });
      navigate("/policies");
    },
  });

  const handleSave = () => {
    if (!form) return;
    saveMutation.mutate(form);
  };

  const update = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    if (!form) return;
    setForm({ ...form, [key]: value });
  };

  return (
    <div className="space-y-4">
      <PageHeader
        icon={ShieldCheck}
        title={`Edit Policy #${id}`}
        actions={
          <Button size="sm" variant="outline" onClick={() => navigate("/policies")}>
            <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
            Back
          </Button>
        }
      />

      {isLoading && <Spinner label="Loading policy..." />}
      {error && <ErrorAlert error={error} compact />}
      {saveMutation.error && <ErrorAlert error={saveMutation.error} compact />}

      {form && (
        <Panel>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="mb-1 block text-xs text-[var(--text-faint)]">Rule Type</label>
                <select
                  value={form.rule_type}
                  onChange={(e) => update("rule_type", e.target.value)}
                  className={`w-full ${INPUT_CLASS}`}
                >
                  <option value="access">Access</option>
                  <option value="rate_limit">Rate Limit</option>
                  <option value="circuit_breaker">Circuit Breaker</option>
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs text-[var(--text-faint)]">Priority</label>
                <input
                  type="number"
                  value={form.priority}
                  onChange={(e) => update("priority", Number(e.target.value))}
                  className={`w-full ${INPUT_CLASS}`}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs text-[var(--text-faint)]">Source Scenario</label>
                <input
                  type="text"
                  value={form.source_scenario}
                  onChange={(e) => update("source_scenario", e.target.value)}
                  className={`w-full ${INPUT_CLASS}`}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs text-[var(--text-faint)]">Target Scenario</label>
                <input
                  type="text"
                  value={form.target_scenario}
                  onChange={(e) => update("target_scenario", e.target.value)}
                  className={`w-full ${INPUT_CLASS}`}
                />
              </div>
              <div className="col-span-2">
                <label className="mb-1 block text-xs text-[var(--text-faint)]">Endpoint Pattern</label>
                <input
                  type="text"
                  value={form.endpoint_pattern ?? ""}
                  onChange={(e) => update("endpoint_pattern", e.target.value || undefined)}
                  className={`w-full ${INPUT_CLASS}`}
                  placeholder="/api/v1/*"
                />
              </div>
            </div>

            {/* Access-specific fields */}
            {form.rule_type === "access" && (
              <div>
                <label className="mb-1 block text-xs text-[var(--text-faint)]">Effect</label>
                <select
                  value={form.effect ?? "allow"}
                  onChange={(e) => update("effect", e.target.value)}
                  className={`w-full ${INPUT_CLASS}`}
                >
                  <option value="allow">Allow</option>
                  <option value="deny">Deny</option>
                </select>
              </div>
            )}

            {/* Rate-limit-specific fields */}
            {form.rule_type === "rate_limit" && (
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="mb-1 block text-xs text-[var(--text-faint)]">Max Requests</label>
                  <input
                    type="number"
                    value={form.max_requests ?? 0}
                    onChange={(e) => update("max_requests", Number(e.target.value))}
                    className={`w-full ${INPUT_CLASS}`}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs text-[var(--text-faint)]">Window (seconds)</label>
                  <input
                    type="number"
                    value={form.window_seconds ?? 0}
                    onChange={(e) => update("window_seconds", Number(e.target.value))}
                    className={`w-full ${INPUT_CLASS}`}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs text-[var(--text-faint)]">Burst Allowance</label>
                  <input
                    type="number"
                    value={form.burst_allowance ?? 0}
                    onChange={(e) => update("burst_allowance", Number(e.target.value))}
                    className={`w-full ${INPUT_CLASS}`}
                  />
                </div>
              </div>
            )}

            {/* Circuit-breaker-specific fields */}
            {form.rule_type === "circuit_breaker" && (
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="mb-1 block text-xs text-[var(--text-faint)]">Failure Threshold</label>
                  <input
                    type="number"
                    value={form.failure_threshold ?? 0}
                    onChange={(e) => update("failure_threshold", Number(e.target.value))}
                    className={`w-full ${INPUT_CLASS}`}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs text-[var(--text-faint)]">Cooldown (seconds)</label>
                  <input
                    type="number"
                    value={form.cooldown_seconds ?? 0}
                    onChange={(e) => update("cooldown_seconds", Number(e.target.value))}
                    className={`w-full ${INPUT_CLASS}`}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs text-[var(--text-faint)]">Success Threshold</label>
                  <input
                    type="number"
                    value={form.success_threshold ?? 0}
                    onChange={(e) => update("success_threshold", Number(e.target.value))}
                    className={`w-full ${INPUT_CLASS}`}
                  />
                </div>
              </div>
            )}

            <div className="flex items-center gap-3">
              <label className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                <input
                  type="checkbox"
                  checked={form.enabled}
                  onChange={(e) => update("enabled", e.target.checked)}
                />
                Enabled
              </label>
            </div>

            <div className="flex gap-2">
              <Button size="sm" onClick={handleSave} disabled={saveMutation.isPending}>
                {saveMutation.isPending ? "Saving..." : "Save Changes"}
              </Button>
              <Button size="sm" variant="outline" onClick={() => navigate("/policies")}>
                Cancel
              </Button>
            </div>
          </div>
        </Panel>
      )}
    </div>
  );
}
