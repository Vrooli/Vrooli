// Version negotiation. The Go supervisor sends {type:"handshake",
// protocol_version, request_id}; the sidecar replies with its own
// protocol_version, sidecar version, and ts-morph version. The supervisor
// rejects mismatched major versions (handled Go-side).

import { createRequire } from "node:module";

import type { HandshakeRequest, HandshakeResponse } from "./protocol.js";
import { PROTOCOL_VERSION } from "./protocol.js";

const require = createRequire(import.meta.url);

let cachedTsMorphVersion: string | null = null;
function tsMorphVersion(): string {
  if (cachedTsMorphVersion !== null) return cachedTsMorphVersion;
  try {
    const pkg = require("ts-morph/package.json") as { version: string };
    cachedTsMorphVersion = pkg.version;
  } catch {
    cachedTsMorphVersion = "unknown";
  }
  return cachedTsMorphVersion;
}

let cachedSelfVersion: string | null = null;
function selfVersion(): string {
  if (cachedSelfVersion !== null) return cachedSelfVersion;
  try {
    const pkg = require("../package.json") as { version: string };
    cachedSelfVersion = pkg.version;
  } catch {
    cachedSelfVersion = "0.0.0";
  }
  return cachedSelfVersion;
}

export function handleHandshake(req: HandshakeRequest): HandshakeResponse {
  return {
    type: "handshake",
    request_id: req.request_id,
    protocol_version: PROTOCOL_VERSION,
    sidecar_version: selfVersion(),
    ts_morph_version: tsMorphVersion(),
  };
}
