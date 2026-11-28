import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import {
  GetReadinessRequestSchema,
  ListProvidersRequestSchema,
  SetupService,
} from "@vrooli/proto-types/compute-manager/v1/setup/setup_pb";

import { transport } from "./client";

const setup = createClient(SetupService, transport);

export async function fetchProviderCatalogue() {
  const { providers } = await setup.listProviders(create(ListProvidersRequestSchema));
  const readiness = await Promise.all(providers.map(async (provider) => {
    const result = await setup.getReadiness(create(GetReadinessRequestSchema, { providerId: provider.id }));
    return [provider.id, result.readiness] as const;
  }));
  return { providers, readiness: Object.fromEntries(readiness) };
}
