import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";

const sources = vi.hoisted(() => ({ listAdapters: vi.fn(), setAdapterEnabled: vi.fn(), uploadArchive: vi.fn() }));
vi.mock("../../api/sources", () => ({ sourcesClient: sources, uploadArchive: sources.uploadArchive }));

import { SourcesCard } from "./SourcesCard";

describe("SourcesCard [REQ:SIG-P0-008] [REQ:SIG-P0-014]", () => {
  beforeEach(() => {
    sources.listAdapters.mockResolvedValue({ adapters: [{ adapterId: "chrome-bookmarks", riskTier: 1, enabled: true, disabledReason: "", lastError: "" }] });
    sources.setAdapterEnabled.mockResolvedValue({});
    sources.uploadArchive.mockResolvedValue({ result: { created: 2, duplicated: 1, failed: 0 } });
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("imports an operator-selected export through the tier-0 adapter", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SourcesCard />);
    expect(await screen.findByText("chrome-bookmarks")).toBeInTheDocument();
    const file = new File(["<DL><p><DT><A HREF='https://example.test'>Example</A>"], "bookmarks.html", { type: "text/html" });
    await user.upload(screen.getByLabelText("Archive for chrome-bookmarks"), file);
    await user.click(screen.getByRole("button", { name: "Import export" }));
    await waitFor(() => expect(sources.uploadArchive).toHaveBeenCalledWith("chrome-bookmarks", file));
    expect(await screen.findByText(/Imported 2; duplicates 1/)).toBeInTheDocument();
  });

  it("requires an explicit action to change adapter enablement", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SourcesCard />);
    await screen.findByText("chrome-bookmarks");
    await user.click(screen.getByRole("button", { name: "Disable" }));
    await waitFor(() => expect(sources.setAdapterEnabled).toHaveBeenCalledWith({ adapterId: "chrome-bookmarks", enabled: false }));
  });

  it("keeps a soft-blocked network adapter disabled until explicit enablement", async () => {
    sources.listAdapters.mockResolvedValue({ adapters: [{ adapterId: "x-archive", riskTier: 3, enabled: false, disabledReason: "anomalous 429 response", lastError: "429" }] });
    renderWithProviders(<SourcesCard />);
    expect(await screen.findByText(/tier 2 — explicit enablement required/)).toBeInTheDocument();
    expect(screen.getByText(/Disabled: anomalous 429 response · Last error: 429/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Import export" })).toBeDisabled();
  });
});
