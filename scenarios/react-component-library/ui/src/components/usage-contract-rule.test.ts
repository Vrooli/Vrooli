import { Linter } from "eslint";
import parser from "@typescript-eslint/parser";
import { describe, expect, it } from "vitest";
import usageContractRule from "../../eslint-rules/usage-contracts.js";

function check(source: string) {
  const linter = new Linter({ configType: "flat" });
  return linter.verify(source, [{
    files: ["**/*.tsx"],
    languageOptions: {
      parser,
      parserOptions: { ecmaVersion: 2022, sourceType: "module", ecmaFeatures: { jsx: true } },
    },
    plugins: { "rcl-usage-contracts": { rules: { "usage-contract": usageContractRule } } },
    rules: { "rcl-usage-contracts/usage-contract": "error" },
  }], { filename: "fixture.tsx" }).map((finding) => finding.messageId);
}

describe("RCL usage-contract ESLint rule", () => {
  it("flags excessive actions, unstable ids, long labels, and missing primary action", () => {
    const findings = check(`
      const actions = [
        { id: makeId(), label: "Ein sehr langer deutscher Aktionstext, der viel zu breit ist", onSelect: "not-a-function" },
        { id: "two", label: "Zwei", onSelect: run },
        { id: "three", label: "Drei", onSelect: run },
        { id: "four", label: "Vier", onSelect: run },
        { id: "five", label: "Fünf", onSelect: run },
      ];
      const view = <FullPageDrawer actions={actions} />;
    `);
    expect(findings).toContain("tooMany");
    expect(findings).toContain("unstableID");
    expect(findings).toContain("invalidFieldType");
    expect(findings).toContain("longLabel");
    expect(findings).toContain("primaryCount");
  });

  it("flags custom overlay footers that rebuild shared action controls", () => {
    const findings = check(`
      function DrawerActions() { return <div><button>Save</button></div>; }
      const view = <FullPageDrawer footer={<footer><button>Cancel</button></footer>} />;
    `);
    expect(findings).toContain("customRebuild");
    expect(findings).toContain("rebuild");
  });

  it("accepts a compliant action list", () => {
    expect(check(`
      const actions = [
        { id: "save", label: "Save", primary: true, onSelect: save },
        { id: "cancel", label: "Cancel", primary: false, onSelect: cancel },
      ];
      const view = <Dialog actions={actions} />;
    `)).toEqual([]);
  });
});
