import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { WorkspaceStudioLoadState } from "./WorkspaceStudioLoadState";

describe("WorkspaceStudioLoadState", () => {
  it("explains when the requested workspace does not exist and links back", () => {
    renderWithProviders(<WorkspaceStudioLoadState loadError="not-found" ready={false} />);

    expect(screen.getByText(strings.workbench.workspace.unavailable)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.workspace.notFound)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: strings.workbench.workspace.back })).toHaveAttribute(
      "href",
      "/design",
    );
  });

  it("shows the transport failure instead of a loading state", () => {
    renderWithProviders(<WorkspaceStudioLoadState loadError="server unavailable" ready={false} />);

    expect(screen.getByText(strings.workbench.workspace.loadFailed)).toBeInTheDocument();
    expect(screen.getByTestId("workspace-studio").textContent).toContain("server unavailable");
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("keeps the editor busy while its workspace is still loading", () => {
    renderWithProviders(<WorkspaceStudioLoadState loadError={null} ready={false} />);

    expect(screen.getByTestId("workspace-studio")).toHaveAttribute("aria-busy", "true");
  });

  it("renders no blocking state once the workspace is ready", () => {
    const { container } = renderWithProviders(<WorkspaceStudioLoadState loadError={null} ready />);

    expect(container).toBeEmptyDOMElement();
  });
});
