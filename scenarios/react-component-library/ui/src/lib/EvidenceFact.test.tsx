import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { EvidenceFact } from "../../../library/components/EvidenceFact/versions/0.1.0/EvidenceFact";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("EvidenceFact disclosure", () => {
  it("keeps definition semantics and exposes evidence through a keyboard-operable control", async () => {
    const user = userEvent.setup();
    const label = "Last update";
    const value = "Five minutes ago";
    const disclosureLabel = "Why?";
    const expandedLabel = "Hide";
    const sourceReference = "owner file line 4";
    renderWithProviders(
      <dl>
        <EvidenceFact
          label={label}
          value={value}
          disclosure={{ label: disclosureLabel, expandedLabel, content: <ul><li>{sourceReference}</li></ul> }}
        />
      </dl>,
    );

    expect(screen.getByText(label).tagName).toBe("DT");
    expect(screen.getByText(value).tagName).toBe("SPAN");
    const toggle = screen.getByRole("button", { name: disclosureLabel });
    const target = document.getElementById(toggle.getAttribute("aria-controls") ?? "");
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(target).toHaveAttribute("hidden");

    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText(sourceReference)).toBeVisible();
    await user.click(screen.getByRole("button", { name: expandedLabel }));
    expect(screen.queryByText(sourceReference)).not.toBeVisible();
  });

  it("does not add an evidence control when no disclosure is supplied", () => {
    const value = "Unknown";
    renderWithProviders(<dl><EvidenceFact label="State" value={value} /></dl>);
    expect(screen.getByText(value)).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
