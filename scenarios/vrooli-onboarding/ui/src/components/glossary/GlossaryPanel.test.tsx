// [REQ:REQ-P2-004] Contextual Help UI
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
