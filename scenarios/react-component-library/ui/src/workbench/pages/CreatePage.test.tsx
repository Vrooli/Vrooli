import { fireEvent, screen } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { CreatePage } from "./CreatePage";

vi.mock("./CreateWorkflowLayout", () => ({
  CreateWorkflowLayout: ({ route, title }: { route: string; title: string }) => (
    <output data-testid="create-workflow" data-route={route} data-title={title} />
  ),
}));

function renderCreatePage() {
  return renderWithProviders(
    <Routes>
      <Route path="/new" element={<CreatePage />} />
      <Route path="/components" element={<div data-testid="components-route">Component catalog</div>} />
    </Routes>,
    { routerEntries: ["/new"] },
  );
}

describe("CreatePage", () => {
  it("switches among create, extract, and adopt workflows without losing the selected route", () => {
    renderCreatePage();

    expect(screen.getByTestId("create-workflow")).toHaveAttribute("data-route", "create");
    const tabs = document.querySelectorAll<HTMLButtonElement>(".wb-create-tab");
    expect(tabs[0]).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(tabs[1]!);
    expect(screen.getByTestId("create-workflow")).toHaveAttribute("data-route", "extract");
    expect(screen.getByTestId("create-workflow")).toHaveAttribute("data-title", strings.workbench.create.extractTitle);

    fireEvent.click(tabs[2]!);
    expect(screen.getByTestId("create-workflow")).toHaveAttribute("data-route", "adopt");
    expect(tabs[2]).toHaveAttribute("aria-pressed", "true");
  });

  it("cancels the authoring route back to the component catalog", () => {
    renderCreatePage();
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.create.cancel }));
    expect(screen.getByTestId("components-route")).toBeInTheDocument();
  });
});
