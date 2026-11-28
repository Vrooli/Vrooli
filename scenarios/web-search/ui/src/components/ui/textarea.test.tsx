import { renderWithProviders as render } from "../../test-utils";
/**
 * Consumer checks for the shared versioned RCL Textarea contract.
 */
import { useRef } from "react";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Textarea } from "@vrooli/react-component-library/Textarea/1.1.1";

describe("Textarea", () => {
  it("renders the shared RCL Textarea marker", () => {
    render(<Textarea data-testid="t" />);
    const el = screen.getByTestId("t");
    expect(el).toHaveAttribute("data-rcl-textarea", "true");
  });

  it("merges a custom className with the base classes via cn()", () => {
    render(<Textarea data-testid="t" className="custom-extra" />);
    const el = screen.getByTestId("t");
    expect(el.className).toMatch(/custom-extra/);
    expect(el).toHaveAttribute("data-rcl-textarea", "true");
  });

  it("forwards ref to the underlying HTMLTextAreaElement", () => {
    const Capture = () => {
      const ref = useRef<HTMLTextAreaElement | null>(null);
      return (
        <>
          <Textarea ref={ref} data-testid="t" />
          <button
            type="button"
            data-testid="probe"
            onClick={() => {
              if (ref.current instanceof HTMLTextAreaElement) {
                ref.current.setAttribute("data-ref-ok", "true");
              }
            }}
          >
            probe
          </button>
        </>
      );
    };
    render(<Capture />);
    screen.getByTestId("probe").click();
    expect(screen.getByTestId("t")).toHaveAttribute("data-ref-ok", "true");
  });

  it("registers user typing through the native textarea value", async () => {
    const user = userEvent.setup();
    render(<Textarea data-testid="t" />);
    const el = screen.getByTestId<HTMLTextAreaElement>("t");

    await user.type(el, "hello\nworld");

    expect(el.value).toBe("hello\nworld");
  });
});
