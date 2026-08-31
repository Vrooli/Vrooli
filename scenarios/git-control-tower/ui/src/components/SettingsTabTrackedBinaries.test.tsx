import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TrackedBinariesSection, formatBytes } from "./SettingsTabTrackedBinaries";
import { jsonResponse, renderWithQueryClient } from "../test-utils";

function requestUrl(input: RequestInfo | URL) {
  if (input instanceof Request) return input.url;
  if (input instanceof URL) return input.toString();
  return input;
}

const oneBinary = {
  binaries: [
    {
      path: "scenarios/tidiness-manager/cli/cli",
      bytes: 8_460_000,
      format: "elf",
      owner_dir: "scenarios/tidiness-manager",
      ignore_pattern: "/cli/cli",
      already_ignored: false,
    },
  ],
  total_bytes: 8_460_000,
  history_warning:
    "Untracking removes these from the working tree and future commits. The bytes remain in git history, so repository size is unchanged until history is rewritten.",
};

describe("TrackedBinariesSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("lists tracked binaries with size and ignore target", async () => {
    globalThis.fetch = vi.fn(async () => jsonResponse(oneBinary)) as unknown as typeof fetch;

    renderWithQueryClient(<TrackedBinariesSection isMobile={false} repoId="repo-1" />);

    expect(await screen.findByText("scenarios/tidiness-manager/cli/cli")).toBeInTheDocument();
    // Size shows twice on purpose: once as the row's own cost, once in the
    // header total, so the panel answers "how big is this problem" at a glance.
    expect(screen.getAllByText(/8\.1 MB/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/scenarios\/tidiness-manager\/\.gitignore/)).toBeInTheDocument();
  });

  // Untracking never shrinks the repo. If the panel omitted this, users would
  // reasonably conclude the space was reclaimed.
  it("states that history is unchanged", async () => {
    globalThis.fetch = vi.fn(async () => jsonResponse(oneBinary)) as unknown as typeof fetch;

    renderWithQueryClient(<TrackedBinariesSection isMobile={false} repoId="repo-1" />);

    expect(await screen.findByText(/remain in git history/i)).toBeInTheDocument();
  });

  it("posts the untrack request with the owning gitignore target", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, _init?: RequestInit) => {
      if (requestUrl(input).includes("/tracked-binaries/untrack")) {
        return jsonResponse({ success: true, removed_from_index: true, ignore_added_to: "scenarios/tidiness-manager/.gitignore" });
      }
      return jsonResponse(oneBinary);
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(<TrackedBinariesSection isMobile={false} repoId="repo-1" />);

    fireEvent.click(await screen.findByRole("button", { name: /untrack & ignore/i }));

    await waitFor(() => {
      const call = fetchMock.mock.calls.find(([input]) => requestUrl(input).includes("/tracked-binaries/untrack"));
      expect(call).toBeTruthy();
      const body = call?.[1]?.body;
      expect(JSON.parse(typeof body === "string" ? body : JSON.stringify(body))).toEqual({
        path: "scenarios/tidiness-manager/cli/cli",
        owner_dir: "scenarios/tidiness-manager",
        ignore_pattern: "/cli/cli",
      });
    });
  });

  it("reports a clean repository instead of an empty list", async () => {
    globalThis.fetch = vi.fn(async () => jsonResponse({ binaries: [], total_bytes: 0 })) as unknown as typeof fetch;

    renderWithQueryClient(<TrackedBinariesSection isMobile={false} repoId="repo-1" />);

    expect(await screen.findByText(/no compiled binaries are tracked/i)).toBeInTheDocument();
  });
});

describe("formatBytes", () => {
  it("scales units and keeps one decimal below ten", () => {
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(8_460_000)).toBe("8.1 MB");
    expect(formatBytes(164 * 1024 * 1024)).toBe("164 MB");
  });
});
