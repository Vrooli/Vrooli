import { proxyToApi, buildWebSocketUrl } from "../utils.js";

/**
 * Create a new console session via the API.
 * @param {unknown} payload
 * @returns {Promise<Response>}
 */
export function createSessionRequest(payload) {
  return proxyToApi("/api/v1/sessions", {
    method: "POST",
    json: payload,
  });
}

/**
 * Delete a session by ID.
 * @param {string} sessionId
 * @returns {Promise<Response>}
 */
export function deleteSessionRequest(sessionId) {
  return proxyToApi(`/api/v1/sessions/${sessionId}`, {
    method: "DELETE",
  });
}

/**
 * Fetch session details.
 * @param {string} sessionId
 * @returns {Promise<Response>}
 */
export function fetchSessionRequest(sessionId) {
  return proxyToApi(`/api/v1/sessions/${sessionId}`);
}

/**
 * Fetch session transcript contents.
 * @param {string} sessionId
 * @param {{ range?: string }} [options]
 * @returns {Promise<Response>}
 */
export function fetchSessionTranscriptRequest(sessionId, options = {}) {
  const headers = options.range ? { Range: options.range } : undefined;
  return proxyToApi(`/api/v1/sessions/${sessionId}/transcript`, {
    headers,
  });
}

/**
 * Open the WebSocket stream for a session.
 * @param {string} sessionId
 * @returns {WebSocket}
 */
export function openSessionStream(sessionId) {
  const url = buildWebSocketUrl(`/ws/sessions/${sessionId}/stream`);
  return new WebSocket(url);
}
