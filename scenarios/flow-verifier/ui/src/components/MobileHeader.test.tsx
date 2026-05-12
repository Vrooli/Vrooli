import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { MobileHeader } from "./MobileHeader";

describe("MobileHeader", () => {
  afterEach(() => cleanup());

  it("renders brand, drawer toggle, and settings link", () => {
    renderWithProviders(<MobileHeader onOpenDrawer={() => {}} />);
    expect(screen.getByTestId("mobile-header")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-header-brand")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-header-drawer")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-header-settings")).toHaveAttribute(
      "href",
      "/settings",
    );
  });

  it("invokes onOpenDrawer when the drawer button is clicked", async () => {
    const onOpen = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<MobileHeader onOpenDrawer={onOpen} />);
    await user.click(screen.getByTestId("mobile-header-drawer"));
    expect(onOpen).toHaveBeenCalledOnce();
  });
});
