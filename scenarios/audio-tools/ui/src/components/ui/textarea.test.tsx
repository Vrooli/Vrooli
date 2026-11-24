/**
 * Consumer contract checks for the shared RCL Textarea used in Audio Tools.
 */
import { useRef } from "react";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { Textarea } from "@vrooli/react-component-library/Textarea/1.1.1";

describe("Textarea", () => {
  it("exposes the RCL component marker", () => {
    render(<Textarea data-testid="t" />);
    const el = screen.getByTestId("t");
    expect(el).toHaveAttribute("data-rcl-textarea", "true");
  });

  it("preserves caller classes", () => {
    render(<Textarea data-testid="t" className="custom-extra" />);
    const el = screen.getByTestId("t");
    expect(el.className).toMatch(/custom-extra/);
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
