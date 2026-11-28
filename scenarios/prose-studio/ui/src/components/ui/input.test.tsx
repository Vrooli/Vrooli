import { useRef } from "react";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";

describe("Input", () => {
  it("forwards caller classes to the input", () => {
    renderWithProviders(<Input data-testid="i" className="custom-extra px-6" />);
    expect(screen.getByTestId("i")).toHaveClass("custom-extra", "px-6");
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
