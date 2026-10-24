import type { CatalogRevision } from "../api/catalog";
import type { SupplementSchedule } from "../api/supplement";

export type ScheduledNutrient = {
  productName: string;
  nutrientId: string;
  amount: string;
  unit: string;
  evidence: string;
  unresolvedReason: string;
};

export function catalogRevisionReference(revision: Pick<CatalogRevision, "id" | "revision">): string {
  return `catalog:${revision.id}:${revision.revision}`;
}

function parseCatalogReference(reference: string): { id: string; revision: bigint } | undefined {
  if (!reference.startsWith("catalog:")) return undefined;
  const separator = reference.lastIndexOf(":");
  const id = reference.slice("catalog:".length, separator);
  const revision = reference.slice(separator + 1);
  if (!id || revision.length > 19 || !/^\d+$/.test(revision)) return undefined;
  return { id, revision: BigInt(revision) };
}

type Decimal = { coefficient: bigint; scale: number };
function parseDecimal(value: string): Decimal | undefined {
  const match = /^(\d+)(?:\.(\d+))?$/.exec(value.trim());
  if (!match || value.length > 64) return undefined;
  const fraction = match[2] ?? "";
  if (fraction.length > 12) return undefined;
  const coefficient = BigInt(`${match[1]}${fraction}`);
  return { coefficient, scale: fraction.length };
}
function gcd(left: bigint, right: bigint): bigint {
  let a = left;
  let b = right;
  while (b !== 0n) [a, b] = [b, a % b];
  return a;
}
function exactScaledAmount(amount: string, dose: string, basis: string): string | undefined {
  const parsedAmount = parseDecimal(amount);
  const parsedDose = parseDecimal(dose);
  const parsedBasis = parseDecimal(basis);
  if (!parsedAmount || !parsedDose || !parsedBasis || parsedBasis.coefficient <= 0n) return undefined;
  let numerator = parsedAmount.coefficient * parsedDose.coefficient * 10n ** BigInt(parsedBasis.scale);
  let denominator = parsedBasis.coefficient * 10n ** BigInt(parsedAmount.scale + parsedDose.scale);
  const common = gcd(numerator, denominator);
  numerator /= common;
  denominator /= common;
  let reduced = denominator;
  let twos = 0;
  let fives = 0;
  while (reduced % 2n === 0n) { reduced /= 2n; twos++; }
  while (reduced % 5n === 0n) { reduced /= 5n; fives++; }
  if (reduced !== 1n) return undefined;
  const scale = Math.max(twos, fives);
  if (scale > 12) return undefined;
  const scaled = numerator * (10n ** BigInt(scale)) / denominator;
  const digits = scaled.toString().padStart(scale + 1, "0");
  if (scale === 0) return digits;
  const whole = digits.slice(0, -scale);
  const fraction = digits.slice(-scale).replace(/0+$/, "");
  return `${whole}${fraction ? `.${fraction}` : ""}`;
}

export function scheduledSupplementNutrients(
  schedule: SupplementSchedule,
  catalog: CatalogRevision[],
): ScheduledNutrient[] {
  const parsed = parseCatalogReference(schedule.productRevisionId);
  if (!parsed) return [{ productName: schedule.productRevisionId, nutrientId: "", amount: "", unit: "", evidence: "", unresolvedReason: "catalog revision is not linked; nutrient amount remains unknown" }];
  const revision = catalog.find((item) => item.id === parsed.id && item.revision === parsed.revision);
  if (!revision) return [{ productName: schedule.productRevisionId, nutrientId: "", amount: "", unit: "", evidence: "", unresolvedReason: "pinned catalog revision is unavailable; nutrient amount remains unknown" }];
  const productName = revision.productName || revision.name;
  if (revision.nutrients.length === 0) return [{ productName, nutrientId: "", amount: "", unit: "", evidence: "", unresolvedReason: "catalog revision has no nutrient assertions; contributions remain unknown" }];
  return revision.nutrients.map((nutrient) => {
    if (!nutrient.amount || !nutrient.basis || !nutrient.unit || !nutrient.basisUnit) {
      return { productName, nutrientId: nutrient.nutrientId, amount: "", unit: nutrient.unit, evidence: nutrient.evidence, unresolvedReason: "catalog contribution is incomplete" };
    }
    if (schedule.doseUnit !== nutrient.basisUnit) {
      return { productName, nutrientId: nutrient.nutrientId, amount: "", unit: nutrient.unit, evidence: nutrient.evidence, unresolvedReason: `dose unit ${schedule.doseUnit} does not match nutrient basis ${nutrient.basisUnit}` };
    }
    const amount = exactScaledAmount(nutrient.amount, schedule.dose, nutrient.basis);
    return amount === undefined
      ? { productName, nutrientId: nutrient.nutrientId, amount: "", unit: nutrient.unit, evidence: nutrient.evidence, unresolvedReason: "dose cannot be scaled exactly from the catalog basis" }
      : { productName, nutrientId: nutrient.nutrientId, amount, unit: nutrient.unit, evidence: nutrient.evidence, unresolvedReason: "" };
  });
}
