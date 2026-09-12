import { createClient } from "@connectrpc/connect";
import {
  PairingService,
  type IssuePairingCodeResponse,
} from "@vrooli/proto-types/vrooli-bridge/v1/pairing/pairing_pb";

import { transport } from "./client";

/**
 * Typed client for the PairingService — one-touch node onboarding
 * (pairing domain, OT-P0-002). IssuePairingCode (owner-gated) mints a
 * single-use plaintext code returned ONCE alongside the control-plane public
 * key; it is delivered out-of-band to the node's bootstrap installer. The
 * fleet dashboard exposes this so an operator can pair a node without leaving
 * the surface.
 */
export const pairingClient = createClient(PairingService, transport);

export interface PairingRequest {
  id: string;
  name: string;
  os: string;
  arch: string;
  endpoint: string;
  confirmationWords: string[];
}

export interface PermissionPreset {
  name: string;
  description: string;
  scopes: string[];
  withholds: string[];
}

export type { IssuePairingCodeResponse };
