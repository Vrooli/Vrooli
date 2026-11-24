import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, renderWithProviders, screen, waitFor } from "../../test-utils";
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

  it("retains the selected target's results and search draft through refresh loss and retry", async () => {
    const descriptor = {
      id: "operator.mode",
      title: "Operating mode",
      purpose: "Choose how the node runs.",
      prerequisites: [],
    };
    let providerUnavailable = false;
    calls.search.mockImplementation((search: string) => {
      if (search === "") return Promise.resolve({ results: [] });
      if (providerUnavailable) return Promise.reject(new Error("selected target temporarily unavailable"));
      return Promise.resolve({ results: [descriptor] });
    });
    renderWithProviders(<ConfigurationSearchPanel target="node-7" onNavigate={vi.fn()} />);
    const input = screen.getByTestId("configuration-search");
    await waitFor(() => expect(calls.search).toHaveBeenCalledWith("", "node-7"));

    fireEvent.change(input, { target: { value: "operating mode" } });
    await waitFor(() => expect(screen.getByText("Operating mode")).toBeInTheDocument());

    providerUnavailable = true;
    fireEvent.click(screen.getByRole("button", { name: "Refresh results" }));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Operating mode")).toBeInTheDocument();
    expect(input).toHaveValue("operating mode");

    providerUnavailable = false;
    fireEvent.click(screen.getByRole("button", { name: "Retry search" }));
    await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
    expect(screen.getByText("Operating mode")).toBeInTheDocument();
    expect(calls.search).toHaveBeenLastCalledWith("operating mode", "node-7");
  });

  it("does not show cached search results for a different target", async () => {
    const local = {
      id: "operator.mode",
      title: "Local operating mode",
      purpose: "Choose how the node runs.",
      prerequisites: [],
    };
    const remote = { ...local, title: "Remote operating mode" };
    let resolveLocal!: (response: { results: typeof local[] }) => void;
    calls.search.mockImplementation((_search: string, target: string) => target === "local"
      ? new Promise((resolve) => { resolveLocal = resolve; })
      : Promise.resolve({ results: [remote] }));
    const view = renderWithProviders(<ConfigurationSearchPanel target="local" onNavigate={vi.fn()} />);
    await waitFor(() => expect(calls.search).toHaveBeenCalledWith("", "local"));

    view.rerender(<ConfigurationSearchPanel target="node-7" onNavigate={vi.fn()} />);
    expect(screen.queryByText("Local operating mode")).not.toBeInTheDocument();
    expect(await screen.findByText("Remote operating mode")).toBeInTheDocument();
    await act(async () => { resolveLocal({ results: [local] }); });
    expect(screen.getByText("Remote operating mode")).toBeInTheDocument();
    expect(screen.queryByText("Local operating mode")).not.toBeInTheDocument();
  });
});
