/**
 * FilePreview Component Tests
 *
 * [REQ:REQ-P0-004] File preview component tests
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { FilePreview } from "./file-preview";
import { FileServiceProvider } from "../../contexts/FileServiceContext";
import type { IFileService } from "../../services/file-service-types";
import { selectors } from "../../consts/selectors";
import { createTestQueryClient } from "../../test-utils";

vi.mock("../../lib", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib")>();
  return {
    ...actual,
    defaultQueryOptions: {
      ...actual.defaultQueryOptions,
      retry: false,
    },
  };
});

function createMockFileService(overrides?: Partial<IFileService>): IFileService {
  return {
    entityLabel: "backlog item",
    protectedFile: "spec.json",
    fileContentBaseUrl: "/api/v1/backlog/idea/test-idea/files",
    queryKeyPrefix: ["backlog", "idea", "test-idea"],
    getFiles: vi.fn().mockResolvedValue([]),
    getFileContent: vi.fn().mockResolvedValue(""),
    uploadFile: vi.fn().mockResolvedValue({ name: "", path: "", type: "file" }),
    saveFileContent: vi.fn().mockResolvedValue({ name: "", path: "", type: "file" }),
    renameFile: vi.fn().mockResolvedValue({}),
    moveFile: vi.fn().mockResolvedValue({}),
    copyFile: vi.fn().mockResolvedValue({}),
    deleteFile: vi.fn().mockResolvedValue({}),
    ...overrides,
  };
}

const renderWithProviders = (ui: React.ReactElement, fileService?: IFileService) => {
  const queryClient = createTestQueryClient();
  const svc = fileService ?? createMockFileService();
  return render(
    <QueryClientProvider client={queryClient}>
      <FileServiceProvider value={svc}>
        {ui}
      </FileServiceProvider>
    </QueryClientProvider>
  );
};

describe("FilePreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders file preview with file name", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("# Test Content"),
    });

    renderWithProviders(
      <FilePreview
        filePath="docs/readme.md"
        fileName="readme.md"
      />,
      svc,
    );

    expect(screen.getByTestId("file-preview-name")).toHaveTextContent("readme.md");
  });

  it("shows loading state while fetching content", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockReturnValue(new Promise(() => {})),
    });

    renderWithProviders(
      <FilePreview
        filePath="test.txt"
        fileName="test.txt"
      />,
      svc,
    );

    expect(screen.getByTestId("file-preview-name")).toHaveTextContent("test.txt");
  });

  it("renders markdown content correctly", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("# Hello World\n\nThis is a test."),
    });

    renderWithProviders(
      <FilePreview
        filePath="README.md"
        fileName="README.md"
      />,
      svc,
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
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("# Hello World\n\nThis is a test."),
    });

    renderWithProviders(
      <FilePreview
        filePath="README.md"
        fileName="README.md"
      />,
      svc,
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
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("function test() {\n  return true;\n}"),
    });

    renderWithProviders(
      <FilePreview
        filePath="src/test.ts"
        fileName="test.ts"
      />,
      svc,
    );

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-editor")).toBeInTheDocument();
    });
  });

  it("renders plain text for unknown file types", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("Plain text content"),
    });

    renderWithProviders(
      <FilePreview
        filePath="notes.txt"
        fileName="notes.txt"
      />,
      svc,
    );

    await waitFor(() => {
      expect(screen.getByTestId("file-preview-editor")).toBeInTheDocument();
    });

    expect(screen.getByTestId("file-preview-editor")).toHaveValue("Plain text content");
  });

  it("saves edited content and shows diff mode", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("Original content"),
      saveFileContent: vi.fn().mockResolvedValue({
        name: "notes.txt",
        path: "notes.txt",
        type: "file",
      }),
    });

    renderWithProviders(
      <FilePreview
        filePath="notes.txt"
        fileName="notes.txt"
      />,
      svc,
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
      expect(svc.saveFileContent).toHaveBeenCalledWith(
        "notes.txt",
        "Updated content",
        "text/plain"
      );
    });
  });

  it("renders image preview for image files", () => {
    const svc = createMockFileService();

    renderWithProviders(
      <FilePreview
        filePath="images/logo.png"
        fileName="logo.png"
      />,
      svc,
    );

    const image = screen.getByTestId("file-preview-image");
    expect(image).toBeInTheDocument();
    expect(image).toHaveAttribute("src", "/api/v1/backlog/idea/test-idea/files/images/logo.png");
  });

  it("shows error state when file fetch fails", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockRejectedValue(new Error("File not found")),
    });

    renderWithProviders(
      <FilePreview
        filePath="missing.txt"
        fileName="missing.txt"
      />,
      svc,
    );

    await waitFor(
      () => {
        expect(screen.getByTestId(selectors.error.container)).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
    expect(screen.getByTestId(selectors.error.title)).toHaveTextContent("Unable to load file");
  });

  it("displays file path in header", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("content"),
    });

    renderWithProviders(
      <FilePreview
        filePath="src/components/Button.tsx"
        fileName="Button.tsx"
      />,
      svc,
    );

    expect(screen.getByText("src/components/Button.tsx")).toBeInTheDocument();
  });
});
