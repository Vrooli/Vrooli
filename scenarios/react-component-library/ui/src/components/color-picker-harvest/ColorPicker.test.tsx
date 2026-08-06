import { cleanup, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import ColorPicker from "./ColorPicker";
import { isLightColor, parseColorValue, serializeColorValue } from "./colorUtils";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

describe("ColorPicker", () => {
  it("supports palette, transparent, gradient, recent, and custom-color flows", () => {
    const onChange = vi.fn();
    const onRecordRecent = vi.fn();

    renderWithProviders(
      <ColorPicker
        palette={["#000000", "#ffffff"]}
        value="#000000"
        recentColors={["#ffffff"]}
        onChange={onChange}
        onRecordRecent={onRecordRecent}
        allowGradient
        labels={{
          heading: "Theme colors",
          primary: "Primary",
          secondary: "Secondary",
          custom: "Custom",
          transparent: "None",
        }}
      />,
    );

    expect(screen.getByRole("region", { name: "Theme colors" })).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("color-picker-palette-#ffffff"));
    expect(onChange).toHaveBeenCalledWith("#ffffff");
    expect(onRecordRecent).toHaveBeenCalledWith("#ffffff");

    fireEvent.click(screen.getByTestId("color-picker-add-gradient"));
    expect(screen.getByTestId("color-picker-slot-1")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("color-picker-slot-1"));
    fireEvent.click(screen.getByRole("button", { name: "#000000" }));
    expect(onChange).toHaveBeenCalledWith("#000000|#000000");

    fireEvent.click(screen.getByTestId("color-picker-remove-gradient"));
    expect(onChange).toHaveBeenCalledWith("#000000");
    fireEvent.click(screen.getByRole("button", { name: "None" }));
    expect(onChange).toHaveBeenCalledWith("transparent");

    const custom = screen.getByTestId("color-picker-custom-input");
    fireEvent.change(custom, { target: { value: "#abcdef" } });
    fireEvent.blur(custom);
    expect(onChange).toHaveBeenCalledWith("#abcdef");
    expect(onRecordRecent).toHaveBeenCalledWith("#abcdef");
    expect(screen.getByTestId("color-picker-recents")).toBeInTheDocument();
  });

  it("normalizes color utility inputs and supports short and light hex values", () => {
    expect(parseColorValue(" #abc | #000000 | invalid ")).toEqual({
      colors: ["#abc", "#000000"],
      transparent: false,
    });
    expect(parseColorValue(undefined)).toEqual({ colors: [], transparent: true });
    expect(serializeColorValue(["#abc", "invalid", "#ffffff"])).toBe("#abc|#ffffff");
    expect(serializeColorValue([])).toBe("transparent");
    expect(isLightColor("#fff")).toBe(true);
    expect(isLightColor("#000000")).toBe(false);
    expect(isLightColor("not-a-color")).toBe(false);
  });

  it("keeps the compact non-gradient and pre-existing gradient variants accessible", () => {
    const onChange = vi.fn();
    cleanup();
    renderWithProviders(<ColorPicker palette={[]} value="transparent" onChange={onChange} />);
    expect(screen.getByRole("region", { name: "Color picker" })).toBeInTheDocument();
    expect(screen.queryByTestId("color-picker-add-gradient")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("color-picker-transparent"));
    expect(onChange).toHaveBeenCalledWith("transparent");

    cleanup();
    renderWithProviders(
      <ColorPicker palette={["#ffffff"]} value="transparent" onChange={onChange} allowGradient />,
    );
    fireEvent.click(screen.getByTestId("color-picker-add-gradient"));
    fireEvent.click(screen.getByTestId("color-picker-palette-#ffffff"));

    cleanup();
    renderWithProviders(
      <ColorPicker palette={["#ffffff"]} value="#abc|#000000" onChange={onChange} allowGradient />,
    );
    expect(screen.getByTestId("color-picker-slot-1")).toHaveAttribute("aria-label", "Color picker");
    expect(screen.getByTestId("color-picker-custom")).toHaveAttribute("title", "Custom color");
    fireEvent.click(screen.getByTestId("color-picker-slot-1"));
    fireEvent.click(screen.getByTestId("color-picker-palette-#ffffff"));
    fireEvent.change(screen.getByTestId("color-picker-custom-input"), {
      target: { value: "#123456" },
    });
    fireEvent.blur(screen.getByTestId("color-picker-custom-input"));
    fireEvent.click(screen.getByTestId("color-picker-remove-gradient"));
    expect(onChange).toHaveBeenCalledWith("#abc");
  });
});
