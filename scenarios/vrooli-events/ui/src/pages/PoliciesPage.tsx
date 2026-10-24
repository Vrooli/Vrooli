// REQ-UI-006: Policy management — rule list
// DOC: docs/reference/api-endpoints.md#policies
// DOC: docs/guides/managing-policies.md
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Shield, Plus, Trash2, RefreshCw } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/button";
import { EmptyState } from "../components/EmptyState";
import { ErrorAlert } from "../components/ErrorAlert";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { Spinner } from "../components/Spinner";
import { StatusBadge } from "../components/StatusBadge";
import {
  fetchPolicies,
  createPolicy,
  deletePolicy,
  updatePolicy,
  type PolicyRule,
} from "../lib/api";
import { HEALTH_POLL_INTERVAL_MS, INPUT_CLASS } from "../lib/constants";

const RULE_TYPES = ["access", "rate_limit", "circuit_breaker"] as const;
type RuleType = typeof RULE_TYPES[number];

type NewRuleForm = {
  rule_type: RuleType;
  source_scenario: string;
  target_scenario: string;
  endpoint_pattern: string;
  effect: "allow" | "deny";
  priority: number;
  enabled: boolean;
};

const RULE_TYPE_SET: ReadonlySet<string> = new Set<string>(RULE_TYPES);
function isRuleType(value: string): value is RuleType {
  return RULE_TYPE_SET.has(value);
}

const DEFAULT_FORM: NewRuleForm = {
  rule_type: "access",
  source_scenario: "",
  target_scenario: "",
  endpoint_pattern: "",
  effect: "allow",
  priority: 0,
  enabled: true,
};

export function PoliciesPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<NewRuleForm>({ ...DEFAULT_FORM });

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["policies"],
    queryFn: fetchPolicies,
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  const createMutation = useMutation({
    mutationFn: createPolicy,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] });
      setShowForm(false);
      setForm({ ...DEFAULT_FORM });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deletePolicy,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
      updatePolicy(id, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] });
    },
  });

  const handleCreate = () => {
    createMutation.mutate({
      rule_type: form.rule_type,
      source_scenario: form.source_scenario,
      target_scenario: form.target_scenario,
      endpoint_pattern: form.endpoint_pattern || undefined,
      effect: form.effect,
      priority: form.priority,
      enabled: form.enabled,
    });
  };

  const renderThresholds = (rule: PolicyRule) => {
    if (rule.rule_type === "access") return rule.effect ?? "-";
    if (rule.rule_type === "rate_limit") {
      return `${rule.max_requests ?? 0}/${rule.window_seconds ?? 0}s`;
    }
    if (rule.rule_type === "circuit_breaker") {
      return `fail:${rule.failure_threshold ?? 0} cool:${rule.cooldown_seconds ?? 0}s`;
    }
    return "-";
  };

  return (
    <div className="space-y-4" data-testid="policies-page">
      <PageHeader
        icon={Shield}
        title="Policy Rules"
        actions={
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={() => refetch()}>
              <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
              Refresh
            </Button>
            <Button
              size="sm"
              onClick={() => setShowForm((v) => !v)}
              data-testid="policies-new-button"
            >
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              New Rule
            </Button>
          </div>
        }
      />

      {isLoading && <Spinner label="Loading policies..." />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}
      {createMutation.error && <ErrorAlert error={createMutation.error} compact />}

      {showForm && (
        <Panel data-testid="policies-new-form">
          <h3 className="mb-3 text-sm font-medium text-[var(--text-secondary)]">Create Policy Rule</h3>
          <div className="flex flex-wrap gap-3">
            <select
              value={form.rule_type}
              onChange={(e) => { if (isRuleType(e.target.value)) setForm({ ...form, rule_type: e.target.value }); }}
              className={INPUT_CLASS}
            >
              <option value="access">Access</option>
              <option value="rate_limit">Rate Limit</option>
              <option value="circuit_breaker">Circuit Breaker</option>
            </select>
            <input
              type="text"
              placeholder="Source scenario"
              value={form.source_scenario}
              onChange={(e) => setForm({ ...form, source_scenario: e.target.value })}
              className={`w-40 ${INPUT_CLASS}`}
            />
            <input
              type="text"
              placeholder="Target scenario"
              value={form.target_scenario}
              onChange={(e) => setForm({ ...form, target_scenario: e.target.value })}
              className={`w-40 ${INPUT_CLASS}`}
            />
            <input
              type="text"
              placeholder="Endpoint pattern"
              value={form.endpoint_pattern}
              onChange={(e) => setForm({ ...form, endpoint_pattern: e.target.value })}
              className={`w-48 ${INPUT_CLASS}`}
            />
            <select
              value={form.effect}
              onChange={(e) => setForm({ ...form, effect: e.target.value as "allow" | "deny" })}
              className={INPUT_CLASS}
            >
              <option value="allow">Allow</option>
              <option value="deny">Deny</option>
            </select>
            <input
              type="number"
              placeholder="Priority"
              value={form.priority}
              onChange={(e) => setForm({ ...form, priority: Number(e.target.value) })}
              className={`w-24 ${INPUT_CLASS}`}
            />
            <Button size="sm" onClick={handleCreate} disabled={createMutation.isPending}>
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </div>
        </Panel>
      )}

      {data && data.length > 0 && (
        <Panel>
          <div className="overflow-x-auto">
            <table className="w-full text-sm" data-testid="policies-table">
              <thead>
                <tr className="border-b border-[var(--border-subtle)] text-left text-xs text-[var(--text-faint)]">
                  <th className="px-3 py-2">Type</th>
                  <th className="px-3 py-2">Source</th>
                  <th className="px-3 py-2">Target</th>
                  <th className="px-3 py-2">Endpoint</th>
                  <th className="px-3 py-2">Effect / Thresholds</th>
                  <th className="px-3 py-2">Priority</th>
                  <th className="px-3 py-2">Enabled</th>
                  <th className="px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {data.map((rule) => (
                  <tr
                    key={rule.id}
                    className="border-b border-[var(--border-subtle)] hover:bg-white/5"
                  >
                    <td className="px-3 py-2 font-mono text-xs text-[var(--text-accent)]">
                      {rule.rule_type}
                    </td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{rule.source_scenario}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{rule.target_scenario}</td>
                    <td className="px-3 py-2 font-mono text-xs text-[var(--text-secondary)]">
                      {rule.endpoint_pattern ?? "-"}
                    </td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{renderThresholds(rule)}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{rule.priority}</td>
                    <td className="px-3 py-2">
                      <button onClick={() => toggleMutation.mutate({ id: rule.id, enabled: !rule.enabled })}>
                        <StatusBadge active={rule.enabled} />
                      </button>
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex gap-2">
                        <button
                          onClick={() => navigate(`/policies/${rule.id}/edit`)}
                          className="text-xs text-[var(--text-accent)] hover:underline"
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => deleteMutation.mutate(rule.id)}
                          className="text-[var(--error-link)] hover:text-[var(--error-link-hover)]"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}

      {data && data.length === 0 && (
        <EmptyState icon={Shield} title="No policy rules yet" description="Create access control, rate limiting, or circuit breaker rules to govern inter-scenario communication." />
      )}
    </div>
  );
}
