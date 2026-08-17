import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { Button } from "./button";
import { renderWithProviders } from "../../test-utils";

describe("Button", () => {
  it("renders its default button element and forwards state", () => {
    renderWithProviders(<Button disabled>Save</Button>);

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("supports the Radix Slot composition path", () => {
    renderWithProviders(
      <Button asChild>
        <a href="/settings">Settings</a>
      </Button>,
    );

    expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/settings");
  });
});
