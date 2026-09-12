/**
 * Shared type helpers used across domain type files.
 */

import type { Message } from "@bufbuild/protobuf";

export type ProtoMessage<T extends Message> = Omit<T, "$typeName" | "$unknown">;

export interface PlanRef {
  provider: "plan-manager";
  planId: string;
  slug: string;
  role: "execution_spec" | "operating_mode_plan";
}
