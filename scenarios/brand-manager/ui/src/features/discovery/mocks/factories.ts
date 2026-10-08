/**
 * Test data factories for the discovery domain. Co-located with the feature so
 * deleting `features/discovery/` takes the factories with it (no central
 * residue).
 *
 * The response/source/draft types are GENERATED proto messages; factories use
 * `create(<Schema>, overrides)` so field defaults match proto3 semantics and
 * adding a proto field is instantly available without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  DiscoveryResultSchema,
  DiscoverySourceSchema,
  DraftBrandSchema,
  DraftColorsSchema,
  DraftIdentitySchema,
  type DiscoveryResult,
  type DiscoverySource,
  type DraftBrand,
  type DraftColors,
  type DraftIdentity,
} from "@vrooli/proto-types/brand-manager/v1/discovery/discovery_pb";

export type { DiscoveryResult, DiscoverySource, DraftBrand, DraftColors, DraftIdentity };

export const makeDiscoverySource = (
  overrides: MessageInitShape<typeof DiscoverySourceSchema> = {},
): DiscoverySource =>
  create(DiscoverySourceSchema, {
    file: ".vrooli/branding.json",
    type: "branding_json",
    confidence: 0.6,
    fields: 2,
    ...overrides,
  });

// Nested message fields must be constructed (not plain objects) so the proto
// $typeName brand is present — bare `{ displayName }` fails MessageInitShape.
export const makeDraftIdentity = (
  overrides: MessageInitShape<typeof DraftIdentitySchema> = {},
): DraftIdentity => create(DraftIdentitySchema, { displayName: "Acme", ...overrides });

export const makeDraftColors = (
  overrides: MessageInitShape<typeof DraftColorsSchema> = {},
): DraftColors => create(DraftColorsSchema, { primary: "#112233", ...overrides });

export const makeDraftBrand = (overrides: MessageInitShape<typeof DraftBrandSchema> = {}): DraftBrand =>
  create(DraftBrandSchema, {
    name: "web-console",
    identity: makeDraftIdentity(),
    colors: makeDraftColors(),
    ...overrides,
  });

export const makeDiscoveryResult = (
  overrides: MessageInitShape<typeof DiscoveryResultSchema> = {},
): DiscoveryResult =>
  create(DiscoveryResultSchema, {
    scenario: "web-console",
    sources: [],
    confidence: 0,
    suggestions: [],
    ...overrides,
  });
