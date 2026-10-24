import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import {
  GlossaryService,
  type SearchConfigurationResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/glossary/glossary_pb";

const client = createClient(GlossaryService, onboardingTransport());

export type ConfigurationDescriptor = SearchConfigurationResponse["results"][number];

export function searchConfiguration(query: string, target = "local"): Promise<SearchConfigurationResponse> {
  return client.searchConfiguration({ query, target }) as unknown as Promise<SearchConfigurationResponse>;
}
