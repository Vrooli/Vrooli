import assert from "node:assert/strict";
import test from "node:test";
import React from "react";
import { inspectUsageContract } from "./usage-contract-runtime.mjs";

const contract = {
  componentId: "react-component-library:ActionList",
  version: "1.2.3",
  children: { maximumCount: 3 },
  items: {
    property: "actions",
    maximumCount: 2,
    requiredFields: ["id", "label", "onSelect"],
    requiredFieldTypes: { id: "string", label: "string", onSelect: "function" },
    stableIdField: "id",
    labelField: "label",
    maximumLabelCharacters: 8,
    primaryField: "primary",
    primaryCount: 1,
  },
};

test("DEV contract inspection catches excess children and malformed array shapes", () => {
  const Child = () => null;
  Child.rclUsageComponentName = "Card";
  const childViolations = inspectUsageContract(contract, {
    children: React.createElement(
      React.Fragment,
      null,
      "text",
      React.createElement(Child),
      React.createElement(Child),
      React.createElement(Child),
      React.createElement(Child),
    ),
  });
  assert.deepEqual(
    childViolations.map(({ field, limit, actual }) => ({
      field,
      limit,
      actual,
    })),
    [{ field: "children", limit: "maximumCount <= 3", actual: 4 }],
  );

  const wrongShape = inspectUsageContract(contract, {
    actions: { id: "save" },
  });
  assert.equal(wrongShape[0].field, "actions");
  assert.equal(wrongShape[0].limit, "array of item objects");
  assert.equal(wrongShape[0].actual, "object");
});

test("DEV contract inspection diagnoses array members with the wrong object shape and types", () => {
  const violations = inspectUsageContract(contract, {
    actions: [{ id: 42, label: ["Save"], onSelect: "not a function" }, null],
  });
  assert.deepEqual(
    violations.map(({ field, limit }) => `${field}: ${limit}`),
    [
      "actions[0].id: type string",
      "actions[0].label: type string",
      "actions[0].onSelect: type function",
      "actions[1]: object item",
      "actions.primary: primaryCount = 1",
    ],
  );
  const missingTypedField = inspectUsageContract(
    {
      componentId: "react-component-library:TypedList",
      version: "1.0.0",
      items: {
        property: "entries",
        requiredFieldTypes: { key: "string" },
      },
    },
    { entries: [{}] },
  );
  assert.deepEqual(missingTypedField[0], {
    field: "entries[0].key",
    limit: "type string",
    actual: "missing",
  });
});

test("valid collections pass and localized label caps count grapheme clusters", () => {
  assert.deepEqual(
    inspectUsageContract(contract, {
      actions: [{ id: "save", label: "Speich.", onSelect() {}, primary: true }],
    }),
    [],
  );
  const longLabel = inspectUsageContract(contract, {
    actions: [
      {
        id: "save",
        label: "Sichern 🧭🧭🧭🧭🧭🧭🧭🧭🧭",
        onSelect() {},
        primary: true,
      },
    ],
  });
  assert.ok(
    longLabel.some(
      (item) =>
        item.field === "actions[0].label" &&
        item.limit === "maximumLabelCharacters <= 8",
    ),
  );
});

test("structured action slots and named slots use the same versioned contract evaluator", () => {
  const slotContract = {
    ...contract,
    children: undefined,
    items: undefined,
    actions: {
      primaryAction: "required",
      primaryCount: 1,
      primarySlot: "saveLabel / onSave",
      secondarySlots: ["cancelLabel / onCancel", "helpLabel / onHelp"],
      maximumCount: 2,
      maximumLabelCharacters: 5,
    },
    slots: {
      footer: { maximumCount: 1, allowedChildComponents: ["OverlayActionBar"] },
    },
  };
  const FooterButton = () => null;
  FooterButton.rclUsageComponentName = "Button";
  const violations = inspectUsageContract(slotContract, {
    saveLabel: "Save now",
    onSave: "not a callback",
    onCancel() {},
    onHelp() {},
    footer: React.createElement(FooterButton),
  });
  assert.deepEqual(
    violations.map(({ field, limit }) => `${field}: ${limit}`),
    [
      "slots.footer.Button: allowedChildComponents = OverlayActionBar",
      "onSave: type function",
      "saveLabel: maximumLabelCharacters <= 5",
      "actions: maximumCount <= 2",
      "actions.primary: primaryCount = 1",
    ],
  );
});

