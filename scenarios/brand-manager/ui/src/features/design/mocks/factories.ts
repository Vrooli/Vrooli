/**
 * Test data factories for the design domain. Co-located with the feature so
 * deleting `features/design/` takes the factories with it (no central residue).
 *
 * The response type is a GENERATED proto message; the factory uses
 * `create(<Schema>, overrides)` so field defaults match proto3 semantics and
 * adding a proto field is instantly available without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  GenerateDesignLanguageResponseSchema,
  type GenerateDesignLanguageResponse,
} from "@vrooli/proto-types/brand-manager/v1/design/design_pb";

export type { GenerateDesignLanguageResponse };

export const makeDesignResponse = (
  overrides: MessageInitShape<typeof GenerateDesignLanguageResponseSchema> = {},
): GenerateDesignLanguageResponse =>
  create(GenerateDesignLanguageResponseSchema, {
    brandId: "brand-1",
    markdown: "---\nid: brand-1\nname: \"Acme\"\nversion: 1\nsource: brand-manager\n---\n\n# Acme DESIGN.md\n",
    ...overrides,
  });
