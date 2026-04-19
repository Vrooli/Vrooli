import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render } from "@testing-library/react";

const frameCallbacks: Array<(state: unknown, delta: number) => void> = [];

vi.mock("@react-three/fiber", () => ({
  useFrame: (cb: (state: unknown, delta: number) => void) => {
    frameCallbacks.push(cb);
  },
}));

describe("TrivialCube scene module", () => {
  beforeEach(() => {
    frameCallbacks.length = 0;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("exports a function (React component) as the default export", async () => {
    const mod = await import("./trivialCube");
    expect(typeof mod.default).toBe("function");
    expect(mod.default.length).toBeGreaterThanOrEqual(0);
  });

  it("renders a mesh with a mesh-standard material using the accent color", async () => {
    vi.spyOn(window, "getComputedStyle").mockReturnValue({
      getPropertyValue: (key: string) => (key === "--cc-accent" ? "#00ff00" : ""),
    } as unknown as CSSStyleDeclaration);

    const { default: TrivialCube } = await import("./trivialCube");
    const { container } = render(<TrivialCube />);

    expect(container.querySelector("mesh")).not.toBeNull();
    const material = container.querySelector("meshStandardMaterial");
    expect(material).not.toBeNull();
    expect(material?.getAttribute("color")).toBe("#00ff00");
  });

  it("falls back to a default accent color when the CSS variable is missing", async () => {
    vi.spyOn(window, "getComputedStyle").mockReturnValue({
      getPropertyValue: () => "",
    } as unknown as CSSStyleDeclaration);

    const { default: TrivialCube } = await import("./trivialCube");
    const { container } = render(<TrivialCube />);

    const material = container.querySelector("meshStandardMaterial");
    expect(material).not.toBeNull();
    const color = material?.getAttribute("color") ?? "";
    expect(color.length).toBeGreaterThan(0);
    expect(color).not.toBe("#00ff00");
  });

  it("registers a useFrame callback on mount", async () => {
    const { default: TrivialCube } = await import("./trivialCube");
    const before = frameCallbacks.length;
    render(<TrivialCube />);
    expect(frameCallbacks.length).toBeGreaterThan(before);
    expect(typeof frameCallbacks[frameCallbacks.length - 1]).toBe("function");
  });
});
