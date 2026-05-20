import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { MobileDrawer } from "./MobileDrawer";

describe("MobileDrawer", () => {
  it("does not render when closed", () => {
    renderWithProviders(<MobileDrawer open={false} onClose={() => {}} />);
    expect(screen.queryByTestId(selectors.layout.drawer)).not.toBeInTheDocument();
  });

  it("renders the drawer + close button when open", () => {
    renderWithProviders(<MobileDrawer open={true} onClose={() => {}} />);
    expect(screen.getByTestId(selectors.layout.drawer)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.drawerClose)).toBeInTheDocument();
  });

  it("calls onClose when the close button is clicked", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<MobileDrawer open={true} onClose={onClose} />);
    await user.click(screen.getByTestId(selectors.layout.drawerClose));
    expect(onClose).toHaveBeenCalled();
  });
});
