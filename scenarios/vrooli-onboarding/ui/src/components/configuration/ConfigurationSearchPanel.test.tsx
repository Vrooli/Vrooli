import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, renderWithProviders, screen, waitFor } from "../../test-utils";
import { ConfigurationSearchPanel } from "./ConfigurationSearchPanel";

const calls = vi.hoisted(() => ({ search: vi.fn() }));
vi.mock("../../api/configuration", () => ({ searchConfiguration: calls.search }));

describe("ConfigurationSearchPanel", () => {
  beforeEach(() => calls.search.mockReset());

  it("renders results and forwards the selected descriptor", async () => {
    const descriptor = {
      id: "operator.mode",
      title: "Operating mode",
      purpose: "Choose how the node runs.",
      prerequisites: ["readiness"],
    };
    calls.search.mockResolvedValue({ results: [descriptor] });
    const onNavigate = vi.fn();

    renderWithProviders(<ConfigurationSearchPanel target="local" onNavigate={onNavigate} />);

    await waitFor(() => expect(screen.getByTestId("configuration-results")).toBeInTheDocument());
    expect(screen.getByText("Operating mode")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("configuration-open-operator.mode"));
    expect(onNavigate).toHaveBeenCalledWith(descriptor);
  });

  it("shows the empty state when no configuration matches", async () => {
    calls.search.mockResolvedValue({ results: [] });

    renderWithProviders(<ConfigurationSearchPanel target="local" onNavigate={vi.fn()} />);

    await waitFor(() => expect(screen.getByText(/no settings match/i)).toBeInTheDocument());
  });
});
