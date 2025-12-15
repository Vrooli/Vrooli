import "@testing-library/jest-dom";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import App from "./App";
import { selectors } from "./consts/selectors";

function jsonResponse(payload: unknown, status = 200) {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "Content-Type": "application/json" }
  });
}

test("validates a manifest and renders the result", async () => {
  // [REQ:STC-P0-001] cloud manifest validation is exposed through the UI
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.includes("/health")) {
      return jsonResponse({ status: "healthy", service: "Scenario To Cloud API", timestamp: "2025-01-01T00:00:00Z" });
    }
    if (url.includes("/manifest/validate")) {
      return jsonResponse({ valid: true, issues: [], timestamp: "2025-01-01T00:00:00Z" });
    }
    if (url.includes("/bundle/build")) {
      return jsonResponse({ artifact: { path: "/tmp/mini.tar.gz", sha256: "abc", size_bytes: 123 }, timestamp: "2025-01-01T00:00:00Z" });
    }
    return jsonResponse({ error: { message: "not found" } }, 404);
  });
  vi.stubGlobal("fetch", fetchMock);

  const queryClient = new QueryClient();
  render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  );

  const input = await screen.findByTestId(selectors.manifest.input);
  fireEvent.change(input, {
    target: {
      value:
        `{"version":"1.0.0","target":{"type":"vps","vps":{"host":"203.0.113.10"}},"scenario":{"id":"landing-page-business-suite"},"dependencies":{"scenarios":["landing-page-business-suite"],"resources":[]},"bundle":{"include_packages":true,"include_autoheal":true},"ports":{"ui":3000,"api":3001,"ws":3002},"edge":{"domain":"example.com","caddy":{"enabled":true}}}`
    }
  });

  await userEvent.click(screen.getByTestId(selectors.manifest.validateButton));

  expect(await screen.findByTestId(selectors.manifest.validateResult)).toHaveTextContent(
    "Valid manifest"
  );

  await userEvent.click(screen.getByTestId(selectors.manifest.bundleBuildButton));

  expect(await screen.findByTestId(selectors.manifest.bundleBuildResult)).toHaveTextContent(
    "/tmp/mini.tar.gz"
  );
});
