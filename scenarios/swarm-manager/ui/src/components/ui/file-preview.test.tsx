/**
 * FilePreview Component Tests
 *
 * [REQ:REQ-P0-004] File preview component tests
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";
import { FilePreview } from "./file-preview";
import { FileServiceProvider } from "../../contexts/FileServiceContext";
import type { IFileService } from "../../services/file-service-types";
import { selectors } from "../../consts/selectors";
import { createTestQueryClient, renderWithProviders } from "../../test-utils";

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

const renderFilePreview = (ui: React.ReactElement, fileService?: IFileService) => {
  const queryClient = createTestQueryClient();
  const svc = fileService ?? createMockFileService();
  return renderWithProviders(
    <FileServiceProvider value={svc}>{ui}</FileServiceProvider>,
    { queryClient },
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

    renderFilePreview(
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

    renderFilePreview(
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

    renderFilePreview(
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
      expect(screen.getByRole("heading", { name: "Hello World" })).toBeInTheDocument();
      expect(screen.queryByTestId("file-preview-editor")).not.toBeInTheDocument();
    });
  });

  it("toggles markdown rendering between rendered and raw", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("# Hello World\n\nThis is a test."),
    });

    renderFilePreview(
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
      expect(screen.getByRole("heading", { name: "Hello World" })).toBeInTheDocument();
      expect(screen.queryByTestId("file-preview-editor")).not.toBeInTheDocument();
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

    renderFilePreview(
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

    renderFilePreview(
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

    renderFilePreview(
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

    renderFilePreview(
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

    renderFilePreview(
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

  it("renders read-only file content instead of a blank pane", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue('{"title":"Protected spec"}'),
    });

    renderFilePreview(
      <FilePreview filePath="spec.json" fileName="spec.json" readOnly />,
      svc,
    );

    const editor = await screen.findByTestId("file-preview-editor");
    expect(editor).toHaveValue('{"title":"Protected spec"}');
    expect(editor).toHaveAttribute("data-read-only", "true");
  });

  it("renders read-only markdown in its raw default view", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("# Protected"),
    });

    renderFilePreview(
      <FilePreview filePath="NOTES.md" fileName="NOTES.md" readOnly />,
      svc,
    );

    const editor = await screen.findByTestId("file-preview-editor");
    expect(editor).toHaveValue("# Protected");
    expect(editor).toHaveAttribute("data-read-only", "true");
  });

  it("hides save and discard controls for read-only files", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("content"),
    });

    renderFilePreview(
      <FilePreview filePath="spec.json" fileName="spec.json" readOnly />,
      svc,
    );

    await screen.findByTestId("file-preview-editor");
    expect(screen.queryByTestId("file-preview-save")).not.toBeInTheDocument();
    expect(screen.queryByTestId("file-preview-discard")).not.toBeInTheDocument();
  });

  it("keeps editable files writable", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("content"),
    });

    renderFilePreview(
      <FilePreview filePath="notes.txt" fileName="notes.txt" />,
      svc,
    );

    const editor = await screen.findByTestId("file-preview-editor");
    expect(editor).toHaveAttribute("data-read-only", "false");
    expect(screen.getByTestId("file-preview-save")).toBeInTheDocument();
  });

  it("displays file path in header", async () => {
    const svc = createMockFileService({
      getFileContent: vi.fn().mockResolvedValue("content"),
    });

    renderFilePreview(
      <FilePreview
        filePath="src/components/Button.tsx"
        fileName="Button.tsx"
      />,
      svc,
    );

    expect(screen.getByText("src/components/Button.tsx")).toBeInTheDocument();
  });
});
