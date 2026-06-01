import { describe, it, expect } from "vitest";
import { createSelectorRegistry, defineDynamicSelector } from "./selectors";

describe("selectors registry — literal selectors", () => {
  // Runtime convention: `selectors.x.y` returns the raw test ID (suitable for
  // `screen.getByTestId(...)`). The manifest exposes both the raw test ID and
  // the wrapped `[data-testid="..."]` form for CSS-style queries.

  it("exposes the raw test ID at the runtime path", () => {
    const { selectors } = createSelectorRegistry(
      {
        dashboard: { newProjectButton: "dashboard-new-project-button" },
      },
      {},
    );
    expect(selectors.dashboard.newProjectButton).toBe(
      "dashboard-new-project-button",
    );
  });

  it("emits manifest entries with both testId and wrapped selector", () => {
    const { manifest } = createSelectorRegistry(
      { foo: { bar: "foo-bar" } },
      {},
    );
    expect(manifest.selectors["foo.bar"]).toEqual({
      testId: "foo-bar",
      selector: '[data-testid="foo-bar"]',
    });
  });

  it("supports nested literal trees with dotted manifest keys", () => {
    const { manifest } = createSelectorRegistry(
      { a: { b: { c: "a-b-c" } } },
      {},
    );
    expect(manifest.selectors["a.b.c"]).toEqual({
      testId: "a-b-c",
      selector: '[data-testid="a-b-c"]',
    });
  });
});

describe("selectors registry — dynamic selectors", () => {
  const dynamicTree = {
    user: {
      cardByName: defineDynamicSelector({
        description: "User card filtered by name",
        selectorPattern:
          '[data-testid="user-card"][data-name="${name}"]',
        params: { name: { type: "string" as const } },
      }),
      itemAt: defineDynamicSelector({
        description: "Item at numeric index",
        testIdPattern: "item-${index}",
        params: { index: { type: "number" as const } },
      }),
      statusBadge: defineDynamicSelector({
        description: "Badge with enum status",
        testIdPattern: "status-${state}",
        params: {
          state: {
            type: "enum" as const,
            values: ["ok", "warn", "error"] as const,
          },
        },
      }),
    },
  };

  it("interpolates string params into the selector pattern", () => {
    const { selectors } = createSelectorRegistry({}, dynamicTree);
    expect(selectors.user.cardByName({ name: "Alice" })).toBe(
      '[data-testid="user-card"][data-name="Alice"]',
    );
  });

  it("interpolates numeric params via testIdPattern (returns raw ID)", () => {
    const { selectors } = createSelectorRegistry({}, dynamicTree);
    // testIdPattern emits the raw ID; callers wrap as needed via the manifest.
    expect(selectors.user.itemAt({ index: 3 })).toBe("item-3");
  });

  it("accepts enum values from the declared set", () => {
    const { selectors } = createSelectorRegistry({}, dynamicTree);
    expect(selectors.user.statusBadge({ state: "ok" })).toBe("status-ok");
  });

  it("throws when a required parameter is missing", () => {
    const { selectors } = createSelectorRegistry({}, dynamicTree);
    expect(() =>
      Reflect.apply(selectors.user.cardByName, undefined, [{}]),
    ).toThrow(/missing parameter 'name'/i);
  });

  it("throws when an unknown parameter is provided", () => {
    const { selectors } = createSelectorRegistry({}, dynamicTree);
    expect(() =>
      Reflect.apply(selectors.user.cardByName, undefined, [{
        name: "Bob",
        typo: "x",
      }]),
    ).toThrow(/unknown parameter/i);
  });

  it("throws when a numeric param receives a non-number", () => {
    const { selectors } = createSelectorRegistry({}, dynamicTree);
    expect(() =>
      Reflect.apply(selectors.user.itemAt, undefined, [{ index: "three" }]),
    ).toThrow(/must be numeric/i);
  });

  it("throws when an enum param is outside the declared set", () => {
    const { selectors } = createSelectorRegistry({}, dynamicTree);
    expect(() =>
      Reflect.apply(selectors.user.statusBadge, undefined, [{ state: "broken" }]),
    ).toThrow(/must be one of/i);
  });

  it("emits dynamic manifest entries with description and params", () => {
    const { manifest } = createSelectorRegistry({}, dynamicTree);
    expect(manifest.dynamicSelectors["user.cardByName"]).toMatchObject({
      description: "User card filtered by name",
      params: [{ name: "name", type: "string" }],
    });
  });
});
