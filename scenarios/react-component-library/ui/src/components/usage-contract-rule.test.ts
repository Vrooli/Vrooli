import { Linter } from "eslint";
import parser from "@typescript-eslint/parser";
import { describe, expect, it } from "vitest";
import usageContractRule from "../../eslint-rules/usage-contracts.js";

function check(source: string) {
  const linter = new Linter({ configType: "flat" });
  return linter
    .verify(
      source,
      [
        {
          files: ["**/*.tsx"],
          languageOptions: {
            parser,
            parserOptions: { ecmaVersion: 2022, sourceType: "module", ecmaFeatures: { jsx: true } },
          },
          plugins: { "rcl-usage-contracts": { rules: { "usage-contract": usageContractRule } } },
          rules: { "rcl-usage-contracts/usage-contract": "error" },
        },
      ],
      { filename: "fixture.tsx" },
    )
    .map((finding) => finding.messageId);
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

  it("flags shared DialogFooter composition that renders buttons without the action bar", () => {
    expect(
      check(`
      const view = <Dialog><DialogContent><DialogFooter><Button>Save</Button></DialogFooter></DialogContent></Dialog>;
    `),
    ).toContain("rebuild");
  });

  it("flags a custom component with an unrelated name when a footer uses it", () => {
    const findings = check(`
      function ProjectActions() { return <div><button>Save</button></div>; }
      const view = <Dialog footer={<ProjectActions />} />;
    `);
    expect(findings).toContain("customRebuild");
  });

  it("accepts a compliant action list", () => {
    expect(
      check(`
      const actions = [
        { id: "save", label: "Save", primary: true, onSelect: save },
        { id: "cancel", label: "Cancel", primary: false, onSelect: cancel },
      ];
      const view = <Dialog actions={actions} />;
    `),
    ).toEqual([]);
  });

  it("does not treat a dynamic template's interpolation fragments as an empty label", () => {
    const source = [
      "const count = 3;",
      "const actions = [{ id: 'apply', label: busy ? 'Applying' : `Apply ${count} recommendations`, primary: true, onSelect: apply }];",
      "const view = <OverlayActionBar actions={actions} />;",
    ].join("\n");
    expect(check(source)).not.toContain("emptyLabel");
  });

  it("defers empty-label judgment when a template consists only of runtime expressions", () => {
    const source = [
      "const label = 'verified';",
      "const rungs = [{ id: label, label: `${label.charAt(0).toUpperCase()}${label.slice(1)}`, complete: true }];",
      "const view = <ProgressLadder rungs={rungs} />;",
    ].join("\n");
    expect(check(source)).not.toContain("emptyLabel");
  });

  it("applies the latest catalog declaration to ordinary collection props", () => {
    const findings = check(`
      const destinations = [
        { id: "home", label: "Home" },
        { label: "Missing a stable id" },
        { id: "overflow", label: "A destination label that is deliberately much longer than the sixteen character budget" },
        { id: "settings", label: "Settings" },
        { id: "help", label: "Help" },
        { id: "extra", label: "Extra" },
      ];
      const view = <BottomNav items={destinations} />;
    `);
    expect(findings).toContain("tooMany");
    expect(findings).toContain("missingField");
    expect(findings).toContain("longLabel");
  });

  it("uses item field names declared by each component contract", () => {
    expect(
      check(`
      const records = [{ id: "r1", title: "An item title" }];
      const view = <CollectionList items={records} />;
    `),
    ).toEqual([]);
    expect(
      check(`
      const records = [{ id: "r1", label: "Wrong field" }];
      const view = <CollectionList items={records} />;
    `),
    ).toContain("missingField");
  });

  it("checks DataTable's component-owned row identity callback without inventing row labels", () => {
    expect(
      check(`
      const rows = [{ statementId: "s1", amount: 42 }];
      const columns = [{ id: "amount", header: "Amount" }];
      const view = <DataTable rows={rows} columns={columns} getRowKey={(row) => row.statementId} />;
    `),
    ).toEqual([]);
    expect(
      check(`
      const rows = [{ statementId: "s1", amount: 42 }];
      const columns = [{ id: "amount", header: "Amount" }];
      const view = <DataTable rows={rows} columns={columns} />;
    `),
    ).toContain("missingField");
  });

  it("inspects known entries inside conditional spreads without treating the spread as a malformed item", () => {
    expect(
      check(`
      const items = [
        ...(enabled ? [{ id: "new", label: t("new"), onSelect: create }] : []),
        { id: "settings", label: t("settings"), onSelect: openSettings },
      ];
      const view = <ContextMenu items={items} />;
    `),
    ).toEqual([]);
  });

  it("accepts data-key expressions as stable collection identities", () => {
    expect(
      check(`
      const source = [{ id: "record", title: "Title" }];
      const records = source.map((item) => ({ id: item.id, title: item.title }));
      const view = <CollectionList items={records} />;
    `),
    ).toEqual([]);
  });

  it("accepts a blank Select sentinel as a stable value identity", () => {
    expect(
      check(`
      const options = [{ value: "", label: "Choose a topic" }, { value: "health", label: "Health" }];
      const view = <Select options={options} />;
    `),
    ).toEqual([]);
  });

  it("accepts deterministic enum-to-string values as Select identities", () => {
    expect(
      check(`
      enum Range { DAY = "day", WEEK = "week" }
      const options = [{ value: String(Range.DAY), label: "Day" }, { value: String(Range.WEEK), label: "Week" }];
      const view = <Select options={options} />;
    `),
    ).toEqual([]);
  });

  it("unwraps TypeScript casts around literal collection identities", () => {
    expect(
      check(`
      type Door = "network" | "code";
      const items = [{ id: "network" as Door, label: "Network" }, { id: "code" as Door, label: "Code" }];
      const view = <Tabs items={items} />;
    `),
    ).toEqual([]);
  });

  it("does not count translation keys as rendered label text or identity strings", () => {
    expect(
      check(`
      const facts = [{ value: "4", label: i18n.t("onboarding.plan.resources") }];
      const view = <PlanSummary facts={facts} />;
      const options = [{ value: "attention", label: i18n.t("onboarding.readiness.credentialFilterAttention") }];
      const select = <Select options={options} />;
    `),
    ).toEqual([]);
  });

  it("checks every conditional object alternative without reporting valid fields as missing", () => {
    expect(
      check(`
      const items = [kind === "file"
        ? { id: "open-file", label: "Open file", onSelect: openFile }
        : { id: "open-url", label: "Open link", onSelect: openUrl }];
      const view = <ContextMenu items={items} />;
    `),
    ).toEqual([]);
    expect(
      check(`
      const items = [kind === "file"
        ? { id: "open-file", onSelect: openFile }
        : { id: "open-url", label: "Open link", onSelect: openUrl }];
      const view = <ContextMenu items={items} />;
    `),
    ).toContain("emptyLabel");
  });

  it("checks child component allowlists and maximum child counts", () => {
    const disallowed = check(`
      const view = <Form><Card>Unexpected child</Card></Form>;
    `);
    expect(disallowed).toContain("disallowedChild");

    const excessiveChildren = Array.from(
      { length: 11 },
      (_, index) => `<FormSection key="${index}" />`,
    ).join("");
    expect(check(`const view = <Form>${excessiveChildren}</Form>;`)).toContain("tooManyChildren");
  });

  it("does not count portal-owned AlertDialogs as ResponsiveDialog body children", () => {
    expect(
      check(`
      const view = <ResponsiveDialog><SettingsList /><AlertDialog open /></ResponsiveDialog>;
    `),
    ).toEqual([]);
  });

  it("checks only phone-visible AppShell items against its mobile tab budget", () => {
    expect(
      check(`
      const items = [
        { id: "home", label: "Home" },
        { id: "projects", label: "Projects" },
        { id: "activity", label: "Activity" },
        { id: "reports", label: "Reports" },
        { id: "settings", label: "Settings" },
        { id: "help", label: "Help", mobile: false },
      ];
      const view = <AppShell items={items} />;
    `),
    ).toEqual([]);
    expect(
      check(`
      const items = [
        { id: "home", label: "Home" },
        { id: "projects", label: "Projects" },
        { id: "activity", label: "Activity" },
        { id: "reports", label: "Reports" },
        { id: "settings", label: "Settings" },
        { id: "help", label: "Help" },
      ];
      const view = <AppShell items={items} />;
    `),
    ).toContain("tooMany");
    expect(
      check(`
      const items = [
        { id: "home", label: "Home" },
        { id: "projects", label: "Projects" },
        { id: "activity", label: "Activity" },
        { id: "reports", label: "Reports" },
        { id: "settings", label: "Settings" },
        buildDynamicNavItem(),
      ];
      const view = <AppShell items={items} />;
    `),
    ).toContain("tooMany");
  });
});
