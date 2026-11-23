import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { PathText } from "./PathText";

const { useResearchHome } = vi.hoisted(() => ({ useResearchHome: vi.fn() }));
vi.mock("../hooks/useResearchHome", () => ({ useResearchHome }));

describe("PathText", () => {
  beforeEach(() => useResearchHome.mockReturnValue("/workspace/research"));

  it("uses the shared wrapping path/link and copy behavior for a real in-home path", () => {
    const path = "/workspace/research/topics/experience-engineering/MISSION.md";
    renderWithProviders(<PathText path={path} />);

    const link = screen.getByRole("link", { name: "~research/topics/experience-engineering/MISSION.md" });
    expect(link).toHaveAttribute("href", "/files/topics/experience-engineering/MISSION.md");
    expect(screen.getByTestId("file-path")).toHaveAttribute("data-presentation", "inline");
    expect(screen.getByRole("button", { name: "common.copyPath" })).toBeInTheDocument();
    expect(link).toHaveAttribute("title", path);
  });

  it("keeps external paths as readable text with canonical copy", () => {
    renderWithProviders(<PathText path="/tmp/build/report.json" label="Build report" />);
    expect(screen.getByText("Build report")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "common.copyPath" })).toBeInTheDocument();
  });
});
