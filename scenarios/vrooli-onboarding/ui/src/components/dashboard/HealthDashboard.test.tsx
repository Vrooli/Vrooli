// [REQ:REQ-P1-002] Health Dashboard UI
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import App from "../../App";

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  );
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
