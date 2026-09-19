import { createClient } from "@connectrpc/connect";
import {
  ScenarioValidationService,
  type ValidateTargetResponse,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";
import {
  ValidationTargetKind,
  ValidationTargetSchema,
} from "@vrooli/proto-types/common/v1/validation_target_pb";
import { create } from "@bufbuild/protobuf";

import { transport } from "./client";

export const TARGET_KINDS = [
  { value: ValidationTargetKind.SCENARIO, label: "scenario" },
  { value: ValidationTargetKind.RESOURCE, label: "resource" },
  { value: ValidationTargetKind.TOOL, label: "tool" },
  { value: ValidationTargetKind.SAFEGUARD, label: "safeguard" },
  { value: ValidationTargetKind.PACKAGE, label: "package" },
  { value: ValidationTargetKind.CONTROL_PLANE, label: "control-plane" },
  { value: ValidationTargetKind.DOCS, label: "docs" },
  { value: ValidationTargetKind.TEAM, label: "team" },
  { value: ValidationTargetKind.PROJECT, label: "project" },
] as const;

export type TargetKindValue = (typeof TARGET_KINDS)[number]["value"];

const client = createClient(ScenarioValidationService, transport);

export const validationClient = {
  validateTarget: (input: {
    kind: TargetKindValue;
    id: string;
    root?: string;
    path?: string;
  }): Promise<ValidateTargetResponse> =>
    client.validateTarget({
      target: create(ValidationTargetSchema, {
        kind: input.kind,
        id: input.id,
        root: input.root ?? "",
      }),
      includeExecution: false,
      path: input.path ?? "",
    }),
};

export { ValidationTargetKind };
export type { ValidateTargetResponse };
