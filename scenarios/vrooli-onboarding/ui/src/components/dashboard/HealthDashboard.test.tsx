// [REQ:REQ-P1-002] Health Dashboard UI
import { screen, fireEvent } from "@testing-library/react";
import { renderWithQueryClient } from "../../test-utils";
import App from "../../App";

function renderApp() {
  return renderWithQueryClient(<App />);
}

describe("Health Dashboard UI", () => {
  it("navigates to dashboard view and shows loading state", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("nav-dashboard"));
    expect(screen.getByTestId("health-loading")).toBeInTheDocument();
  });

  it("returns to wizard from dashboard view", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("nav-dashboard"));
    expect(screen.queryByTestId("wizard-shell")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("nav-wizard"));
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });

  it("renders navigation with dashboard link", () => {
    renderApp();
    const dashboardLink = screen.getByTestId("nav-dashboard");
    expect(dashboardLink).toBeInTheDocument();
    expect(dashboardLink).toHaveTextContent(/health dashboard/i);
  });
});
