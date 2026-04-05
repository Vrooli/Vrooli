/**
 * FileUpload Component Tests
 *
 * [REQ:REQ-P0-004] File upload component tests with drag-and-drop
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FileUpload } from "./file-upload";
import { FileServiceProvider } from "../../contexts/FileServiceContext";
import type { IFileService } from "../../services/file-service-types";

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

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  });

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

const createFile = (name: string, content: string = "test content"): File => {
  return new File([content], name, { type: "text/plain" });
};

describe("FileUpload", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders upload dropzone", () => {
    renderWithProviders(<FileUpload />);

    expect(screen.getByTestId("file-upload-dropzone")).toBeInTheDocument();
    expect(screen.getByText(/click to upload/i)).toBeInTheDocument();
    expect(screen.getByText(/drag and drop/i)).toBeInTheDocument();
  });

  it("handles file selection via input", async () => {
    const svc = createMockFileService({
      uploadFile: vi.fn().mockResolvedValue({
        name: "test.txt",
        path: "test.txt",
        type: "file",
        size: 12,
      }),
    });

    renderWithProviders(<FileUpload />, svc);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(svc.uploadFile).toHaveBeenCalledWith(file, undefined);
    });
  });

  it("shows upload list when files are added", async () => {
    const svc = createMockFileService({
      uploadFile: vi.fn().mockResolvedValue({
        name: "test.txt",
        path: "test.txt",
        type: "file",
        size: 12,
      }),
    });

    renderWithProviders(<FileUpload />, svc);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByTestId("file-upload-list")).toBeInTheDocument();
    });

    expect(screen.getByText("test.txt")).toBeInTheDocument();
  });

  it("shows success state after successful upload", async () => {
    const svc = createMockFileService({
      uploadFile: vi.fn().mockResolvedValue({
        name: "test.txt",
        path: "test.txt",
        type: "file",
        size: 12,
      }),
    });

    renderWithProviders(<FileUpload />, svc);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      const item = screen.getByTestId("file-upload-item-0");
      expect(item).toHaveClass("border-green-500/30");
    });
  });

  it("shows error state when upload fails", async () => {
    const svc = createMockFileService({
      uploadFile: vi.fn().mockRejectedValue(new Error("Upload failed")),
    });

    renderWithProviders(<FileUpload />, svc);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      const item = screen.getByTestId("file-upload-item-0");
      expect(item).toHaveClass("border-red-500/30");
    });

    expect(screen.getByTestId("file-upload-retry-0")).toBeInTheDocument();
  });

  it("allows retry on failed upload", async () => {
    const svc = createMockFileService({
      uploadFile: vi.fn()
        .mockRejectedValueOnce(new Error("First attempt failed"))
        .mockResolvedValueOnce({
          name: "test.txt",
          path: "test.txt",
          type: "file",
          size: 12,
        }),
    });

    renderWithProviders(<FileUpload />, svc);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByTestId("file-upload-retry-0")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("file-upload-retry-0"));

    await waitFor(() => {
      const item = screen.getByTestId("file-upload-item-0");
      expect(item).toHaveClass("border-green-500/30");
    });
  });

  it("allows removing an upload", async () => {
    const svc = createMockFileService({
      uploadFile: vi.fn().mockResolvedValue({
        name: "test.txt",
        path: "test.txt",
        type: "file",
        size: 12,
      }),
    });

    renderWithProviders(<FileUpload />, svc);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByTestId("file-upload-remove-0")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("file-upload-remove-0"));

    await waitFor(() => {
      expect(screen.queryByTestId("file-upload-item-0")).not.toBeInTheDocument();
    });
  });

  it("handles drag over visual feedback", () => {
    renderWithProviders(<FileUpload />);

    const dropzone = screen.getByTestId("file-upload-dropzone");

    fireEvent.dragOver(dropzone);
    expect(dropzone).toHaveClass("border-cyan-400");

    fireEvent.dragLeave(dropzone);
    expect(dropzone).toHaveClass("border-slate-600");
  });

  it("handles file drop", async () => {
    const svc = createMockFileService({
      uploadFile: vi.fn().mockResolvedValue({
        name: "test.txt",
        path: "test.txt",
        type: "file",
        size: 12,
      }),
    });

    renderWithProviders(<FileUpload />, svc);

    const dropzone = screen.getByTestId("file-upload-dropzone");
    const file = createFile("test.txt");

    fireEvent.drop(dropzone, {
      dataTransfer: { files: [file] },
    });

    await waitFor(() => {
      expect(svc.uploadFile).toHaveBeenCalledWith(file, undefined);
    });
  });

  it("supports uploading to a target path", async () => {
    const svc = createMockFileService({
      uploadFile: vi.fn().mockResolvedValue({
        name: "test.txt",
        path: "docs/test.txt",
        type: "file",
        size: 12,
      }),
    });

    renderWithProviders(
      <FileUpload targetPath="docs" />,
      svc,
    );

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(svc.uploadFile).toHaveBeenCalledWith(file, "docs");
    });
  });
});
