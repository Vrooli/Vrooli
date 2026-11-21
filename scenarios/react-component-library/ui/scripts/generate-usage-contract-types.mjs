#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import prettier from "prettier";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");
const componentsRoot = path.join(repoRoot, "scenarios/react-component-library/library/components");
const outputPath = path.join(
  repoRoot,
  "scenarios/react-component-library/library/usage-contracts/generated-contract-limits.ts",
);
const checkOnly = process.argv.includes("--check");

const contracts = [];
const activeContracts = [];
for (const component of fs.readdirSync(componentsRoot).sort()) {
  const versionsRoot = path.join(componentsRoot, component, "versions");
  if (!fs.existsSync(versionsRoot)) continue;
  const metadata = JSON.parse(
    fs.readFileSync(path.join(componentsRoot, component, "component.json"), "utf8"),
  );
  const activeVersion = metadata.draft || metadata.latest;
  for (const version of fs.readdirSync(versionsRoot).sort()) {
    const contractPath = path.join(versionsRoot, version, "usage-contract.json");
    if (!fs.existsSync(contractPath)) continue;
    const contract = JSON.parse(fs.readFileSync(contractPath, "utf8"));
    const items = contract.items ?? {};
    const key = `react-component-library:${component}@${version}`;
    const entry = {
      key,
      maximumChildren: Number.isInteger(contract.children?.maximumCount)
        ? contract.children.maximumCount
        : null,
      maximumItems: Number.isInteger(items.maximumCount) ? items.maximumCount : null,
      maximumLabelCharacters: Number.isInteger(items.maximumLabelCharacters)
        ? items.maximumLabelCharacters
        : null,
      primaryCount: Number.isInteger(items.primaryCount) ? items.primaryCount : null,
      itemsProperty: typeof items.property === "string" ? items.property : null,
      requiredFields: [
        ...new Set([
          ...(Array.isArray(items.requiredFields) ? items.requiredFields : []),
          ...(typeof items.stableIdField === "string" ? [items.stableIdField] : []),
          ...(typeof items.labelField === "string" ? [items.labelField] : []),
        ]),
      ],
      requiredFieldTypes:
        items.requiredFieldTypes && typeof items.requiredFieldTypes === "object"
          ? items.requiredFieldTypes
          : {},
      optionalFieldEnums:
        items.optionalFieldEnums && typeof items.optionalFieldEnums === "object"
          ? items.optionalFieldEnums
          : {},
      stableIdField: typeof items.stableIdField === "string" ? items.stableIdField : null,
      collectionLimits: [
        ...(Array.isArray(contract.collections) ? contract.collections : []),
        ...(Number.isInteger(contract.maximumCount) && typeof contract.property === "string"
          ? [contract]
          : []),
      ].map((collection) => ({
        property: typeof collection.property === "string" ? collection.property : null,
        maximumCount: Number.isInteger(collection.maximumCount) ? collection.maximumCount : null,
        maximumLabelCharacters: Number.isInteger(collection.maximumLabelCharacters)
          ? collection.maximumLabelCharacters
          : null,
        requiredFields: [
          ...new Set([
            ...(Array.isArray(collection.requiredFields) ? collection.requiredFields : []),
            ...(typeof collection.stableIdField === "string" ? [collection.stableIdField] : []),
            ...(typeof collection.labelField === "string" ? [collection.labelField] : []),
          ]),
        ],
      })),
    };
    contracts.push(entry);
    if (version === activeVersion) activeContracts.push(entry);
  }
}

const duplicate = contracts.find(
  (contract, index) => contracts.findIndex((item) => item.key === contract.key) !== index,
);
if (duplicate) throw new Error(`duplicate usage-contract identity ${duplicate.key}`);
const contractRecord = (contract, key = contract.key) =>
  `  ${JSON.stringify(key)}: { maximumChildren: ${contract.maximumChildren ?? "null"}, maximumItems: ${contract.maximumItems ?? "null"}, maximumLabelCharacters: ${contract.maximumLabelCharacters ?? "null"}, primaryCount: ${contract.primaryCount ?? "null"}, itemsProperty: ${JSON.stringify(contract.itemsProperty)}, requiredFields: ${JSON.stringify(contract.requiredFields)}, requiredFieldTypes: ${JSON.stringify(contract.requiredFieldTypes)}, optionalFieldEnums: ${JSON.stringify(contract.optionalFieldEnums)}, stableIdField: ${JSON.stringify(contract.stableIdField)}, collectionLimits: ${JSON.stringify(contract.collectionLimits)} }`;
