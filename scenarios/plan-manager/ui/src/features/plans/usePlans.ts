/**
 * TanStack Query hooks for the plans domain. Centralizing the query keys and
 * the client calls keeps cache invalidation consistent across the Plans list,
 * detail, and the surfaces (validation/execution) that select a plan.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  archivePlan,
  createFromTemplate,
  getGraph,
  getPlan,
  listPlans,
  listTemplates,
  renderPlan,
} from "../../api/plans";

export const planKeys = {
  all: ["plans"] as const,
  list: () => [...planKeys.all, "list"] as const,
  detail: (id: string) => [...planKeys.all, "detail", id] as const,
  markdown: (id: string) => [...planKeys.all, "markdown", id] as const,
  graph: (id: string) => [...planKeys.all, "graph", id] as const,
  templates: () => [...planKeys.all, "templates"] as const,
};

export function usePlansList() {
  return useQuery({
    queryKey: planKeys.list(),
    queryFn: () => listPlans({ includeArchived: true }),
  });
}

export function usePlanDetail(id: string) {
  return useQuery({
    // react-query v5 forbids `undefined` as resolved data, so coerce the
    // "plan not found" case to `null` — the detail view treats null as empty.
    queryKey: planKeys.detail(id),
    queryFn: async () => (await getPlan(id)) ?? null,
    enabled: id.length > 0,
  });
}

export function usePlanMarkdown(id: string, enabled: boolean) {
  return useQuery({
    queryKey: planKeys.markdown(id),
    queryFn: () => renderPlan(id),
    enabled: enabled && id.length > 0,
  });
}

export function usePlanGraph(id: string) {
  return useQuery({
    queryKey: planKeys.graph(id),
    queryFn: () => getGraph(id),
    enabled: id.length > 0,
  });
}

export function useTemplates() {
  return useQuery({
    queryKey: planKeys.templates(),
    queryFn: listTemplates,
  });
}

export function useCreateFromTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ templateId, title }: { templateId: string; title: string }) =>
      createFromTemplate(templateId, title),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: planKeys.list() });
    },
  });
}

export function useArchivePlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => archivePlan(id),
    onSuccess: (_data, id) => {
      void qc.invalidateQueries({ queryKey: planKeys.list() });
      void qc.invalidateQueries({ queryKey: planKeys.detail(id) });
    },
  });
}
