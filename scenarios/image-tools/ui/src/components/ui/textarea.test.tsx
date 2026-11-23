/**
 * Image Tools adoption contract for the shared RCL Textarea asset.
 * Keep the test beside the deleted local-kit path so this app's required
 * attributes, custom classes, refs, and typing remain explicit.
 */
import { useRef } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import userEvent from "@testing-library/user-event";

import { Textarea } from "@vrooli/react-component-library/Textarea/1.1.1";

afterEach(() => cleanup());

describe("Image Tools RCL Textarea", () => {
  it("renders the shared textarea hook and preserves caller attributes", () => {
    render(<Textarea data-testid="t" aria-label="Review rationale" disabled />);
    const el = screen.getByTestId("t");
    expect(el).toHaveAttribute("data-rcl-textarea", "true");
    expect(el).toHaveAttribute("aria-label", "Review rationale");
    expect(el).toBeDisabled();
  });

  it("preserves scenario and custom classes", () => {
    render(<Textarea data-testid="t" className="image-tools-textarea custom-extra" />);
    const el = screen.getByTestId("t");
    expect(el.className).toMatch(/image-tools-textarea/);
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
