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
import { useRef } from "react";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Input } from "./input";
import { renderWithProviders as render } from "../../test-utils";

describe("Input", () => {
  it("emits the base className chunk so the cn() merge contract holds", () => {
    render(<Input data-testid="i" />);
    const el = screen.getByTestId("i");
    expect(el.className).toMatch(/rounded-control/);
    expect(el.className).toMatch(/border/);
  });

  it("merges a custom className with the base classes via cn()", () => {
    render(<Input data-testid="i" className="custom-extra" />);
    const el = screen.getByTestId("i");
    expect(el.className).toMatch(/custom-extra/);
    expect(el.className).toMatch(/rounded-control/);
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
    render(<Capture />);
    screen.getByTestId("probe").click();
    expect(screen.getByTestId("i")).toHaveAttribute("data-ref-ok", "true");
  });

  it("passes through arbitrary input attributes (type, placeholder, disabled)", () => {
    render(
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
    render(<Input data-testid="i" />);
    const el = screen.getByTestId<HTMLInputElement>("i");

    await user.type(el, "abc");

    expect(el.value).toBe("abc");
  });
});
