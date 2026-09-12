/**
 * Unit tests for the Textarea primitive.
 *
 * Mirrors `input.test.tsx`: the contract is identical (forwardRef, cn
 * merge, prop pass-through, value-on-type). Kept as a separate file so
 * a regression in either primitive has a single locus and the failure
 * message names the primitive that broke.
 */
import { useRef } from "react";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Textarea } from "./textarea";
import { renderWithProviders as render } from "../../test-utils";

describe("Textarea", () => {
  it("emits the base className chunk so the cn() merge contract holds", () => {
    render(<Textarea data-testid="t" />);
    const el = screen.getByTestId("t");
    expect(el.className).toMatch(/rounded-md/);
    expect(el.className).toMatch(/border/);
  });

  it("merges a custom className with the base classes via cn()", () => {
    render(<Textarea data-testid="t" className="custom-extra" />);
    const el = screen.getByTestId("t");
    expect(el.className).toMatch(/custom-extra/);
    expect(el.className).toMatch(/rounded-md/);
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
