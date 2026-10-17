import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithQueryClient } from "../test-utils";
import { FILE_HISTORY_KEY, SEARCH_MODE_KEY } from "../lib/fileSearchUtils";
import type { ContentSearchMatch, FileInfo } from "../lib/api";
import { FileSearchModal } from "./FileSearchModal";

const hookMocks = vi.hoisted(() => ({
  useFileSearch: vi.fn(),
  useContentSearch: vi.fn(),
}));

vi.mock("../lib/hooks", () => hookMocks);

const files: FileInfo[] = [
  {
    path: "src/components/FileSearchModal.tsx",
    language: "typescript",
    status: "tracked",
  },
  {
    path: "api/main.go",
    language: "go",
    status: "untracked",
  },
];

const matches: ContentSearchMatch[] = [
  {
    path: "src/components/FileSearchModal.tsx",
    line_number: 42,
    content: "const searchQuery = query.trim();",
  },
  {
    path: "src/components/FileSearchModal.tsx",
    line_number: 88,
    content: "return searchQuery.length > 1;",
  },
  {
    path: "api/main.go",
    line_number: 12,
    content: "func searchRepository() {}",
  },
];

describe("FileSearchModal", () => {
  beforeEach(() => {
    Element.prototype.scrollIntoView = vi.fn();
    hookMocks.useFileSearch.mockReturnValue({
      data: { files, truncated: false, cancelled: false },
      isLoading: false,
      error: null,
    });
    hookMocks.useContentSearch.mockReturnValue({
      data: { matches, total: matches.length, truncated: false, cancelled: false },
      isLoading: false,
      error: null,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders recent and repository file results, then records selected files in history", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const onSelectFile = vi.fn();
    localStorage.setItem(FILE_HISTORY_KEY, JSON.stringify(["docs/plan.md"]));

    const { unmount } = renderWithQueryClient(
      <FileSearchModal
        isOpen
        repoId="repo-1"
        onClose={onClose}
        onSelectFile={onSelectFile}
      />,
    );

    expect(screen.getByTestId("file-search-modal")).toBeInTheDocument();
    expect(hookMocks.useFileSearch).toHaveBeenLastCalledWith(undefined, false, true, "repo-1");
    expect(screen.getByText("Recent Files")).toBeInTheDocument();

    await user.click(screen.getByRole("option", { name: /plan\.mddocs/i }));
    expect(onSelectFile).toHaveBeenCalledWith("docs/plan.md");
    expect(onClose).toHaveBeenCalledTimes(1);

    unmount();
    onClose.mockClear();
    onSelectFile.mockClear();
    renderWithQueryClient(
      <FileSearchModal
        isOpen
        repoId="repo-1"
        onClose={onClose}
        onSelectFile={onSelectFile}
      />,
    );

    await user.click(screen.getByRole("option", { name: /FileSearchModal\.tsxsrc\/components/i }));
    expect(onSelectFile).toHaveBeenCalledWith("src/components/FileSearchModal.tsx");
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(JSON.parse(localStorage.getItem(FILE_HISTORY_KEY) ?? "[]")).toEqual([
      "src/components/FileSearchModal.tsx",
      "docs/plan.md",
    ]);
  });

  it("switches to content mode, forwards search options, and selects line-level matches", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const onSelectFile = vi.fn();

    renderWithQueryClient(
      <FileSearchModal
        isOpen
        repoId="repo-1"
        onClose={onClose}
        onSelectFile={onSelectFile}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Content" }));
    expect(localStorage.getItem(SEARCH_MODE_KEY)).toBe("content");

    await user.click(screen.getByTitle("Case sensitive"));
    await user.click(screen.getByTitle("Whole word"));
    await user.click(screen.getByTitle("Regular expression"));
    await user.click(screen.getByRole("button", { name: /Filters/i }));
    await user.type(screen.getByPlaceholderText("*.go, *.ts"), "*.tsx");
    await user.type(screen.getByPlaceholderText("*.test.ts"), "*.test.tsx");
    await user.type(screen.getByTestId("file-search-input"), "search");

    await waitFor(() => {
      expect(hookMocks.useContentSearch).toHaveBeenLastCalledWith(
        "search",
        {
          case_sensitive: true,
          whole_word: true,
          regex: true,
          include: "*.tsx",
          exclude: "*.test.tsx",
        },
        true,
        "repo-1",
      );
    });

    expect(screen.getByText("3 matches")).toBeInTheDocument();
    const firstGroup = screen.getByRole("button", {
      name: /src\/components\/FileSearchModal\.tsx\(2 matches\)/i,
    });
    expect(firstGroup).toBeInTheDocument();

    const firstMatch = within(screen.getByText("42").closest("button") as HTMLElement);
    await user.click(firstMatch.getByText(/const/i));

    expect(onSelectFile).toHaveBeenCalledWith("src/components/FileSearchModal.tsx", 42);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