test("legacy ReactNode action slots enforce the declared limit without requiring an optional primary", () => {
  const legacyActions = {
    componentId: "react-component-library:WorkspaceHeader",
    version: "1.0.5",
    actions: {
      maximumCount: 2,
      maximumLabelCharacters: 8,
      primaryAction: "optional",
      primaryCount: 1,
      primarySlot: "primaryAction",
      secondarySlot: "actions",
    },
  };
  assert.deepEqual(inspectUsageContract(legacyActions, {}), []);

  const Button = () => null;
  Button.rclUsageComponentName = "Button";
  const violations = inspectUsageContract(legacyActions, {
    primaryAction: React.createElement(Button, null, "Primary"),
    actions: React.createElement(
      React.Fragment,
      null,
      React.createElement(Button, null, "Secondary"),
      React.createElement(Button, null, "Overflow"),
    ),
  });
  assert.ok(
    violations.some(
      (item) => item.field === "actions" && item.limit === "maximumCount <= 2",
    ),
  );
  assert.ok(
    violations.some(
      (item) =>
        item.field === "primaryAction/actions.label" &&
        item.limit === "maximumLabelCharacters <= 8",
    ),
  );
});

test("legacy flat collection contracts remain enforceable without creating a second declaration", () => {
  const legacy = {
    componentId: "react-component-library:LegacyList",
    version: "1.0.0",
    property: "items",
    maximumCount: 1,
    requiredFields: ["id", "label"],
    requiredFieldTypes: { id: "string", label: "string" },
    identityCallbackProp: "getKey",
  };
  const violations = inspectUsageContract(legacy, {
    items: [{ id: "a" }, { id: "b", label: "B" }],
  });
  assert.ok(
    violations.some(
      (item) => item.field === "items" && item.limit === "maximumCount <= 1",
    ),
  );
  assert.ok(
    violations.some(
      (item) =>
        item.field === "items[0].label" && item.limit === "required field",
    ),
  );
  assert.ok(
    violations.some(
      (item) => item.field === "getKey" && item.limit === "type function",
    ),
  );
});

test("phone-specific item caps and child allowlists are enforced from their versioned fields", () => {
  const responsiveContract = {
    ...contract,
    children: {
      maximumCount: 4,
      allowedChildComponents: ["Card"],
      excludedChildComponents: ["OverlayActionBar"],
    },
    items: {
      property: "items",
      maximumCount: 8,
      mobileMaximumCount: 2,
      mobileField: "mobile",
      phoneLabelField: "shortLabel",
      maximumLabelCharacters: 5,
    },
  };
  const namedChild = (name) => {
    const Component = () => null;
    Component.rclUsageComponentName = name;
    return React.createElement(Component);
  };
  const violations = inspectUsageContract(responsiveContract, {
    children: [namedChild("Button"), namedChild("OverlayActionBar")],
    items: [
      { mobile: true, shortLabel: "First" },
      { mobile: true, shortLabel: "Second" },
      { mobile: true, shortLabel: "Third" },
      { mobile: false, shortLabel: "Desktop only" },
    ],
  });
  assert.ok(
    violations.some(
      (item) =>
        item.field === "children.Button" &&
        item.limit.includes("allowedChildComponents"),
    ),
  );
  assert.ok(
    !violations.some((item) => item.field.includes("OverlayActionBar")),
  );
  assert.ok(
    violations.some(
      (item) =>
        item.field === "items.mobile" &&
        item.limit === "mobileMaximumCount <= 2",
    ),
  );
  assert.ok(
    violations.some(
      (item) =>
        item.field === "items[3].shortLabel" &&
        item.limit === "maximumLabelCharacters <= 5",
    ),
  );
});
