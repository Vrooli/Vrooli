import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { expectNoA11yViolations } from "../../test-utils";
import { StudioChrome } from "./StudioChrome";

const labels = {
  radius: "Corner radius",
  serif: "Serif",
  sans: "Sans",
} as const;

describe("StudioChrome", () => {
  it("emits a clamped radius and a display face", async () => {
    await setLocale("en");
    const onChange = vi.fn();
    renderWithProviders(<StudioChrome options={{ radius: 14, display: "sans" }} onChange={onChange} />);

    const radius = screen.getByLabelText(labels.radius);
    fireEvent.change(radius, { target: { value: "99" } });
    expect(onChange).toHaveBeenCalledWith({ radius: 48 }, expect.any(String));

    fireEvent.click(screen.getByRole("button", { name: labels.serif }));
    expect(onChange).toHaveBeenCalledWith({ display: "serif" }, expect.any(String));
  });

  it("falls back to the library defaults when options are absent", async () => {
    await setLocale("en");
    renderWithProviders(<StudioChrome options={{}} onChange={vi.fn()} />);
    expect(screen.getByLabelText(labels.radius)).toHaveValue(16);
    expect(screen.getByRole("button", { name: labels.serif })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });

  it("has no axe violations", async () => {
    await setLocale("en");
    const { container } = renderWithProviders(
      <StudioChrome options={{ radius: 14, display: "sans" }} onChange={vi.fn()} />,
    );
    await expectNoA11yViolations(container);
  });
});
