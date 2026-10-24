/**
 * Canonical persistence boundary for preflight stage evidence.
 *
 * Scenario state is intentionally stored as a protobuf Struct to support
 * drafts. This codec preserves the generated PreflightResponse vocabulary
 * through that JSON-only boundary; application state never uses a shadow DTO.
 */
import { fromJson, toJson, type JsonObject } from "@bufbuild/protobuf";
import {
  PreflightResponseSchema,
  type PreflightResponse,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";

export function serializePreflight(result: PreflightResponse): unknown {
  return toJson(PreflightResponseSchema, result);
}

export function deserializePreflight(value: unknown): PreflightResponse | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  try {
    return fromJson(PreflightResponseSchema, value as JsonObject);
  } catch {
    return null;
  }
}
