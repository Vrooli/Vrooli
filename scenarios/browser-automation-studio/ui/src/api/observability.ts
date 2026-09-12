import { createClient } from '@connectrpc/connect';
import type { JsonObject } from '@bufbuild/protobuf';
import { ObservabilityService } from '@vrooli/proto-types/browser-automation-studio/v1/observability/observability_pb';
import { transport } from './client';

export const observabilityClient = createClient(ObservabilityService, transport);

// =============================================================================
// Proto ↔ legacy-shape mappers
// =============================================================================
//
// The observability surface proxies byte-for-byte to playwright-driver, whose
// payload shapes are owned downstream. We round-trip them through
// google.protobuf.Struct in the proto layer; in the TS bindings that surfaces
// as a JsonObject which can be cast to the legacy TypeScript interfaces in
// `domains/observability/types` without further translation.

const asObject = <T,>(value: JsonObject | undefined): T => (value ?? {}) as T;

const toJsonObject = (value: unknown): JsonObject => {
  if (value === undefined || value === null) return {};
  return value as JsonObject;
};

export const observability = {
  async get<T>(depth: string, noCache: boolean): Promise<T> {
    const resp = await observabilityClient.getObservability({ depth, noCache });
    return asObject<T>(resp.snapshot);
  },

  async refresh<T = unknown>(): Promise<T> {
    const resp = await observabilityClient.refreshObservability({});
    return asObject<T>(resp.result);
  },

  async runDiagnostics<T = unknown>(options: unknown): Promise<T> {
    const resp = await observabilityClient.runDiagnostics({ options: toJsonObject(options) });
    return asObject<T>(resp.result);
  },

  async listSessions<T = unknown>(): Promise<T> {
    const resp = await observabilityClient.getSessionList({});
    return asObject<T>(resp.result);
  },

  async runCleanup<T = unknown>(): Promise<T> {
    const resp = await observabilityClient.runCleanup({});
    return asObject<T>(resp.result);
  },

  async getMetrics<T = unknown>(): Promise<T> {
    const resp = await observabilityClient.getMetrics({});
    return asObject<T>(resp.result);
  },

  async runPipelineTest<T = unknown>(options: unknown): Promise<T> {
    const resp = await observabilityClient.runPipelineTest({ options: toJsonObject(options) });
    return asObject<T>(resp.result);
  },

  async getConfigRuntime<T = unknown>(): Promise<T> {
    const resp = await observabilityClient.getConfigRuntime({});
    return asObject<T>(resp.result);
  },

  async updateConfig<T = unknown>(envVar: string, value: string): Promise<T> {
    const resp = await observabilityClient.updateConfig({ envVar, value });
    return asObject<T>(resp.result);
  },

  async resetConfig<T = unknown>(envVar: string): Promise<T> {
    const resp = await observabilityClient.resetConfig({ envVar });
    return asObject<T>(resp.result);
  },

  async getDebugMode() {
    return observabilityClient.getDebugMode({});
  },

  async setDebugMode(enabled: boolean, components: string[] = [], durationMinutes = 0) {
    return observabilityClient.setDebugMode({ enabled, components, durationMinutes });
  },
};
