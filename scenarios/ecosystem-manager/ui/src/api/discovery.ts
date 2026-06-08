import { createClient } from '@connectrpc/connect';
import { DiscoveryService } from '@vrooli/proto-types/ecosystem-manager/v1/discovery/discovery_pb';

import { transport } from './client';

/**
 * Typed Connect-RPC client for the discovery domain. Replaces the former
 * hand-rolled Zod parse layer + REST fetches (api.getResources/getScenarios).
 * Consumed by ui/src/stores/discoveryStore.ts.
 */
export const discoveryClient = createClient(DiscoveryService, transport);
