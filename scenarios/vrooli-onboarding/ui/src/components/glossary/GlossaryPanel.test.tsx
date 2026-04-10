// [REQ:REQ-P2-004] Contextual Help UI
import { screen, fireEvent } from "@testing-library/react";
import { renderWithQueryClient } from "../../test-utils";
import App from "../../App";

function renderApp() {
  return renderWithQueryClient(<App />);
}

describe("Glossary / Contextual Help UI", () => {
  it("navigates to glossary view", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("nav-glossary"));
    expect(screen.getByTestId("glossary-panel")).toBeInTheDocument();
  });

  it("renders search input in glossary view", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("nav-glossary"));
    expect(screen.getByTestId("glossary-search")).toBeInTheDocument();
  });

  it("returns to wizard from glossary view", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("nav-glossary"));
    expect(screen.queryByTestId("wizard-shell")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("nav-wizard"));
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });
});
