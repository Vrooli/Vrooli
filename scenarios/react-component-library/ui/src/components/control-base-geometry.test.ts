import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createElement } from "react";
import { ControlBase } from "@vrooli/react-component-library/ControlBase/1.1.0";
import { renderWithProviders } from "../test-utils";

const directory = path.dirname(fileURLToPath(import.meta.url));
const sizingContract = readFileSync(
  path.resolve(directory, "../../../docs/reference/sizing-contract.md"),
  "utf8",
);
const controlBaseSource = readFileSync(
  path.resolve(directory, "../../../library/components/ControlBase/versions/1.1.0/ControlBase.tsx"),
  "utf8",
);
const baseStylesSource = readFileSync(
  path.resolve(directory, "../../../library/foundations/BaseStyles/versions/1.0.0/BaseStyles.ts"),
  "utf8",
);

const documentedRungs = [...sizingContract.matchAll(/^\| `([^`]+)` \| (\d+)px \|/gm)].map(
  (match) => ({ name: match[1] ?? "", pixels: match[2] ?? "" }),
);

describe("ControlBase documented geometry", () => {
  it("keeps every documented rung backed by its matching size token", () => {
    expect(documentedRungs).toEqual([
      { name: "xs", pixels: "32" },
      { name: "sm", pixels: "36" },
      { name: "md", pixels: "40" },
      { name: "lg", pixels: "44" },
      { name: "xl", pixels: "48" },
      { name: "icon", pixels: "40" },
    ]);

    for (const rung of documentedRungs) {
      expect(controlBaseSource).toContain(
        `${rung.name}: { minBlockSize: "var(--control-size-${rung.name})"`,
      );
    }
    expect(controlBaseSource).toContain(
      'icon: { minBlockSize: "var(--control-size-icon)", minInlineSize: "var(--control-size-icon)"',
    );
    expect(controlBaseSource).not.toContain("min-height: var(--tap-target-min)");
    expect(controlBaseSource).not.toContain("min-width: var(--tap-target-min)");
  });

  it("keeps the direct-child icon scale monotonic and tokenized", () => {
    expect(baseStylesSource).toContain('--control-icon-size-xs: 12px');
    expect(baseStylesSource).toContain('--control-icon-size-sm: 14px');
    expect(baseStylesSource).toContain('--control-icon-size-md: 16px');
    expect(baseStylesSource).toContain('--control-icon-size-lg: 18px');
    expect(baseStylesSource).toContain('--control-icon-size-xl: 20px');
    expect(baseStylesSource).toContain('--control-icon-size-icon: 16px');
    expect(baseStylesSource).toContain('[data-control-size="xs"] { --control-icon-size: var(--control-icon-size-xs); }');
    expect(baseStylesSource).toContain('[data-control-size="icon"] { --control-icon-size: var(--control-icon-size-icon); }');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  // The tap-target rung is a property of the static size table, so it is
  // asserted from the source rather than from a render-time console warning.
  // The warning this replaced was gated on `import.meta.env.DEV`, which only
  // exists under Vite and vite-node: the assertion passed here while every
  // browser render threw. A contract the test runner can satisfy alone is not
  // a contract the consumers get.
  it("states each rung's pixels once and derives the tap-target marking from them", () => {
    expect(controlBaseSource).toContain(
      'const sizePixels: Record<ControlSize, number> = { xs: 32, sm: 36, md: 40, lg: 44, xl: 48, icon: 40, default: 40 };',
    );
    expect(controlBaseSource).toContain("const tapTargetMinimum = 44;");
    expect(controlBaseSource).toContain("const belowTapTarget = sizePixels[size] < tapTargetMinimum;");

    for (const rung of documentedRungs) {
      const documented = Number(rung.pixels);
      expect(controlBaseSource).toContain(`${rung.name}: ${documented},`);
      expect(documented < 44).toBe(["xs", "sm", "md", "icon"].includes(rung.name));
    }
  });

  it("never reads bundler-injected environment", () => {
    // Vitest materializes the bundler env object; browsers do not. Library
    // source that reads it is a render-time TypeError no suite here can
    // reproduce, so the assertion is on code, not on prose about it.
    const executable = controlBaseSource
      .replace(/\/\*[\s\S]*?\*\//g, "")
      .replace(/\/\/[^\n]*/g, "");
    expect(executable).not.toContain("import.meta");
  });

  it("marks dense rungs in the DOM without blocking rendering", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    renderWithProviders(
      createElement(ControlBase, { size: "xs", children: "Compact action" }),
    );

    expect(screen.getByRole("button", { name: "Compact action" })).toHaveAttribute(
      "data-control-below-tap-target",
      "true",
    );
    expect(warning).not.toHaveBeenCalled();
  });

  it("leaves comfortable rungs unmarked", () => {
    renderWithProviders(
      createElement(ControlBase, { size: "lg", children: "Roomy action" }),
    );

    expect(screen.getByRole("button", { name: "Roomy action" })).not.toHaveAttribute(
      "data-control-below-tap-target",
    );
  });

  it("keeps density as an internal gap choice", () => {
    const { rerender } = renderWithProviders(
      createElement(ControlBase, { density: "comfortable", children: "Action" }),
    );
    expect(screen.getByRole("button", { name: "Action" })).toHaveStyle({ gap: "var(--space-2xs)" });

    rerender(createElement(ControlBase, { density: "compact", children: "Action" }));
    expect(screen.getByRole("button", { name: "Action" })).toHaveStyle({ gap: "var(--space-3xs)" });
    expect(screen.getByRole("button", { name: "Action" })).toHaveAttribute("data-control-size", "md");
  });
});
