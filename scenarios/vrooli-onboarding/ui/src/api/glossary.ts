import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import {
  GlossaryService,
  type SearchGlossaryResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/glossary/glossary_pb";

const client = createClient(GlossaryService, onboardingTransport());

export function fetchGlossary(query = ""): Promise<SearchGlossaryResponse> {
  return client.searchGlossary({ query }) as unknown as Promise<SearchGlossaryResponse>;
}
