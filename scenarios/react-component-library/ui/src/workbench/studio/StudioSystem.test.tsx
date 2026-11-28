import { fireEvent, screen } from "@testing-library/react";
import { vi } from "vitest";

import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import type { DesignFrame } from "../design-types";
import { node, type TemplateNode, type TemplatePalette } from "../templates/spec";
import type { Coverage } from "./rcl";
import { StudioSystem } from "./StudioSystem";

vi.mock("./StudioChrome", () => ({
  StudioChrome: ({
    onChange,
  }: {
    onChange: (patch: { radius: number }, label: string) => void;
  }) => <button onClick={() => onChange({ radius: 16 }, "Change radius")}>Set frame radius</button>,
}));
vi.mock("./StudioExtract", () => ({
  StudioExtract: ({ onSelect }: { spec: TemplateNode; onSelect: (id: string) => void }) => (
    <button onClick={() => onSelect("extracted-layer")}>Select extracted component</button>
  ),
}));

const theme: TemplatePalette = {
  bg: "#ffffff",
  surface: "#f4f4f4",
  surfaceAlt: "#eeeeee",
  ink: "#111111",
  muted: "#666666",
  line: "#dddddd",
  accent: "#3366cc",
  accentInk: "#ffffff",
  accentSoft: "#ddeeff",
  secondary: "#445566",
};
const frame: DesignFrame = {
  id: "phone",
  name: "Phone",
  device: "phone",
  spec: node("page", "page", {}, [
    node("hero", "stack", {}, [
      node("primary-action", "button", { label: "Continue" }),
      node("secondary-action", "button", { label: "Back" }),
      node("content-card", "card", {}, [node("copy", "text", { text: "Welcome" })]),
    ]),
  ]),
};
const desktopFrame: DesignFrame = {
  ...frame,
  id: "desktop",
  name: "Desktop",
  device: "desktop",
  spec: node("desktop-page", "page", {}, [node("desktop-action", "button", {})]),
};
const coverage: Coverage = { rcl: 3, custom: 2, total: 5, ratio: 0.6 };

function renderSystem(overrides: Partial<Parameters<typeof StudioSystem>[0]> = {}) {
  const props = {
    theme,
    frames: [frame, desktopFrame],
    activeFrame: frame,
    token: null,
    onSelectToken: vi.fn(),
    onTokenChange: vi.fn(),
    onSelectNode: vi.fn(),
    onSelectFrame: vi.fn(),
    coverage,
    options: { radius: 8, display: "sans" as const },
    onOptionsChange: vi.fn(),
    ...overrides,
  };
  return { ...renderWithProviders(<StudioSystem {...props} />), props };
}

describe("StudioSystem", () => {
  it("selects an unselected color token", () => {
    const { props } = renderSystem();
    const background = screen.getByRole("button", { name: /workbench\.studio\.token\.bg/ });

    expect(background).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(background);
    expect(props.onSelectToken).toHaveBeenCalledWith("bg");
  });

  it("clears the current token when it is selected again", () => {
    const { props } = renderSystem({ token: "bg" });
    const background = screen.getByRole("button", { name: /workbench\.studio\.token\.bg/ });

    expect(background).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(background);
    expect(props.onSelectToken).toHaveBeenCalledWith(null);
  });

  it("edits valid color tokens through both the picker and text field", () => {
    const onTokenChange = vi.fn();
    renderSystem({ token: "accent", onTokenChange });

    fireEvent.change(screen.getByLabelText(strings.workbench.studio.tokenPicker), {
      target: { value: "#abcdef" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.tokenValue), {
      target: { value: "blue" },
    });
    expect(onTokenChange).toHaveBeenNthCalledWith(1, "accent", "#abcdef");
    expect(onTokenChange).toHaveBeenNthCalledWith(2, "accent", "blue");
  });

  it("shows primitives, grouping patterns and frame navigation for the active design", () => {
    const { props } = renderSystem();

    fireEvent.click(screen.getByRole("button", { name: /Button/ }));
    expect(props.onSelectNode).toHaveBeenCalledWith("primary-action");
    expect(screen.getByText(strings.workbench.studio.pattern.stack)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.studio.pattern.card)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Desktop/ }));
    expect(props.onSelectFrame).toHaveBeenCalledWith("desktop");
    expect(screen.getByText(strings.workbench.studio.chainCoverage)).toBeInTheDocument();
  });

  it("shows empty usage states and omits chrome controls when no updater is supplied", () => {
    const textOnly: DesignFrame = {
      ...frame,
      spec: node("plain-page", "page", {}, [node("copy-only", "text", { text: "Just copy" })]),
    };
    renderSystem({
      frames: [textOnly],
      activeFrame: textOnly,
      onOptionsChange: undefined,
      options: undefined,
    });

    expect(screen.getByText(strings.workbench.studio.noPrimitives)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.studio.noPatterns)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Set frame radius" })).not.toBeInTheDocument();
  });

  it("passes design chrome updates and extracted selections through their callbacks", () => {
    const { props } = renderSystem();

    fireEvent.click(screen.getByRole("button", { name: "Set frame radius" }));
    expect(props.onOptionsChange).toHaveBeenCalledWith({ radius: 16 }, "Change radius");
    fireEvent.click(screen.getByRole("button", { name: "Select extracted component" }));
    expect(props.onSelectNode).toHaveBeenCalledWith("extracted-layer");
  });
});
