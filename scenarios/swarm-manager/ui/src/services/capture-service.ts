/**
 * Capture Service - Data access layer for quick-capture operations
 *
 * This service encapsulates all capture-related API operations behind a clean seam.
 * It accepts an API client as a dependency, making it easy to substitute for testing.
 *
 * DOC: docs/internal/SEAMS.md
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Capture, CaptureClassification } from "../types";

export interface ClassifyResponse {
  taskId: string;
  runId: string;
  baseUrl: string;
  created: string;
}

export interface CreateCaptureResponse {
  capture: Capture;
  taskId?: string;
  runId?: string;
  baseUrl?: string;
}

export interface ICaptureService {
  list(): Promise<Capture[]>;
  get(id: string): Promise<Capture>;
  create(text: string): Promise<CreateCaptureResponse>;
  remove(id: string): Promise<void>;
  classify(id: string): Promise<ClassifyResponse>;
}

function mapCapture(raw: Record<string, unknown>): Capture {
  const cls = raw.classification as Record<string, unknown> | null;
  return {
    id: (raw.id as string) ?? "",
    text: (raw.text as string) ?? "",
    attachments: (raw.attachments as string[]) ?? [],
    created: (raw.created as string) ?? "",
    status: (raw.status as Capture["status"]) ?? "classifying",
    classification: cls
      ? {
          items: ((cls.items as Record<string, unknown>[]) ?? []).map((item) => ({
            kind: (item.kind as Capture["classification"] extends infer C ? C extends CaptureClassification ? C["items"][number]["kind"] : never : never) ?? "idea",
            title: (item.title as string) ?? "",
            description: (item.description as string) ?? "",
            priority: (item.priority as number) ?? 5,
            tags: (item.tags as string[]) ?? [],
            confidence: (item.confidence as number) ?? 0,
          })),
          classifiedAt: (cls.classified_at as string) ?? (cls.classifiedAt as string) ?? "",
        }
      : null,
  };
}

export function createCaptureService(apiClient: IApiClient = defaultApiClient): ICaptureService {
  return {
    async list(): Promise<Capture[]> {
      const data = await apiClient.get<{ captures: Record<string, unknown>[] }>(API_ENDPOINTS.captures);
      return (data.captures ?? []).map(mapCapture);
    },

    async get(id: string): Promise<Capture> {
      const data = await apiClient.get<{ capture: Record<string, unknown> }>(API_ENDPOINTS.captureById(id));
      return mapCapture(data.capture);
    },

    async create(text: string): Promise<CreateCaptureResponse> {
      const data = await apiClient.post<{
        capture: Record<string, unknown>;
        task_id?: string;
        run_id?: string;
        base_url?: string;
      }>(API_ENDPOINTS.captures, { text });
      return {
        capture: mapCapture(data.capture),
        taskId: data.task_id,
        runId: data.run_id,
        baseUrl: data.base_url,
      };
    },

    async remove(id: string): Promise<void> {
      await apiClient.delete(API_ENDPOINTS.captureById(id));
    },

    async classify(id: string): Promise<ClassifyResponse> {
      const data = await apiClient.post<{
        task_id: string;
        run_id: string;
        base_url: string;
        created: string;
      }>(API_ENDPOINTS.captureClassify(id), {});
      return {
        taskId: data.task_id,
        runId: data.run_id,
        baseUrl: data.base_url,
        created: data.created,
      };
    },
  };
}

export const captureService = createCaptureService();
