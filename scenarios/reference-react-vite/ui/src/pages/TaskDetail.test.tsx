import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TaskDetail } from "./TaskDetail";
import { renderWithProviders, createMockTask, createMockNote } from "../test-utils";
import * as api from "../lib/api";

// Mock react-router-dom hooks
const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "test-task-id" }),
    useNavigate: () => mockNavigate
  };
});

// Mock API functions
vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchTask: vi.fn(),
    updateTask: vi.fn(),
    deleteTask: vi.fn(),
    fetchNotes: vi.fn(),
    createNote: vi.fn(),
    deleteNote: vi.fn()
  };
});

const mockTask = createMockTask({
  id: "test-task-id",
  title: "Test Task Title",
  description: "Test task description",
  status: "pending"
});

const mockNotes = [
  createMockNote({ id: "note-1", content: "First note content", task_id: "test-task-id" }),
  createMockNote({ id: "note-2", content: "Second note content", task_id: "test-task-id" })
];

describe("TaskDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchTask).mockResolvedValue(mockTask);
    vi.mocked(api.fetchNotes).mockResolvedValue({ data: mockNotes, pagination: { total: 2, limit: 100, offset: 0 } });
    vi.mocked(api.updateTask).mockResolvedValue(mockTask);
    vi.mocked(api.deleteTask).mockResolvedValue(undefined);
    vi.mocked(api.createNote).mockResolvedValue(mockNotes[0]!);
    vi.mocked(api.deleteNote).mockResolvedValue(undefined);
  });

  describe("Loading State", () => {
    it("shows loading state while fetching task", () => {
      vi.mocked(api.fetchTask).mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<TaskDetail />);
      expect(screen.getByTestId("task-detail-loading")).toBeInTheDocument();
    });
  });

  describe("Error State", () => {
    it("shows error state when task not found", async () => {
      vi.mocked(api.fetchTask).mockRejectedValue(new Error("Not found"));
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("task-detail-error")).toBeInTheDocument();
      });
      expect(screen.getByText("Task not found")).toBeInTheDocument();
    });

    it("shows back link in error state", async () => {
      vi.mocked(api.fetchTask).mockRejectedValue(new Error("Not found"));
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("back-to-tasks")).toBeInTheDocument();
      });
    });
  });

  describe("Task Display [REQ:P1-001a]", () => {
    it("displays task title", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByText("Test Task Title")).toBeInTheDocument();
      });
    });

    it("displays task description", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByText("Test task description")).toBeInTheDocument();
      });
    });

    it("displays task priority", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("task-priority")).toBeInTheDocument();
      });
    });

    it("displays task status", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("task-status")).toBeInTheDocument();
      });
    });

    it("displays task creation date", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("task-created")).toBeInTheDocument();
      });
    });
  });

  describe("Task Actions", () => {
    it("renders edit button", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("edit-task")).toBeInTheDocument();
      });
    });

    it("renders delete button", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("delete-task")).toBeInTheDocument();
      });
    });

    it("renders status toggle", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("task-detail-status-toggle")).toBeInTheDocument();
      });
    });
  });

  describe("Edit Mode", () => {
    it("enters edit mode when edit button clicked", async () => {
      const user = userEvent.setup();
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("edit-task")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("edit-task"));
      expect(screen.getByTestId("edit-title-input")).toBeInTheDocument();
      expect(screen.getByTestId("edit-description-input")).toBeInTheDocument();
    });

    it("shows save and cancel buttons in edit mode", async () => {
      const user = userEvent.setup();
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("edit-task")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("edit-task"));
      expect(screen.getByTestId("save-edit")).toBeInTheDocument();
      expect(screen.getByTestId("cancel-edit")).toBeInTheDocument();
    });

    it("cancels edit mode when cancel clicked", async () => {
      const user = userEvent.setup();
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("edit-task")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("edit-task"));
      await user.click(screen.getByTestId("cancel-edit"));
      expect(screen.queryByTestId("edit-title-input")).not.toBeInTheDocument();
    });
  });

  describe("Notes Section [REQ:P1-006b]", () => {
    it("displays notes section", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("notes-section")).toBeInTheDocument();
      });
    });

    it("displays note form", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("note-form")).toBeInTheDocument();
      });
    });

    it("displays notes list", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("notes-list")).toBeInTheDocument();
      });
    });

    it("displays note items", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("note-item-note-1")).toBeInTheDocument();
        expect(screen.getByTestId("note-item-note-2")).toBeInTheDocument();
      });
    });

    it("displays notes count", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("notes-count")).toBeInTheDocument();
        expect(screen.getByText("2 notes")).toBeInTheDocument();
      });
    });
  });

  describe("Notes Empty State", () => {
    it("shows empty state when no notes", async () => {
      vi.mocked(api.fetchNotes).mockResolvedValue({ data: [], pagination: { total: 0, limit: 100, offset: 0 } });
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("notes-empty")).toBeInTheDocument();
      });
      expect(screen.getByText("No notes yet")).toBeInTheDocument();
    });
  });

  describe("Note Creation", () => {
    it("creates note when form submitted", async () => {
      const user = userEvent.setup();
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("note-form")).toBeInTheDocument();
      });
      await user.type(screen.getByTestId("note-input"), "New note content");
      await user.click(screen.getByTestId("note-submit"));
      await waitFor(() => {
        expect(api.createNote).toHaveBeenCalledWith("test-task-id", { content: "New note content" });
      });
    });

    it("disables submit when input is empty", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("note-submit")).toBeDisabled();
      });
    });
  });

  describe("Back Navigation", () => {
    it("renders back to tasks link", async () => {
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("back-to-tasks")).toBeInTheDocument();
      });
    });
  });

  describe("Delete Confirmation", () => {
    it("opens delete confirmation when delete button clicked", async () => {
      const user = userEvent.setup();
      renderWithProviders(<TaskDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("delete-task")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("delete-task"));
      await waitFor(() => {
        expect(screen.getByTestId("confirm-dialog")).toBeInTheDocument();
      });
    });
  });
});
