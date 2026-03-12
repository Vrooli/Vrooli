import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ProjectDetail } from "./ProjectDetail";
import { renderWithProviders, createMockProject, createMockTask } from "../test-utils";
import * as api from "../lib/api";

// Mock react-router-dom hooks
const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "test-project-id" }),
    useNavigate: () => mockNavigate
  };
});

// Mock API functions
vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchProject: vi.fn(),
    updateProject: vi.fn(),
    deleteProject: vi.fn(),
    fetchTasks: vi.fn()
  };
});

const mockProject = createMockProject({
  id: "test-project-id",
  name: "Test Project Name",
  description: "Test project description",
  status: "active",
  color: "#3B82F6"
});

const mockTasks = [
  createMockTask({ id: "task-1", title: "First task", project_id: "test-project-id" }),
  createMockTask({ id: "task-2", title: "Second task", project_id: "test-project-id", status: "completed" })
];

describe("ProjectDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchProject).mockResolvedValue(mockProject);
    vi.mocked(api.fetchTasks).mockResolvedValue({ data: mockTasks, pagination: { total: 2, limit: 50, offset: 0 } });
    vi.mocked(api.updateProject).mockResolvedValue(mockProject);
    vi.mocked(api.deleteProject).mockResolvedValue(undefined);
  });

  describe("Loading State", () => {
    it("shows loading state while fetching project", () => {
      vi.mocked(api.fetchProject).mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<ProjectDetail />);
      expect(screen.getByTestId("project-detail-loading")).toBeInTheDocument();
    });
  });

  describe("Error State", () => {
    it("shows error state when project not found", async () => {
      vi.mocked(api.fetchProject).mockRejectedValue(new Error("Not found"));
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-detail-error")).toBeInTheDocument();
      });
      expect(screen.getByText("Project not found")).toBeInTheDocument();
    });

    it("shows back link in error state", async () => {
      vi.mocked(api.fetchProject).mockRejectedValue(new Error("Not found"));
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("back-to-projects")).toBeInTheDocument();
      });
    });
  });

  describe("Project Display [REQ:P1-001a]", () => {
    it("displays project name", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByText("Test Project Name")).toBeInTheDocument();
      });
    });

    it("displays project description", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByText("Test project description")).toBeInTheDocument();
      });
    });

    it("displays project color indicator", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-color")).toBeInTheDocument();
      });
    });

    it("displays project status", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-status")).toBeInTheDocument();
      });
    });

    it("displays project creation date", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-created")).toBeInTheDocument();
      });
    });
  });

  describe("Project Actions", () => {
    it("renders edit button", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("edit-project")).toBeInTheDocument();
      });
    });

    it("renders delete button", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("delete-project")).toBeInTheDocument();
      });
    });

    it("renders status toggle", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-detail-status-toggle")).toBeInTheDocument();
      });
    });
  });

  describe("Edit Mode", () => {
    it("enters edit mode when edit button clicked", async () => {
      const user = userEvent.setup();
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("edit-project")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("edit-project"));
      expect(screen.getByTestId("edit-name-input")).toBeInTheDocument();
      expect(screen.getByTestId("edit-description-input")).toBeInTheDocument();
    });

    it("shows save and cancel buttons in edit mode", async () => {
      const user = userEvent.setup();
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("edit-project")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("edit-project"));
      expect(screen.getByTestId("save-edit")).toBeInTheDocument();
      expect(screen.getByTestId("cancel-edit")).toBeInTheDocument();
    });

    it("cancels edit mode when cancel clicked", async () => {
      const user = userEvent.setup();
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("edit-project")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("edit-project"));
      await user.click(screen.getByTestId("cancel-edit"));
      expect(screen.queryByTestId("edit-name-input")).not.toBeInTheDocument();
    });
  });

  describe("Tasks Section [REQ:P1-006b]", () => {
    it("displays tasks section", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-tasks-section")).toBeInTheDocument();
      });
    });

    it("displays tasks list", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-tasks-list")).toBeInTheDocument();
      });
    });

    it("displays task items", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-task-task-1")).toBeInTheDocument();
        expect(screen.getByTestId("project-task-task-2")).toBeInTheDocument();
      });
    });

    it("displays tasks count", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-tasks-count")).toBeInTheDocument();
        expect(screen.getByText("2 tasks in this project")).toBeInTheDocument();
      });
    });

    it("shows view all tasks button", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("view-all-tasks")).toBeInTheDocument();
      });
    });
  });

  describe("Tasks Empty State", () => {
    it("shows empty state when no tasks", async () => {
      vi.mocked(api.fetchTasks).mockResolvedValue({ data: [], pagination: { total: 0, limit: 50, offset: 0 } });
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-tasks-empty")).toBeInTheDocument();
      });
      expect(screen.getByText("No tasks in this project")).toBeInTheDocument();
    });
  });

  describe("Back Navigation", () => {
    it("renders back to projects link", async () => {
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("back-to-projects")).toBeInTheDocument();
      });
    });
  });

  describe("Delete Confirmation", () => {
    it("opens delete confirmation when delete button clicked", async () => {
      const user = userEvent.setup();
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("delete-project")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("delete-project"));
      await waitFor(() => {
        expect(screen.getByTestId("confirm-dialog")).toBeInTheDocument();
      });
    });
  });

  describe("Status Cycling", () => {
    it("calls updateProject when status toggle clicked", async () => {
      const user = userEvent.setup();
      renderWithProviders(<ProjectDetail />);
      await waitFor(() => {
        expect(screen.getByTestId("project-detail-status-toggle")).toBeInTheDocument();
      });
      await user.click(screen.getByTestId("project-detail-status-toggle"));
      await waitFor(() => {
        expect(api.updateProject).toHaveBeenCalledWith("test-project-id", { status: "paused" });
      });
    });
  });
});
