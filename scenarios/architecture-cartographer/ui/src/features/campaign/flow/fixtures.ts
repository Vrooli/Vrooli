/**
 * Campaign fixtures for unit + integration tests. Pure data — no mocks here.
 *
 * Builders produce proto-shaped messages with sane defaults; callers override
 * only the fields they care about.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  CampaignItemStatus,
  CampaignLifecycle,
  CampaignSchema,
  CampaignStatusSchema,
  CampaignItemSchema,
  type Campaign,
  type CampaignStatus,
  type CampaignItem,
} from "@vrooli/proto-types/architecture-cartographer/v1/campaign/campaign_pb";

export const makeCampaign = (
  overrides: MessageInitShape<typeof CampaignSchema> = {},
): Campaign =>
  create(CampaignSchema, {
    id: "m-1",
    scenario: "demo-scenario",
    name: "big-refactor",
    status: CampaignLifecycle.OPEN,
    ...overrides,
  });

export const makeCampaignItem = (
  overrides: MessageInitShape<typeof CampaignItemSchema> = {},
): CampaignItem =>
  create(CampaignItemSchema, {
    stableId: "afid:abc12345",
    scenario: "demo-scenario",
    source: "architecture",
    code: "cycle/cross-domain",
    severity: "blocker",
    locations: ["api/a", "api/b"],
    domains: ["a", "b"],
    message: "import cycle between a and b",
    status: CampaignItemStatus.DETECTED,
    ...overrides,
  });

export const makeCampaignStatus = (
  overrides: MessageInitShape<typeof CampaignStatusSchema> = {},
): CampaignStatus =>
  create(CampaignStatusSchema, {
    campaign: makeCampaign(),
    items: [makeCampaignItem()],
    total: 1,
    open: 1,
    resolved: 0,
    validated: 0,
    regressions: 0,
    ...overrides,
  });
