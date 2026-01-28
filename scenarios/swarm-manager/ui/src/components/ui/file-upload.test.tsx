/**
 * FileUpload Component Tests
 *
 * [REQ:REQ-P0-004] File upload component tests with drag-and-drop
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FileUpload } from "./file-upload";

// Mock the ideasService
vi.mock("../../services", () => ({
  ideasService: {
    uploadFile: vi.fn(),
  },
}));

// Import after mock
import { ideasService } from "../../services";

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

const renderWithProviders = (ui: React.ReactElement) => {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
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
    renderWithProviders(<FileUpload ideaName="test-idea" />);

    expect(screen.getByTestId("file-upload-dropzone")).toBeInTheDocument();
    expect(screen.getByText(/click to upload/i)).toBeInTheDocument();
    expect(screen.getByText(/drag and drop/i)).toBeInTheDocument();
  });

  it("handles file selection via input", async () => {
    vi.mocked(ideasService.uploadFile).mockResolvedValue({
      name: "test.txt",
      path: "test.txt",
      type: "file",
      size: 12,
    });

    renderWithProviders(<FileUpload ideaName="test-idea" />);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(ideasService.uploadFile).toHaveBeenCalledWith("test-idea", file, undefined);
    });
  });

  it("shows upload list when files are added", async () => {
    vi.mocked(ideasService.uploadFile).mockResolvedValue({
      name: "test.txt",
      path: "test.txt",
      type: "file",
      size: 12,
    });

    renderWithProviders(<FileUpload ideaName="test-idea" />);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByTestId("file-upload-list")).toBeInTheDocument();
    });

    expect(screen.getByText("test.txt")).toBeInTheDocument();
  });

  it("shows success state after successful upload", async () => {
    vi.mocked(ideasService.uploadFile).mockResolvedValue({
      name: "test.txt",
      path: "test.txt",
      type: "file",
      size: 12,
    });

    renderWithProviders(<FileUpload ideaName="test-idea" />);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      const item = screen.getByTestId("file-upload-item-0");
      expect(item).toHaveClass("border-green-500/30");
    });
  });

  it("shows error state when upload fails", async () => {
    vi.mocked(ideasService.uploadFile).mockRejectedValue(new Error("Upload failed"));

    renderWithProviders(<FileUpload ideaName="test-idea" />);

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
    vi.mocked(ideasService.uploadFile)
      .mockRejectedValueOnce(new Error("First attempt failed"))
      .mockResolvedValueOnce({
        name: "test.txt",
        path: "test.txt",
        type: "file",
        size: 12,
      });

    renderWithProviders(<FileUpload ideaName="test-idea" />);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    // Wait for error state
    await waitFor(() => {
      expect(screen.getByTestId("file-upload-retry-0")).toBeInTheDocument();
    });

    // Click retry
    fireEvent.click(screen.getByTestId("file-upload-retry-0"));

    // Should succeed now
    await waitFor(() => {
      const item = screen.getByTestId("file-upload-item-0");
      expect(item).toHaveClass("border-green-500/30");
    });
  });

  it("allows removing an upload", async () => {
    vi.mocked(ideasService.uploadFile).mockResolvedValue({
      name: "test.txt",
      path: "test.txt",
      type: "file",
      size: 12,
    });

    renderWithProviders(<FileUpload ideaName="test-idea" />);

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
    renderWithProviders(<FileUpload ideaName="test-idea" />);

    const dropzone = screen.getByTestId("file-upload-dropzone");

    // Should not have drag class initially
    expect(dropzone).not.toHaveClass("border-cyan-400");

    // Simulate drag over
    fireEvent.dragOver(dropzone);

    // Should now have drag class
    expect(dropzone).toHaveClass("border-cyan-400");

    // Simulate drag leave
    fireEvent.dragLeave(dropzone);

    // Should remove drag class
    expect(dropzone).not.toHaveClass("border-cyan-400");
  });

  it("handles drop event", async () => {
    vi.mocked(ideasService.uploadFile).mockResolvedValue({
      name: "dropped.txt",
      path: "dropped.txt",
      type: "file",
      size: 12,
    });

    renderWithProviders(<FileUpload ideaName="test-idea" />);

    const dropzone = screen.getByTestId("file-upload-dropzone");
    const file = createFile("dropped.txt");

    const dataTransfer = {
      files: [file],
    };

    fireEvent.drop(dropzone, { dataTransfer });

    await waitFor(() => {
      expect(ideasService.uploadFile).toHaveBeenCalledWith("test-idea", file, undefined);
    });
  });

  it("calls onUploadComplete after successful upload", async () => {
    vi.mocked(ideasService.uploadFile).mockResolvedValue({
      name: "test.txt",
      path: "test.txt",
      type: "file",
      size: 12,
    });

    const onUploadComplete = vi.fn();

    renderWithProviders(
      <FileUpload ideaName="test-idea" onUploadComplete={onUploadComplete} />
    );

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(onUploadComplete).toHaveBeenCalled();
    });
  });

  it("passes targetPath to service", async () => {
    vi.mocked(ideasService.uploadFile).mockResolvedValue({
      name: "test.txt",
      path: "docs/test.txt",
      type: "file",
      size: 12,
    });

    renderWithProviders(<FileUpload ideaName="test-idea" targetPath="docs" />);

    const input = screen.getByTestId("file-upload-input");
    const file = createFile("test.txt");

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(ideasService.uploadFile).toHaveBeenCalledWith("test-idea", file, "docs");
    });
  });
});
