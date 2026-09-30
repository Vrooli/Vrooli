/**
 * Wizard test-data factories. Build real proto messages via `create()` so
 * tests exercise the same field shapes the connect client decodes at runtime.
 * Co-located with the feature so deleting the folder takes the doubles along.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  AnswerSchema,
  CapabilityHintSchema,
  QuestionSchema,
  ScaffoldPreviewSchema,
  ScaffoldResultSchema,
  SessionStateSchema,
} from "@vrooli/proto-types/business-health/v1/wizard/wizard_pb";
import type {
  Answer,
  CapabilityHint,
  Question,
  ScaffoldPreview,
  ScaffoldResult,
  SessionState,
} from "@vrooli/proto-types/business-health/v1/wizard/wizard_pb";

export const makeQuestion = (
  overrides: MessageInitShape<typeof QuestionSchema> = {},
): Question =>
  create(QuestionSchema, {
    id: "overview",
    target: "overview",
    prompt: "Summarize the business outcome this scenario delivers.",
    help: "One or two sentences, in plain language.",
    kind: "multiline",
    required: true,
    minEntries: 0,
    ...overrides,
  });

export const makeAnswer = (
  overrides: MessageInitShape<typeof AnswerSchema> = {},
): Answer =>
  create(AnswerSchema, {
    questionId: "overview",
    text: "Keeps every scenario's contract honest.",
    items: [],
    targets: [],
    invalidReason: "",
    ...overrides,
  });

export const makeCapabilityHint = (
  overrides: MessageInitShape<typeof CapabilityHintSchema> = {},
): CapabilityHint =>
  create(CapabilityHintSchema, {
    scenario: "scenario-auditor",
    capability: "structure validation",
    anchor: "scenarios/scenario-auditor/PRD.md#OT-P0-001",
    score: 0.82,
    ...overrides,
  });

export const makeSessionState = (
  overrides: MessageInitShape<typeof SessionStateSchema> = {},
): SessionState =>
  create(SessionStateSchema, {
    sessionId: "sess-1",
    scenario: "business-health",
    questions: [makeQuestion()],
    answers: {},
    remaining: ["overview"],
    complete: false,
    hints: [],
    ...overrides,
  });

export const makeScaffoldPreview = (
  overrides: MessageInitShape<typeof ScaffoldPreviewSchema> = {},
): ScaffoldPreview =>
  create(ScaffoldPreviewSchema, {
    sessionId: "sess-1",
    files: [
      { path: "scenarios/business-health/PRD.md", before: "", after: "# Business Health\n" },
    ],
    blocking: [],
    ...overrides,
  });

export const makeScaffoldResult = (
  overrides: MessageInitShape<typeof ScaffoldResultSchema> = {},
): ScaffoldResult =>
  create(ScaffoldResultSchema, {
    sessionId: "sess-1",
    written: ["scenarios/business-health/PRD.md"],
    residualFindings: [],
    ...overrides,
  });
