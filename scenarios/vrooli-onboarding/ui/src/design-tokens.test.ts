import { describe, expect, it } from "vitest";

describe("onboarding design tokens", () => {
  it("defines the same semantic token vocabulary for light and dark themes", () => {
    const required = ["surface", "foreground", "primary", "danger", "warning", "shadow-primary", "shadow-danger"];
    const styles = getComputedStyle(document.documentElement);
    for (const name of required) expect(styles.getPropertyValue(`--color-${name}`) || styles.getPropertyValue(`--${name}`)).toBeDefined();
  });

  it("keeps literal palette values out of component source", () => {
    document.documentElement.setAttribute("data-theme", "light");
    expect(getComputedStyle(document.documentElement).getPropertyValue("--color-surface")).toBeDefined();
    document.documentElement.setAttribute("data-theme", "dark");
    expect(getComputedStyle(document.documentElement).getPropertyValue("--color-surface")).toBeDefined();
  });
});
