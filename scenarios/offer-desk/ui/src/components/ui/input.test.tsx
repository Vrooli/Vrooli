/**
 * Unit tests for the Input primitive.
 *
 * What these tests pin:
 *   - The adopted RCL component marks its control with `data-rcl-input` and
 *     supplies its raised surface recipe as the base class.
 *   - Custom className is merged alongside the component's base recipe.
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

import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { renderWithProviders } from "../../test-utils";

describe("RCL Input", () => {
  it("emits the adopted input marker and raised surface recipe", () => {
    renderWithProviders(<Input data-testid="i" />);
    const el = screen.getByTestId("i");
    expect(el).toHaveAttribute("data-rcl-input", "true");
    expect(el.className).toContain("shadow-[var(--elev-raised)]");
  });

  it("merges a custom className with the component's base recipe", () => {
    renderWithProviders(<Input data-testid="i" className="custom-extra px-6" />);
    const el = screen.getByTestId("i");
    expect(el.className).toMatch(/custom-extra/);
    expect(el.className).toContain("px-6");
    expect(el.className).toContain("shadow-[var(--elev-raised)]");
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
    renderWithProviders(<Capture />);
    screen.getByTestId("probe").click();
    expect(screen.getByTestId("i")).toHaveAttribute("data-ref-ok", "true");
  });

  it("passes through arbitrary input attributes (type, placeholder, disabled)", () => {
    renderWithProviders(
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
    renderWithProviders(<Input data-testid="i" />);
    const el = screen.getByTestId<HTMLInputElement>("i");

    await user.type(el, "abc");

    expect(el.value).toBe("abc");
  });
});