const body = contracts.map((contract) => contractRecord(contract)).join(",\n");
const activeBody = activeContracts
  .filter((contract) => Object.keys(contract.optionalFieldEnums).length > 0)
  .map((contract) => contractRecord(contract, contract.key.replace(/@[^@]+$/, "")))
  .join(",\n");
const generated = `// Generated from versioned usage-contract.json files. Run ui/scripts/generate-usage-contract-types.mjs to update.\nexport const usageContractLimits = {\n${body}\n} as const;\n\n/** Active contract for each asset with optional enum fields. */\nexport const activeUsageContractLimits = {\n${activeBody}\n} as const;\n\nexport type UsageContractId = keyof typeof usageContractLimits;\ntype ContractFor<Id extends UsageContractId> = (typeof usageContractLimits)[Id];\ntype RequiredContractFields<Id extends UsageContractId> = keyof ContractFor<Id>["requiredFieldTypes"] | ContractFor<Id>["requiredFields"][number];\ntype ContractFieldTypes<Id extends UsageContractId> = ContractFor<Id>["requiredFieldTypes"];\ntype RequireContractFields<Id extends UsageContractId, Item> = Item extends object\n  ? Item & { [Field in RequiredContractFields<Id>]: Field extends keyof ContractFieldTypes<Id> ? ContractFieldTypes<Id>[Field] extends "string" ? string : ContractFieldTypes<Id>[Field] extends "function" ? (...args: never[]) => unknown : unknown : unknown }\n  : never;\ntype TupleOfAtMost<Items extends readonly unknown[], Maximum extends number, Count extends unknown[] = []> = Items extends readonly [unknown, ...infer Rest]\n  ? Count["length"] extends Maximum ? never : readonly [Items[0], ...TupleOfAtMost<Rest, Maximum, [...Count, unknown]>]\n  : Items;\n/** Validates literal tuples while keeping dynamic arrays honest: runtime length is unknown. */\nexport type ContractItems<Id extends UsageContractId, Items extends readonly unknown[]> = ContractFor<Id>["maximumItems"] extends number\n  ? number extends Items["length"]\n    ? readonly RequireContractFields<Id, Items[number]>[]\n    : TupleOfAtMost<{ [Index in keyof Items]: RequireContractFields<Id, Items[Index]> }, ContractFor<Id>["maximumItems"]>\n  : readonly RequireContractFields<Id, Items[number]>[];
type ContractCollectionFor<Id extends UsageContractId, Property extends string> = Extract<ContractFor<Id>["collectionLimits"][number], { property: Property }>;
type RequireCollectionFields<Id extends UsageContractId, Property extends string, Item> = Item extends object
  ? Item & { [Field in ContractCollectionFor<Id, Property>["requiredFields"][number]]: Field extends "label" ? string : unknown }
  : never;
export type ContractCollectionItems<Id extends UsageContractId, Property extends string, Items extends readonly unknown[]> = ContractCollectionFor<Id, Property>["maximumCount"] extends number
  ? number extends Items["length"]
    ? readonly RequireCollectionFields<Id, Property, Items[number]>[]
    : TupleOfAtMost<{ [Index in keyof Items]: RequireCollectionFields<Id, Property, Items[Index]> }, ContractCollectionFor<Id, Property>["maximumCount"]>
  : readonly RequireCollectionFields<Id, Property, Items[number]>[];
\n`;
const formatted = await prettier.format(generated, { filepath: outputPath });
if (checkOnly) {
  if (!fs.existsSync(outputPath) || fs.readFileSync(outputPath, "utf8") !== formatted) {
    console.error(
      "usage-contract type map is stale; run node scripts/generate-usage-contract-types.mjs",
    );
    process.exitCode = 1;
  } else {
    console.log(`usage-contract type map matches ${contracts.length} versioned contracts`);
  }
} else {
  fs.writeFileSync(outputPath, formatted);
  console.log(`generated type map for ${contracts.length} versioned contracts`);
}
