import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("../../api/manifest", () => ({
  manifestClient: {
    listDomains: vi.fn(),
  },
}));

import { manifestClient } from "../../api/manifest";
import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "../../test-utils/a11y";
import { selectors } from "../../consts/selectors";
import { GraphFilterBar } from "./GraphFilterBar";

type DomainsResult = Awaited<ReturnType<typeof manifestClient.listDomains>>;

afterEach(() => {
  cleanup();
  vi.mocked(manifestClient.listDomains).mockReset();
});

function setDomains(domains: string[]) {
  vi.mocked(manifestClient.listDomains).mockResolvedValue({
    domains: domains.map((name) => ({ name, paths: [] })),
  } as unknown as DomainsResult);
}

describe("GraphFilterBar", () => {
  it("renders one chip per declared domain plus an All chip", async () => {
    setDomains(["graph", "manifest", "conflicts"]);
    renderWithProviders(
      <GraphFilterBar scenario="demo" selected={new Set()} onChange={() => undefined} />,
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.graph.filterBar.root)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.features.graph.filterBar.allChip)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.graph.filterBar.chip({ key: "graph" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.features.graph.filterBar.chip({ key: "conflicts" })),
    ).toBeInTheDocument();
  });

  it("toggles a domain by calling onChange with the new set", async () => {
    setDomains(["graph"]);
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderWithProviders(
      <GraphFilterBar scenario="demo" selected={new Set()} onChange={onChange} />,
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.graph.filterBar.root)).toBeInTheDocument(),
    );
    await user.click(
      screen.getByTestId(selectors.features.graph.filterBar.chip({ key: "graph" })),
    );
    expect(onChange).toHaveBeenCalledTimes(1);
    const next = onChange.mock.calls[0]?.[0] as ReadonlySet<string>;
    expect(next.has("graph")).toBe(true);
  });

  it("clears the selection when the All chip is pressed", async () => {
    setDomains(["graph"]);
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderWithProviders(
      <GraphFilterBar
        scenario="demo"
        selected={new Set(["graph"])}
        onChange={onChange}
      />,
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.graph.filterBar.root)).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId(selectors.features.graph.filterBar.allChip));
    expect(onChange).toHaveBeenCalledTimes(1);
    const next = onChange.mock.calls[0]?.[0] as ReadonlySet<string>;
    expect(next.size).toBe(0);
  });

  it("renders an empty-state when no domains are declared", async () => {
    setDomains([]);
    renderWithProviders(
      <GraphFilterBar scenario="demo" selected={new Set()} onChange={() => undefined} />,
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.graph.filterBar.empty)).toBeInTheDocument(),
    );
  });

  it("has no axe-core violations with domains loaded", async () => {
    setDomains(["graph", "manifest"]);
    const { container } = renderWithProviders(
      <GraphFilterBar scenario="demo" selected={new Set()} onChange={() => undefined} />,
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.graph.filterBar.root)).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
