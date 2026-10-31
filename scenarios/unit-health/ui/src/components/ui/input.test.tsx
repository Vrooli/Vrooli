/**
 * Unit tests for the Input primitive.
 *
 * What these tests pin:
 *   - The base className chunk (`rounded-md`) is always emitted, so a
 *     refactor that drops the cn() merge surfaces immediately.
 *   - Custom className is merged via tailwind-merge (cn helper) — both
 *     base and custom classes survive.
 *   - The forwardRef contract holds: useRef(null) populates with the
 *     real HTMLInputElement.
 *   - Arbitrary props pass through (placeholder, type, disabled).
 *   - Typing produces the expected value (smoke check on the
 *     controlled-input contract).
 *
 * No dedicated mock seam needed; the primitive is a leaf wrapper. When
 * a scenario adds form-validation behaviour, tests grow alongside.
 */
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { useRef } from "react";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Input } from "./input";

const designTokensPath = [
  resolve(process.cwd(), "src/design-tokens.css"),
  resolve(process.cwd(), "scenarios/unit-health/ui/src/design-tokens.css"),
].find(existsSync);
if (!designTokensPath) throw new Error("Could not locate Unit Health design-tokens.css");
const designTokens = readFileSync(designTokensPath, "utf8");

const readThemeBlock = (pattern: RegExp, theme: string) => {
  const block = designTokens.match(pattern)?.[1];
  if (!block) throw new Error(`Could not find the ${theme} design-token block`);
  return block;
};

const readHexToken = (block: string, token: string) => {
  const value = block.match(new RegExp(`--${token}:\\s*(#[\\da-f]{6})`, "i"))?.[1];
  if (!value) throw new Error(`Could not find --${token} in the design-token block`);
  return value;
};

const relativeLuminance = (hex: string) => {
  const channels = hex.match(/[\da-f]{2}/gi)?.map((channel) => {
    const value = parseInt(channel, 16) / 255;
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  if (!channels || channels.length !== 3) throw new Error(`Invalid hex color: ${hex}`);
  const [red, green, blue] = channels;
  if (red === undefined || green === undefined || blue === undefined) {
    throw new Error(`Invalid hex color: ${hex}`);
  }
  return red * 0.2126 + green * 0.7152 + blue * 0.0722;
};

const contrastRatio = (foreground: string, background: string) => {
  const foregroundLuminance = relativeLuminance(foreground);
  const backgroundLuminance = relativeLuminance(background);
  const lighter = Math.max(foregroundLuminance, backgroundLuminance);
  const darker = Math.min(foregroundLuminance, backgroundLuminance);
  return (lighter + 0.05) / (darker + 0.05);
};

describe("Input", () => {
  it("emits the base className chunk so the cn() merge contract holds", () => {
    renderWithProviders(<Input data-testid="i" />);
    const el = screen.getByTestId("i");
    expect(el.className).toMatch(/rounded-md/);
    expect(el.className).toMatch(/border/);
  });

  it("uses semantic theme colors for readable input text and focus", () => {
    renderWithProviders(<Input data-testid="i" />);
    const el = screen.getByTestId("i");

    expect(el.className).toContain("border-app-border");
    expect(el.className).toContain("bg-app-surface");
    expect(el.className).toContain("text-app-foreground");
    expect(el.className).toContain("placeholder:text-app-muted-foreground");
    expect(el.className).toContain("focus-visible:ring-app-primary/50");
  });

  it("keeps foreground and placeholder tokens readable in light and dark themes", () => {
    const themes = [
      { name: "light", block: readThemeBlock(/:root\s*\{([^}]+)\}/, "light") },
      {
        name: "dark",
        block: readThemeBlock(/:root\[data-resolved-theme="dark"\],\s*\.dark\s*\{([^}]+)\}/, "dark"),
      },
    ];

    for (const theme of themes) {
      const surface = readHexToken(theme.block, "color-surface");
      const foreground = readHexToken(theme.block, "color-foreground");
      const mutedForeground = readHexToken(theme.block, "color-muted-foreground");

      expect(contrastRatio(foreground, surface), `${theme.name} input text`).toBeGreaterThanOrEqual(4.5);
      expect(contrastRatio(mutedForeground, surface), `${theme.name} placeholder`).toBeGreaterThanOrEqual(4.5);
    }
  });

  it("merges a custom className with the base classes via cn()", () => {
    renderWithProviders(<Input data-testid="i" className="custom-extra" />);
    const el = screen.getByTestId("i");
    expect(el.className).toMatch(/custom-extra/);
    expect(el.className).toMatch(/rounded-md/);
  });

  it("forwards ref to the underlying HTMLInputElement", () => {
    const Capture = () => {
      const ref = useRef<HTMLInputElement | null>(null);
      return (
        <>
          <Input ref={ref} data-testid="i" />
          <button
            type="button"
            data-testid="probe"
            onClick={() => {
              if (ref.current instanceof HTMLInputElement) {
                ref.current.setAttribute("data-ref-ok", "true");
              }
            }}
          >
            probe
          </button>
        </>
      );
    };
    renderWithProviders(<Capture />);
    screen.getByTestId("probe").click();
    expect(screen.getByTestId("i")).toHaveAttribute("data-ref-ok", "true");
  });

  it("passes through arbitrary input attributes (type, placeholder, disabled)", () => {
    renderWithProviders(
      <Input
        data-testid="i"
        type="email"
        placeholder="email-ph"
        disabled
      />,
    );
    const el = screen.getByTestId<HTMLInputElement>("i");
    expect(el.type).toBe("email");
    expect(el.placeholder).toBe("email-ph");
    expect(el.disabled).toBe(true);
  });

  it("registers user typing through the native input value", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Input data-testid="i" />);
    const el = screen.getByTestId<HTMLInputElement>("i");

    await user.type(el, "abc");

    expect(el.value).toBe("abc");
  });
});
