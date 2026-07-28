import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { selectors } from "@/constants/selectors";
import { WorkflowCreationDialog } from "./WorkflowCreationDialog";

describe("WorkflowCreationDialog", () => {
  it("exposes the visual-builder choice and creates it in the selected project", () => {
    const onClose = vi.fn();
    const onSelectType = vi.fn();
    const project = {
      id: "project-1",
      name: "Demo project",
      folder_path: "scenarios/browser-automation-studio/data/projects/demo-project",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };

    render(
      <WorkflowCreationDialog
        isOpen
        onClose={onClose}
        onCreateProject={vi.fn()}
        onSelectType={onSelectType}
        preSelectedProject={project}
      />,
    );

    expect(screen.getByTestId(selectors.workflows.creation.dialog)).toBeInTheDocument();
    fireEvent.click(screen.getByTestId(selectors.workflows.creation.visualBuilder));

    expect(onSelectType).toHaveBeenCalledWith("visual", project);
    expect(onClose).toHaveBeenCalledOnce();
  });
});
