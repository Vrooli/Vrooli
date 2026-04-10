// REQ-UI-009: Subscription management
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Bell, Plus, Trash2, RefreshCw } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/button";
import { EmptyState } from "../components/EmptyState";
import { ErrorAlert } from "../components/ErrorAlert";
import { PageHeader } from "../components/PageHeader";
import { Panel } from "../components/Panel";
import { StatusBadge } from "../components/StatusBadge";
import {
  fetchSubscriptions,
  createSubscription,
  deleteSubscription,
  type SubscriptionData,
} from "../lib/api";
import { Spinner } from "../components/Spinner";
import { HEALTH_POLL_INTERVAL_MS, INPUT_CLASS } from "../lib/constants";

type NewSubForm = {
  name: string;
  owner_scenario: string;
  event_pattern: string;
  source_filter: string;
  delivery_type: "sse" | "webhook";
  delivery_target: string;
  enabled: boolean;
};

const DEFAULT_FORM: NewSubForm = {
  name: "",
  owner_scenario: "",
  event_pattern: "",
  source_filter: "",
  delivery_type: "sse",
  delivery_target: "",
  enabled: true,
};

export function SubscriptionsPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<NewSubForm>({ ...DEFAULT_FORM });

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["subscriptions"],
    queryFn: fetchSubscriptions,
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  const createMutation = useMutation({
    mutationFn: createSubscription,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
      setShowForm(false);
      setForm({ ...DEFAULT_FORM });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteSubscription,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
    },
  });

  const handleCreate = () => {
    createMutation.mutate({
      name: form.name,
      owner_scenario: form.owner_scenario,
      event_pattern: form.event_pattern,
      source_filter: form.source_filter || undefined,
      delivery_type: form.delivery_type,
      delivery_target: form.delivery_target,
      enabled: form.enabled,
    });
  };

  return (
    <div className="space-y-4" data-testid="subscriptions-page">
      <PageHeader
        icon={Bell}
        title="Subscriptions"
        actions={
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={() => refetch()}>
              <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
              Refresh
            </Button>
            <Button
              size="sm"
              onClick={() => setShowForm((v) => !v)}
              data-testid="subscriptions-new-button"
            >
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              New Subscription
            </Button>
          </div>
        }
      />

      {isLoading && <Spinner label="Loading subscriptions..." />}
      {error && <ErrorAlert error={error} onRetry={() => refetch()} compact />}
      {createMutation.error && <ErrorAlert error={createMutation.error} compact />}

      {showForm && (
        <Panel data-testid="subscriptions-new-form">
          <h3 className="mb-3 text-sm font-medium text-[var(--text-secondary)]">Create Subscription</h3>
          <div className="flex flex-wrap gap-3">
            <input
              type="text"
              placeholder="Name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className={`w-40 ${INPUT_CLASS}`}
            />
            <input
              type="text"
              placeholder="Owner scenario"
              value={form.owner_scenario}
              onChange={(e) => setForm({ ...form, owner_scenario: e.target.value })}
              className={`w-40 ${INPUT_CLASS}`}
            />
            <input
              type="text"
              placeholder="Event pattern (e.g. scenario.*)"
              value={form.event_pattern}
              onChange={(e) => setForm({ ...form, event_pattern: e.target.value })}
              className={`w-48 ${INPUT_CLASS}`}
            />
            <input
              type="text"
              placeholder="Source filter"
              value={form.source_filter}
              onChange={(e) => setForm({ ...form, source_filter: e.target.value })}
              className={`w-40 ${INPUT_CLASS}`}
            />
            <select
              value={form.delivery_type}
              onChange={(e) => setForm({ ...form, delivery_type: e.target.value as "sse" | "webhook" })}
              className={INPUT_CLASS}
            >
              <option value="sse">SSE</option>
              <option value="webhook">Webhook</option>
            </select>
            <input
              type="text"
              placeholder="Delivery target (URL)"
              value={form.delivery_target}
              onChange={(e) => setForm({ ...form, delivery_target: e.target.value })}
              className={`w-56 ${INPUT_CLASS}`}
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
            <table className="w-full text-sm" data-testid="subscriptions-table">
              <thead>
                <tr className="border-b border-[var(--border-subtle)] text-left text-xs text-[var(--text-faint)]">
                  <th className="px-3 py-2">Name</th>
                  <th className="px-3 py-2">Owner</th>
                  <th className="px-3 py-2">Event Pattern</th>
                  <th className="px-3 py-2">Delivery</th>
                  <th className="px-3 py-2">Target</th>
                  <th className="px-3 py-2">Enabled</th>
                  <th className="px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {data.map((sub) => (
                  <tr
                    key={sub.id}
                    className="border-b border-[var(--border-subtle)] hover:bg-white/5"
                  >
                    <td className="px-3 py-2 font-medium text-[var(--text-secondary)]">{sub.name}</td>
                    <td className="px-3 py-2 text-[var(--text-secondary)]">{sub.owner_scenario}</td>
                    <td className="px-3 py-2 font-mono text-xs text-[var(--text-accent)]">
                      {sub.event_pattern}
                    </td>
                    <td className="px-3 py-2">
                      <span className="rounded bg-[var(--surface-inset)] px-2 py-0.5 text-xs text-[var(--text-secondary)]">
                        {sub.delivery_type}
                      </span>
                    </td>
                    <td className="max-w-[200px] truncate px-3 py-2 font-mono text-xs text-[var(--text-secondary)]">
                      {sub.delivery_target}
                    </td>
                    <td className="px-3 py-2">
                      <StatusBadge active={sub.enabled} />
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex gap-2">
                        <button
                          onClick={() => navigate(`/subscriptions/${sub.id}/health`)}
                          className="text-xs text-[var(--text-accent)] hover:underline"
                        >
                          Health
                        </button>
                        <button
                          onClick={() => deleteMutation.mutate(sub.id)}
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
        <EmptyState icon={Bell} title="No subscriptions yet" description="Create a subscription to start receiving event notifications via webhook or SSE." />
      )}
    </div>
  );
}
