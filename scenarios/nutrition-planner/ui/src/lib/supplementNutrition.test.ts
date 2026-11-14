import { create } from "@bufbuild/protobuf";
import { CatalogRevisionSchema, NutrientValueSchema } from "@vrooli/proto-types/nutrition-planner/v1/catalog/catalog_pb";
import { describe, expect, it } from "vitest";
import type { SupplementSchedule } from "../api/supplement";
import { catalogRevisionReference, scheduledSupplementNutrients } from "./supplementNutrition";

const schedule = (productRevisionId: string, dose: string, doseUnit = "IU"): SupplementSchedule => ({ id: "s1", revision: 2n, productRevisionId, dose, doseUnit, weekdays: [1], startDate: "2026-10-01", endDate: "", paused: false, confirmed: true, createdAt: "" });
const product = (amount: string, basis: string, basisUnit = "IU") => create(CatalogRevisionSchema, { id: "vitamin-d", revision: 3n, name: "Vitamin D", productName: "Vitamin D3", servingQuantity: "1000", servingUnit: "IU", nutrients: [create(NutrientValueSchema, { nutrientId: "vitamin_d", amount, unit: "mcg", basis, basisUnit, evidence: "label", sourceRef: "label:v3" })] });

describe("scheduled supplement nutrient evidence", () => {
  it("resolves a pinned catalog revision and scales label amounts exactly", () => {
    const revision = product("25", "1000");
    const reference = catalogRevisionReference(revision);
    expect(reference).toBe("catalog:vitamin-d:3");
    expect(scheduledSupplementNutrients(schedule(reference, "1000"), [revision])).toEqual([{ productName: "Vitamin D3", nutrientId: "vitamin_d", amount: "25", unit: "mcg", evidence: "label", unresolvedReason: "" }]);
  });

  it("formats exact fifths as finite decimals", () => {
    const revision = product("1", "5");
    const [value] = scheduledSupplementNutrients(schedule(catalogRevisionReference(revision), "1"), [revision]);
    expect(value?.amount).toBe("0.2");
  });

  it("scales decimal doses and bases without floating point drift", () => {
    const revision = product("12.5", "2.5");
    const [value] = scheduledSupplementNutrients(schedule(catalogRevisionReference(revision), "0.5"), [revision]);
    expect(value).toMatchObject({ amount: "2.5", unit: "mcg", unresolvedReason: "" });
  });

  it("keeps incomplete nutrient assertions unresolved", () => {
    const revision = create(CatalogRevisionSchema, { id: "vitamin-d", revision: 3n, name: "Vitamin D", productName: "Vitamin D3", nutrients: [create(NutrientValueSchema, { nutrientId: "vitamin_d", amount: "", unit: "mcg", basis: "1", basisUnit: "IU", evidence: "label" })] });
    expect(scheduledSupplementNutrients(schedule(catalogRevisionReference(revision), "1000"), [revision])[0]?.unresolvedReason).toBe("catalog contribution is incomplete");
  });

  it("keeps unpinned references, missing revisions, and incompatible units unresolved", () => {
    const revision = product("25", "1000");
    const ref = catalogRevisionReference(revision);
    expect(scheduledSupplementNutrients(schedule("vitamin-d@revision-3", "1000"), [revision])[0]?.unresolvedReason).toContain("not linked");
    expect(scheduledSupplementNutrients(schedule("catalog:vitamin-d:2", "1000"), [revision])[0]?.unresolvedReason).toContain("unavailable");
    expect(scheduledSupplementNutrients(schedule(ref, "1", "capsule"), [revision])[0]?.unresolvedReason).toContain("does not match");
  });

  it("falls back to a catalog name when product name is absent and keeps empty nutrient profiles unknown", () => {
    const revision = create(CatalogRevisionSchema, { id: "plain", revision: 1n, name: "Plain product", productName: "", servingQuantity: "1", servingUnit: "capsule", nutrients: [] });
    const [value] = scheduledSupplementNutrients(schedule(catalogRevisionReference(revision), "1", "capsule"), [revision]);
    expect(value).toMatchObject({ productName: "Plain product", unresolvedReason: "catalog revision has no nutrient assertions; contributions remain unknown" });
  });

  it("keeps malformed references and unscalable catalog values unresolved", () => {
    const revision = product("not-a-number", "1");
    expect(scheduledSupplementNutrients(schedule("catalog::2", "1"), [revision])[0]?.unresolvedReason).toContain("not linked");
    expect(scheduledSupplementNutrients(schedule(`catalog:long-id:${"1".repeat(20)}`, "1"), [revision])[0]?.unresolvedReason).toContain("not linked");
    expect(scheduledSupplementNutrients(schedule(catalogRevisionReference(revision), "1"), [revision])[0]?.unresolvedReason).toContain("cannot be scaled exactly");
    const zeroBasis = product("1", "0");
    expect(scheduledSupplementNutrients(schedule(catalogRevisionReference(zeroBasis), "1"), [zeroBasis])[0]?.unresolvedReason).toContain("cannot be scaled exactly");
    const tinyResult = product("0.000000000001", "1");
    expect(scheduledSupplementNutrients(schedule(catalogRevisionReference(tinyResult), "0.000000000001"), [tinyResult])[0]?.unresolvedReason).toContain("cannot be scaled exactly");
    const oversized = product("1".repeat(65), "1");
    expect(scheduledSupplementNutrients(schedule(catalogRevisionReference(oversized), "1"), [oversized])[0]?.unresolvedReason).toContain("cannot be scaled exactly");
    const longFraction = product("1", "0.0000000000001");
    expect(scheduledSupplementNutrients(schedule(catalogRevisionReference(longFraction), "1"), [longFraction])[0]?.unresolvedReason).toContain("cannot be scaled exactly");
  });

  it("does not report an inexact repeating decimal as a known amount", () => {
    const revision = product("1", "3");
    const [value] = scheduledSupplementNutrients(schedule(catalogRevisionReference(revision), "1"), [revision]);
    expect(value?.amount).toBe("");
    expect(value?.unresolvedReason).toContain("cannot be scaled exactly");
  });
});
