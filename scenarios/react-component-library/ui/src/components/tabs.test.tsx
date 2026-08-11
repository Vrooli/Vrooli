import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Tabs } from "./Tabs";

describe("Tabs", () => {
  it("supports labeled items, badges, and roving keyboard selection", async () => {
    const user = userEvent.setup();
    render(
      <Tabs
        items={[
          { id: "overview", label: "Overview", badge: 3 },
          { id: "files", label: "Files" },
        ]}
        defaultActive="overview"
        ariaLabel="Asset information"
      />,
    );

    const overview = screen.getByRole("tab", { name: "Overview" });
    const files = screen.getByRole("tab", { name: "Files" });
    expect(overview).toHaveAttribute("aria-selected", "true");
    expect(overview).toHaveAttribute("tabindex", "0");
    expect(files).toHaveAttribute("tabindex", "-1");

    await user.click(files);
    expect(files).toHaveAttribute("aria-selected", "true");
    expect(overview).toHaveAttribute("aria-selected", "false");

    await user.keyboard("{ArrowLeft}");
    expect(overview).toHaveFocus();
    expect(overview).toHaveAttribute("aria-selected", "true");
  });
});
