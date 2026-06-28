import { createClient } from "@connectrpc/connect";
import {
  DesignService,
  type GenerateDesignLanguageResponse,
} from "@vrooli/proto-types/brand-manager/v1/design/design_pb";

import { transport } from "./client";

export const designClient = createClient(DesignService, transport);

/**
 * generateDesignLanguage renders a brand as a canonical DESIGN.md document and
 * returns the markdown. It is read-only — the server reads the brand and writes
 * nothing. The document is the payload, so the UI surfaces it directly for copy
 * or download.
 */
export async function generateDesignLanguage(input: {
  brandId: string;
}): Promise<GenerateDesignLanguageResponse> {
  return designClient.generateDesignLanguage({ brandId: input.brandId });
}

export type { GenerateDesignLanguageResponse };
