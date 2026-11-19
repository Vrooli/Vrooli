import type { Rule } from "eslint";

declare const usageContractRule: Rule.RuleModule;
export default usageContractRule;
export function hasInvalidOptionalEnumValue(
  value: unknown,
  allowedValues: readonly string[],
): boolean;
