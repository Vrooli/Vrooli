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
import type { Capture, CaptureClassification, CaptureFailureReason } from "../types";

export interface ClassifyResponse {
  workflowExecutionId: string;
  workflowDefinitionDigest: string;
  created: string;
}

export interface CreateCaptureResponse {
  capture: Capture;
  workflowExecutionId?: string;
  workflowDefinitionDigest?: string;
}

export interface ICaptureService {
  list(): Promise<Capture[]>;
  get(id: string): Promise<Capture>;
  create(text: string, files?: File[]): Promise<CreateCaptureResponse>;
  remove(id: string): Promise<void>;
  classify(id: string): Promise<ClassifyResponse>;
  applyClassification(id: string, executionId: string): Promise<Capture>;
  updateNote(id: string, note: string): Promise<Capture>;
}

function mapCapture(raw: Record<string, unknown>): Capture {
  const cls = raw.classification as Record<string, unknown> | null;
  return {
    id: (raw.id as string) ?? "",
    text: (raw.text as string) ?? "",
    attachments: (raw.attachments as string[]) ?? [],
    created: (raw.created as string) ?? "",
    status: (raw.status as Capture["status"]) ?? "classifying",
    failureReason: (raw.failure_reason as CaptureFailureReason) || undefined,
    workflowExecutionId: (raw.workflow_execution_id as string) || undefined,
    workflowDefinitionDigest: (raw.workflow_definition_digest as string) || undefined,
    workflowEntityVersion: (raw.workflow_entity_version as string) || undefined,
    note: (raw.note as string) ?? "",
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

    async create(text: string, files?: File[]): Promise<CreateCaptureResponse> {
      const formData = new FormData();
      formData.append("text", text);
      if (files) {
        for (const file of files) {
          formData.append("files", file);
        }
      }
      const data = await apiClient.post<{
        capture: Record<string, unknown>;
        workflow_execution_id?: string;
        workflow_definition_digest?: string;
      }>(API_ENDPOINTS.captures, formData);
      return {
        capture: mapCapture(data.capture),
        workflowExecutionId: data.workflow_execution_id,
        workflowDefinitionDigest: data.workflow_definition_digest,
      };
    },

    async remove(id: string): Promise<void> {
      await apiClient.delete(API_ENDPOINTS.captureById(id));
    },

    async classify(id: string): Promise<ClassifyResponse> {
      const data = await apiClient.post<{
        workflow_execution_id: string;
        workflow_definition_digest: string;
        created: string;
      }>(API_ENDPOINTS.captureClassify(id), {});
      return {
        workflowExecutionId: data.workflow_execution_id,
        workflowDefinitionDigest: data.workflow_definition_digest,
        created: data.created,
      };
    },

    async applyClassification(id: string, executionId: string): Promise<Capture> {
      const data = await apiClient.post<{ capture: Record<string, unknown> }>(
        API_ENDPOINTS.captureClassificationApply(id, executionId),
        {},
      );
      return mapCapture(data.capture);
    },

    async updateNote(id: string, note: string): Promise<Capture> {
      const data = await apiClient.patch<{ capture: Record<string, unknown> }>(
        API_ENDPOINTS.captureById(id),
        { note },
      );
      return mapCapture(data.capture);
    },
  };
}

export const captureService = createCaptureService();
