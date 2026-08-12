import { create, fromJson, } from "@bufbuild/protobuf";
import { resolveApiBase, createScenarioConnectTransport } from "@vrooli/api-base";
import { ErrorEnvelopeSchema, } from "@vrooli/proto-types/web-console/v1/errors/errors_pb";
export const API_BASE = resolveApiBase();
const PROTO_READ_OPTIONS = { ignoreUnknownFields: true };
// transport is the shared Connect-Web transport every domain client
// imports. Routed at API_BASE (no /api/v1 suffix — Connect procedures
// live at /vrooli.web_console.v1.<domain>.<Service>/<RPC>).
export const transport = createScenarioConnectTransport({ baseUrl: API_BASE });
/**
 * Typed error thrown when the API returns a non-2xx response. The
 * server-side ErrorEnvelope round-trips through here so callers branch on
 * structured code/status instead of parsing strings.
 */
export class ApiError extends Error {
    constructor(envelope, status) {
        super(`${envelope.code}: ${envelope.message}`);
        this.name = "ApiError";
        this.code = envelope.code;
        this.status = status;
    }
}
export function makeApiError(code, message, status = 500) {
    const envelope = create(ErrorEnvelopeSchema, { code, message });
    return new ApiError(envelope, status);
}
export async function decodeApiError(res) {
    let envelope;
    try {
        const json = (await res.json());
        envelope = fromJson(ErrorEnvelopeSchema, json, PROTO_READ_OPTIONS);
    }
    catch {
        envelope = create(ErrorEnvelopeSchema, {
            code: "internal",
            message: `unexpected ${res.status} response (no envelope)`,
        });
    }
    return new ApiError(envelope, res.status);
}
export { fromJson, PROTO_READ_OPTIONS };
