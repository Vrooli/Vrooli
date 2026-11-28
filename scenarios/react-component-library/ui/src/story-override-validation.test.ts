import { describe, expect, it } from "vitest";

import harnessSource from "../../api/handlers/preview/assets/harness.js?raw";

type StoryInput = {
  args: {
    fields: Array<Record<string, unknown>>;
  };
  props: Record<string, unknown>;
};

type ValidationError = { path: string; rule: string; message: string };
type ValidateOverride = (override: Record<string, unknown>) => ValidationError[];

function validatorFor(story: StoryInput): ValidateOverride {
  const start = harnessSource.indexOf("const valueAtPath = ");
  const end = harnessSource.indexOf("const hookFixture = ", start);
  expect(start, "harness override validator should exist").not.toBe(-1);
  expect(end, "harness override validator boundary should exist").not.toBe(-1);
  const declarations = harnessSource.slice(start, end);
  // eslint-disable-next-line @typescript-eslint/no-implied-eval -- compiles the actual isolated validation declarations for behavior tests
  const createValidator = new Function(
    "previewStory",
    `${declarations}; return validateOverride;`,
  ) as (previewStory: StoryInput) => ValidateOverride;
  return createValidator(story);
}

const story: StoryInput = {
  args: {
    fields: [
      { path: "label", kind: "text", required: true, minLength: 3 },
      { path: "size", kind: "enum", options: ["small", "large"] },
      { path: "count", kind: "number", minimum: 0, maximum: 4 },
    ],
  },
  props: { label: "Ready", size: "small", count: 2 },
};

describe("preview story override validation", () => {
  it("accepts overrides that satisfy the story's declared fields", () => {
    const validateOverride = validatorFor(story);

    expect(validateOverride({ label: "Go!", size: "large", count: 3 })).toEqual([]);
  });

  it("rejects type, enum, and range violations before rendering", () => {
    const validateOverride = validatorFor(story);

    expect(validateOverride({ label: 3, size: "medium", count: 8 })).toEqual([
      { path: "label", rule: "text", message: "Must be text." },
      { path: "size", rule: "enum", message: "Must be one of the declared options." },
      { path: "count", rule: "maximum", message: "Above the maximum." },
    ]);
  });

  it("rejects required fields that are removed by an override", () => {
    const validateOverride = validatorFor(story);

    expect(validateOverride({ label: undefined })).toEqual([
      { path: "label", rule: "required", message: "A value is required." },
    ]);
  });
});
