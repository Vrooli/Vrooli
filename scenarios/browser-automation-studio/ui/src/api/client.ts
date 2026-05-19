import { resolveApiBase, createScenarioConnectTransport } from '@vrooli/api-base';

/**
 * Shared Connect-Web transport for every generated proto client in BAS.
 *
 * Connect-RPC services are mounted at the chi root (see api/main.go
 * `connectx.RegisterChi`) — NOT under /api/v1 — so the transport uses
 * the bare API base URL (no suffix).
 */
export const API_BASE = resolveApiBase();

export const transport = createScenarioConnectTransport({ baseUrl: API_BASE });
