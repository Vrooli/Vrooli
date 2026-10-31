import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
// These exact package pins are intentional: this test compares previous and
// draft releases and must not resolve both imports to the current major line.
import { ControlBase as PreviousControl } from "@vrooli/react-component-library/ControlBase/1.1.2";
import { ControlBase as DraftControl } from "@vrooli/react-component-library/ControlBase/1.1.3";
import { BoundedMeter as PreviousMeter } from "@vrooli/react-component-library/BoundedMeter/1.0.9";
import { BoundedMeter as DraftMeter } from "@vrooli/react-component-library/BoundedMeter/1.0.10";
import { Button } from "@vrooli/react-component-library/Button/2";

const Control = process.env.RCL_RESTYLE_PRECHANGE ? PreviousControl : DraftControl;
const Meter = process.env.RCL_RESTYLE_PRECHANGE ? PreviousMeter : DraftMeter;

afterEach(cleanup);

describe("consumer restyling", () => {
  it("forwards Button classes through Pressable without inline recipe overrides", () => {
    render(<Button className="rounded-none bg-transparent px-0">Send</Button>);
    const button = screen.getByRole("button", { name: "Send" });
    expect(button.className).toBe("rounded-none bg-transparent px-0");
    expect(button.style.borderRadius).toBe("");
    expect(button.style.background).toBe("");
    expect(button.style.paddingInline).toBe("");
  });

  it("leaves recipe properties available to consumer classes", () => {
    render(<Control className="rounded-none bg-transparent px-0" shape="pill">Save</Control>);
    const button = screen.getByRole("button", { name: "Save" });
    expect(button.className).toBe("rounded-none bg-transparent px-0");
    for (const property of ["background", "border-color", "color", "padding-inline", "font-size", "line-height", "gap", "border-radius", "min-block-size"]) {
      expect(button.style.getPropertyValue(property), property).toBe("");
    }
  });

  it("preserves explicit consumer inline overrides and semantic state", () => {
    render(<Control style={{ borderRadius: 3 }} variant="danger" size="lg" disabled>Delete</Control>);
    const button = screen.getByRole("button", { name: "Delete" });
    expect(button.style.borderRadius).toBe("3px");
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("data-control-variant", "danger");
    expect(button).toHaveAttribute("data-control-size", "lg");
  });

  it("uses one elevation source and retains meter semantics", () => {
    render(<Meter label="Capacity" value={70} min={0} max={100} className="consumer-meter" />);
    const section = screen.getByRole("region", { name: "Capacity meter" });
    expect(section.className).toContain("consumer-meter");
    expect(section.style.boxShadow).toBe("");
    expect(screen.getByRole("meter")).toHaveAttribute("value", "70");
    expect(screen.getByRole("meter")).toHaveAttribute("max", "100");
  });
});
