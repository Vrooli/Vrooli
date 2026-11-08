import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import { DocsPage } from ".";

const fetchMock = vi.fn();

const docsManifest = {
  version: "1",
  title: "Test Genie Docs",
  defaultDocument: "QUICKSTART.md",
  sections: [
    {
      id: "requirements",
      title: "Requirements",
      documents: [
        { path: "requirements/STATUS_MODEL.md", title: "Status Model" },
        { path: "requirements/IMPROVING_COVERAGE.md", title: "Improving Coverage" },
      ],
    },
  ],
};

function jsonResponse(payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function renderDocsPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <DocsPage />
    </QueryClientProvider>,
  );
}

describe("DocsPage", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      const url = new URL(String(input), window.location.origin);
      if (url.pathname.endsWith("/docs/manifest")) {
        return Promise.resolve(jsonResponse(docsManifest));
      }

      const path = url.searchParams.get("path");
      if (path === "requirements/STATUS_MODEL.md") {
        return Promise.resolve(jsonResponse({
          content: "# Requirement Status Model\n\nA failed validation keeps a requirement visible as failed.",
        }));
      }
      return Promise.resolve(jsonResponse({ content: "# Quickstart\n\nStart with a scenario test." }));
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("searches for requirement guidance, opens it, and returns to the default guide", async () => {
    const user = userEvent.setup();
    renderDocsPage();

    expect(await screen.findByRole("heading", { name: "Quickstart" })).toBeInTheDocument();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/docs/content?path=QUICKSTART.md"),
      expect.any(Object),
    ));

    await user.type(screen.getByPlaceholderText("Search docs..."), "Status Model");
    const requirementsSection = screen.getByRole("button", { name: "Requirements" });
    await user.click(requirementsSection);

    const statusModel = screen.getByRole("button", { name: "Status Model" });
    expect(screen.queryByRole("button", { name: "Improving Coverage" })).not.toBeInTheDocument();
    await user.click(statusModel);

    expect(await screen.findByRole("heading", { name: "Requirement Status Model" })).toBeInTheDocument();
    expect(screen.getByText("A failed validation keeps a requirement visible as failed.")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("path=requirements%2FSTATUS_MODEL.md"),
      expect.any(Object),
    );

    await user.click(screen.getByRole("button", { name: "Docs" }));
    expect(await screen.findByRole("heading", { name: "Quickstart" })).toBeInTheDocument();
  });
});
