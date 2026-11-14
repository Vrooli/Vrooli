import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";

const api = vi.hoisted(() => ({ auditRecords: vi.fn() }));
vi.mock("../api/deviceControl", () => api);

import { EvidencePage } from "./EvidencePage";

describe("EvidencePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the empty state after loading an empty audit history", async () => {
    api.auditRecords.mockResolvedValue({ records: [] });

    renderWithProviders(<EvidencePage />);

    expect(await screen.findByTestId(selectors.pages.evidence)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.evidence.noVerbs)).toBeInTheDocument();
  });

  it("renders each retained audit record", async () => {
    api.auditRecords.mockResolvedValue({
      records: [{ id: "record-1", actor: "operator", device_id: "desktop-1", verb: "click", outcome: "passed", created_at: "2026-09-07T21:00:00Z" }],
    });

    renderWithProviders(<EvidencePage />);

    await waitFor(() => expect(screen.getByText("desktop-1")).toBeInTheDocument());
    expect(screen.getByText("operator")).toBeInTheDocument();
    expect(screen.getByText("click")).toBeInTheDocument();
    expect(screen.getByText("passed")).toBeInTheDocument();
  });

  it("shows the API error instead of the empty state", async () => {
    api.auditRecords.mockRejectedValue(new Error("audit unavailable"));

    renderWithProviders(<EvidencePage />);

    expect(await screen.findByRole("alert")).toHaveTextContent("audit unavailable");
    expect(screen.queryByText(strings.pages.evidence.noVerbs)).not.toBeInTheDocument();
  });
});
