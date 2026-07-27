import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { TemplateGrid } from "./TemplateGrid";

const { fetchTemplatesMock } = vi.hoisted(() => ({
  fetchTemplatesMock: vi.fn(),
}));

vi.mock("../../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api")>();
  return { ...actual, fetchTemplates: fetchTemplatesMock };
});

const templates = {
  templates: [
    {
      type: "basic",
      name: "Basic",
      description: "Single-window desktop wrapper",
      useCases: ["Internal tools"],
      features: ["Window"],
    },
    {
      type: "advanced",
      name: "Advanced",
      description: "Desktop integrations",
      useCases: ["Power users"],
      features: ["Tray", "Shortcuts", "Updates", "Menus", "Deep links"],
    },
    {
      type: "kiosk",
      name: "Kiosk",
      description: "Locked-down terminal",
      useCases: ["Signage"],
      features: ["Fullscreen"],
    },
  ],
};

describe("TemplateGrid", () => {
  const onSelect = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    fetchTemplatesMock.mockResolvedValue(templates);
  });

  it("shows available desktop templates and selects an explicitly chosen template", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <TemplateGrid selectedTemplate="basic" onSelect={onSelect} />,
    );

    expect((await screen.findAllByText("Advanced")).length).toBeGreaterThan(0);
    expect(screen.getByText("+1 more")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Kiosk/ }));

    expect(onSelect).toHaveBeenCalledWith("kiosk");
  });

  it("recommends the advanced wrapper for tray and shortcut requirements", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <TemplateGrid selectedTemplate="basic" onSelect={onSelect} />,
    );

    await user.click(
      await screen.findByRole("button", { name: "Help me decide" }),
    );
    const trayQuestion = screen.getByText(
      "Do you need a system tray icon or global shortcuts?",
    ).parentElement;
    if (!trayQuestion)
      throw new Error("Tray recommendation question is not mounted");
    const trayChoice = trayQuestion.querySelector("button");
    if (!trayChoice)
      throw new Error("Tray recommendation choice is not mounted");
    await user.click(trayChoice);

    expect(screen.getByText(/Recommended:/)).toHaveTextContent("Advanced");
    await user.click(
      screen.getByRole("button", { name: "Apply recommendation" }),
    );

    expect(onSelect).toHaveBeenCalledWith("advanced");
  });

  it("prioritizes kiosk safety requirements over other desktop conveniences", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <TemplateGrid selectedTemplate="basic" onSelect={onSelect} />,
    );

    await user.click(
      await screen.findByRole("button", { name: "Help me decide" }),
    );
    const kioskQuestion = screen.getByText(
      "Is this a locked-down kiosk or unattended terminal?",
    ).parentElement;
    if (!kioskQuestion)
      throw new Error("Kiosk recommendation question is not mounted");
    const kioskChoice = kioskQuestion.querySelector("button");
    if (!kioskChoice)
      throw new Error("Kiosk recommendation choice is not mounted");
    await user.click(kioskChoice);
    await user.click(
      screen.getByRole("button", { name: "Apply recommendation" }),
    );

    expect(onSelect).toHaveBeenCalledWith("kiosk");
  });
});
