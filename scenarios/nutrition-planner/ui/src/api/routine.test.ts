import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { OccurrenceSchema, TemplateSchema } from "@vrooli/proto-types/nutrition-planner/v1/routine/routine_pb";

const listTemplates = vi.hoisted(() => vi.fn());
const createTemplate = vi.hoisted(() => vi.fn());
const generateOccurrences = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ listTemplates, createTemplate, generateOccurrences }) }));
vi.mock("./client", () => ({ transport: {} }));

import { createRoutineTemplate, generateRoutineOccurrences, listRoutineTemplates } from "./routine";

describe("routine API", () => {
  beforeEach(() => vi.clearAllMocks());

  it("lists templates and preserves generated values", async () => {
    const template = create(TemplateSchema, { slotName: "breakfast", quantity: "1" });
    listTemplates.mockResolvedValue({ templates: [template] });
    await expect(listRoutineTemplates("w1")).resolves.toEqual([template]);
  });

  it("creates a template with an explicit open-slot default", async () => {
    const template = create(TemplateSchema, { slotName: "breakfast", quantity: "1", mode: "open" });
    createTemplate.mockResolvedValue({ template });
    await expect(createRoutineTemplate({ workspaceId: "w1", slotName: "breakfast", quantity: "1", weekdays: [1], startDate: "2026-09-18", mode: "open", active: true })).resolves.toBe(template);
    expect(createTemplate).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", recipeId: "", endDate: "" }));
  });

  it("generates occurrences and rejects a hollow create response", async () => {
    const occurrence = create(OccurrenceSchema, { date: "2026-09-18", slotName: "breakfast" });
    generateOccurrences.mockResolvedValue({ occurrences: [occurrence] });
    await expect(generateRoutineOccurrences({ workspaceId: "w1", fromDate: "2026-09-18", toDate: "2026-09-18" })).resolves.toEqual([occurrence]);
    createTemplate.mockResolvedValue({});
    await expect(createRoutineTemplate({ workspaceId: "w1", slotName: "breakfast", quantity: "1", weekdays: [1], startDate: "2026-09-18", mode: "open", active: true })).rejects.toThrow("no routine template");
  });
});
