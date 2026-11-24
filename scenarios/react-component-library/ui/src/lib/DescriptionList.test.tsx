import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DescriptionList } from "../../../library/components/DescriptionList/versions/1.1.0/DescriptionList";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("DescriptionList presentation modes", () => {
  it("preserves the default surface and string-entry API", () => {
    const term = "Owner";
    const value = "Platform";
    renderWithProviders(<DescriptionList entries={[{ term, description: value }]} />);
    const list = document.querySelector("[data-rcl-description-list]");
    expect(list).toHaveAttribute("data-appearance", "surface");
    expect(screen.getByText(term).tagName).toBe("DT");
    expect(screen.getByText(value).tagName).toBe("DD");
    expect(document.querySelector("[data-rcl-description-list-row]")).toBeInTheDocument();
  });

  it("accepts rich definition-list children with plain phone-first presentation", () => {
    const updatedLabel = "Updated";
    const today = "Today";
    const sourceLabel = "Source";
    const linkLabel = "Read the file";
    renderWithProviders(
      <DescriptionList appearance="plain">
        <div><dt>{updatedLabel}</dt><dd><time dateTime="2026-11-22">{today}</time></dd></div>
        <div><dt>{sourceLabel}</dt><dd><a href="/file">{linkLabel}</a></dd></div>
      </DescriptionList>,
    );
    const list = document.querySelector("[data-rcl-description-list]");
    expect(list).toHaveAttribute("data-appearance", "plain");
    expect(screen.getByText(updatedLabel).tagName).toBe("DT");
    expect(screen.getByText(today).closest("time")).toHaveAttribute("dateTime", "2026-11-22");
    expect(screen.getByRole("link", { name: linkLabel })).toHaveAttribute("href", "/file");
    expect(document.querySelector("[data-rcl-description-list-row]")).toBeNull();
  });
});
