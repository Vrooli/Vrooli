import { createClient } from "@connectrpc/connect";
import { IntegrationsService, type ProviderConnection } from "@vrooli/proto-types/personal-planner/v1/integrations/integrations_pb";

import { transport } from "./client";

export const integrationsClient = createClient(IntegrationsService, transport);
export async function fetchConnections(): Promise<ProviderConnection[]> { const response = await integrationsClient.listConnections({}); return response.connections; }
export async function createFixtureConnection(displayName = ""): Promise<ProviderConnection> { const response = await integrationsClient.createFixtureConnection({ displayName }); if (!response.connection) throw new Error("connection was not returned"); return response.connection; }
export async function syncConnection(connection: ProviderConnection): Promise<ProviderConnection> { const response = await integrationsClient.syncConnection({ id: connection.id, expectedRevision: connection.revision }); if (!response.connection) throw new Error("connection was not returned"); return response.connection; }
export async function disconnectConnection(connection: ProviderConnection): Promise<ProviderConnection> { const response = await integrationsClient.disconnectConnection({ id: connection.id, expectedRevision: connection.revision }); if (!response.connection) throw new Error("connection was not returned"); return response.connection; }
export type { ProviderConnection };
