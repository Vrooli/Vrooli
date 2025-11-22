// API client with React Query hooks for react-component-library

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { Component, AdoptionRecord, ComponentVersion } from "../types";

const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:16871";

// Generic fetch wrapper
async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || `HTTP ${response.status}`);
  }

  return response.json();
}

// Component API hooks
export function useComponents(search?: string, category?: string) {
  return useQuery({
    queryKey: ["components", search, category],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (search) params.append("search", search);
      if (category) params.append("category", category);
      return fetchAPI<Component[]>(`/api/v1/components?${params}`);
    },
  });
}

export function useComponent(id: string) {
  return useQuery({
    queryKey: ["component", id],
    queryFn: () => fetchAPI<Component>(`/api/v1/components/${id}`),
    enabled: !!id,
  });
}

export function useComponentContent(id: string) {
  return useQuery({
    queryKey: ["component-content", id],
    queryFn: () => fetchAPI<{ content: string }>(`/api/v1/components/${id}/content`),
    enabled: !!id,
  });
}

export function useCreateComponent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Component>) =>
      fetchAPI<Component>("/api/v1/components", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["components"] });
    },
  });
}

export function useUpdateComponent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Component> }) =>
      fetchAPI<Component>(`/api/v1/components/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["component", variables.id] });
      queryClient.invalidateQueries({ queryKey: ["components"] });
    },
  });
}

export function useUpdateComponentContent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, content }: { id: string; content: string }) =>
      fetchAPI(`/api/v1/components/${id}/content`, {
        method: "PUT",
        body: JSON.stringify({ content }),
      }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["component-content", variables.id] });
    },
  });
}

// Adoption API hooks
export function useAdoptions() {
  return useQuery({
    queryKey: ["adoptions"],
    queryFn: () => fetchAPI<AdoptionRecord[]>("/api/v1/adoptions"),
  });
}

export function useCreateAdoption() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { componentId: string; scenarioName: string; adoptedPath: string }) =>
      fetchAPI<AdoptionRecord>("/api/v1/adoptions", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["adoptions"] });
    },
  });
}

// Version API hooks
export function useComponentVersions(componentId: string) {
  return useQuery({
    queryKey: ["component-versions", componentId],
    queryFn: () => fetchAPI<ComponentVersion[]>(`/api/v1/components/${componentId}/versions`),
    enabled: !!componentId,
  });
}

// AI API hooks
export function useAIChat() {
  return useMutation({
    mutationFn: ({ message, context }: { message: string; context?: Record<string, unknown> }) =>
      fetchAPI<{ response: string; suggestions?: string[] }>("/api/v1/ai/chat", {
        method: "POST",
        body: JSON.stringify({ message, context }),
      }),
  });
}

export function useAIRefactor() {
  return useMutation({
    mutationFn: ({ code, instruction }: { code: string; instruction: string }) =>
      fetchAPI<{ refactoredCode: string; diff: string }>("/api/v1/ai/refactor", {
        method: "POST",
        body: JSON.stringify({ code, instruction }),
      }),
  });
}

// Search API hooks
export function useSearchComponents(query: string) {
  return useQuery({
    queryKey: ["search", query],
    queryFn: () =>
      fetchAPI<Component[]>(`/api/v1/components/search?query=${encodeURIComponent(query)}`),
    enabled: query.length > 0,
  });
}

// Health check
export function useHealth() {
  return useQuery({
    queryKey: ["health"],
    queryFn: () =>
      fetchAPI<{ status: string; service: string; timestamp: string }>("/health"),
    refetchInterval: 30000,
  });
}
