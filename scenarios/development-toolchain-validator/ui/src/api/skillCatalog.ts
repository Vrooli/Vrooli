import { createClient } from "@connectrpc/connect";
import {
  SkillCatalogService,
  type Skill,
  type ListSkillsResponse,
  type SyncResponse,
} from "@vrooli/proto-types/development-toolchain-validator/v1/skill_catalog/skill_catalog_pb";

import { transport } from "./client";

export const skillCatalogClient = createClient(SkillCatalogService, transport);

export type { Skill, ListSkillsResponse, SyncResponse };
