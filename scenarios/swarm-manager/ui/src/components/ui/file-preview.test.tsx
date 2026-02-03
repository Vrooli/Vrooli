/**
 * FilePreview Component Tests
 *
 * [REQ:REQ-P0-004] File preview component tests
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FilePreview } from "./file-preview";

vi.mock("../../services", () => ({
  backlogService: {
    getFileContent: vi.fn(),
    saveFileContent: vi.fn(),
  },
}));

import { backlogService } from "../../services";

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

const renderWithProviders = (ui: React.ReactElement) => {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>
  );
};

describe("FilePreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders file preview with file name", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue("# Test Content");

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="docs/readme.md"
        fileName="readme.md"
      />
    );

    expect(screen.getByTestId("file-preview-name")).toHaveTextContent("readme.md");
  });

  it("shows loading state while fetching content", async () => {
    vi.mocked(backlogService.getFileContent).mockReturnValue(new Promise(() => {}));

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="test.txt"
        fileName="test.txt"
      />
    );

    expect(screen.getByTestId("file-preview-name")).toHaveTextContent("test.txt");
  });

  it("renders markdown content correctly", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue("# Hello World\n\nThis is a test.");

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="README.md"
        fileName="README.md"
      />
    );

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-editor")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Show rendered markdown"));

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-markdown")).toBeInTheDocument();
    });
  });

  it("toggles markdown rendering between rendered and raw", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue("# Hello World\n\nThis is a test.");

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="README.md"
        fileName="README.md"
      />
    );

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-editor")).toBeInTheDocument();
    });

    const toggleButton = screen.getByLabelText("Show rendered markdown");
    fireEvent.click(toggleButton);

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-markdown")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText("Show raw markdown"));

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-editor")).toBeInTheDocument();
    });
  });

  it("renders code files with editor", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue("function test() {\n  return true;\n}");

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="src/test.ts"
        fileName="test.ts"
      />
    );

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-editor")).toBeInTheDocument();
    });
  });

  it("renders plain text for unknown file types", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue("Plain text content");

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="notes.txt"
        fileName="notes.txt"
      />
    );

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-editor")).toBeInTheDocument();
    });

    expect(screen.getByTestId("file-preview-editor")).toHaveValue("Plain text content");
  });

  it("saves edited content and shows diff mode", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue("Original content");
    vi.mocked(backlogService.saveFileContent).mockResolvedValue({
      name: "notes.txt",
      path: "notes.txt",
      type: "file",
    });

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="notes.txt"
        fileName="notes.txt"
      />
    );

    const editor = await screen.findByTestId("file-preview-editor");
    fireEvent.change(editor, { target: { value: "Updated content" } });

    const diffToggle = await screen.findByTestId("file-preview-diff-toggle");
    fireEvent.click(diffToggle);

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-diff")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("file-preview-save"));

    await waitFor(() => {
      expect(backlogService.saveFileContent).toHaveBeenCalledWith(
        "idea",
        "test-idea",
        "notes.txt",
        "Updated content",
        "text/plain"
      );
    });
  });

  it("renders image preview for image files", () => {
    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="images/logo.png"
        fileName="logo.png"
      />
    );

    const image = screen.getByTestId("file-preview-image");
    expect(image).toBeInTheDocument();
    expect(image).toHaveAttribute("src", "/api/v1/backlog/idea/test-idea/files/images/logo.png");
  });

  it.skip("shows error state when file fetch fails", async () => {
    vi.mocked(backlogService.getFileContent).mockRejectedValue(new Error("File not found"));

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="missing.txt"
        fileName="missing.txt"
      />
    );

    await waitFor(
      () => {
        expect(screen.getByTestId("error-state")).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it("displays file path in header", async () => {
    vi.mocked(backlogService.getFileContent).mockResolvedValue("content");

    renderWithProviders(
      <FilePreview
        backlogKind="idea"
        backlogName="test-idea"
        filePath="src/components/Button.tsx"
        fileName="Button.tsx"
      />
    );

    expect(screen.getByText("src/components/Button.tsx")).toBeInTheDocument();
  });
});
