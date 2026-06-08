import { resolveApiBase, createScenarioConnectTransport } from '@vrooli/api-base';

/**
 * Shared Connect-RPC substrate for ecosystem-manager's migrated domains.
 *
 * Every migrated domain creates its typed client in ui/src/api/<domain>.ts via
 * `createClient(<Domain>Service, transport)` and consumes it from
 * ui/src/features/<domain>/ through react-query. The legacy hand-rolled Zod
 * layer (ui/src/lib/proto-contracts.ts) is deleted domain-by-domain as each
 * moves onto this transport. See docs/internal/COHERENCE-NOTES.md.
 */

export const API_BASE = resolveApiBase();

/** The single Connect transport all domain clients share. */
export const transport = createScenarioConnectTransport({ baseUrl: API_BASE });

/**
 * Typed error surfaced when an API call fails. Connect's own ConnectError
 * already carries a structured code+message; ApiError gives non-Connect
 * failures the same shape so call sites branch on `code`/`status` instead of
 * parsing strings.
 */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(code: string, message: string, status = 500) {
    super(`${code}: ${message}`);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}
