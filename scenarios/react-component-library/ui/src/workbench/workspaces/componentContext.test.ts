import { describe, expect, it, vi } from "vitest";

import {
  DisplayDevice,
  StateStatus,
  WorkspaceProvenance,
  createWorkspace,
  forkWorkspace,
  saveWorkspace,
  type Workspace,
} from "../../api/workspaces";
import { findSlot, node, slotNode, slotsOf, type Template, type TemplateNode } from "../templates/spec";
import {
  bindComponentToSlot,
  createComponentWorkspace,
  createComponentWorkspaceFromTemplate,
  curatedWorkspaceIdForTemplate,
  withComponentSlot,
} from "./componentContext";

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return {
    ...actual,
    createWorkspace: vi.fn(),
    forkWorkspace: vi.fn(),
    saveWorkspace: vi.fn(),
  };
});

const mockedFork = vi.mocked(forkWorkspace);
const mockedSave = vi.mocked(saveWorkspace);
const mockedCreate = vi.mocked(createWorkspace);

function curatedWorkspace(): Workspace {
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.Workspace",
    id: "ws_curated_aperture",
    name: "Aperture",
    summary: "A signal room",
    provenance: WorkspaceProvenance.CURATED,
    tags: ["analytics", "template"],
    bindingScenario: "",
    bindingPage: "",
    themeJson: "{}",
    optionsJson: "{}",
    revision: 1,
    createdAt: "",
    updatedAt: "",
    originWorkspaceId: "",
    pages: [
      {
        $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage",
        id: "pg_aperture",
        name: "Aperture",
        route: "/",
        pageTypeId: "",
        sortOrder: 0,
        specJson: JSON.stringify(
          node("root", "page", {}, [node("title", "text", { text: "Original" })]),
        ),
        displays: [
          {
            $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
            id: "dp_desktop",
            device: DisplayDevice.DESKTOP,
            width: 1280,
            height: 800,
            sortOrder: 0,
            states: [
              {
                $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceState",
                id: "st_default",
                name: "Default",
                argsJson: "{}",
                scenarioDataJson: "{}",
                status: StateStatus.DRAFT,
                captureId: "",
                sortOrder: 0,
              },
            ],
          },
        ],
      },
    ],
  };
}

describe("component workspace context", () => {
  it("binds the selected component and story to the first authored slot", () => {
    const source = node("page", "page", {}, [
      node("heading", "text", { text: "Original workspace content" }),
    ]);
    const result = bindComponentToSlot(source, "cmp-button", "story-primary");

    expect(findSlot(result)?.props).toMatchObject({
      componentId: "cmp-button",
      storyId: "story-primary",
    });
    expect(source.children).toHaveLength(1);
    expect(result.children).toHaveLength(2);
  });

  it("adds a typed context section when a workspace declares no slot", () => {
    const source = node("page", "page", {}, [node("title", "text", { text: "Original" })]);
    const result = withComponentSlot(source);
    const slots: TemplateNode[] = [];
    const walk = (entry: TemplateNode) => {
      if (entry.type === "slot") slots.push(entry);
      for (const child of entry.children ?? []) walk(child);
    };
    walk(result);
    expect(slots.map((slot) => slot.role)).toEqual(["action", "metric", "list"]);
    for (const slot of slots) {
      expect(slot.accepts?.length).toBeGreaterThan(0);
    }
  });

  it("binds the preferred role when the display's slot ids differ", () => {
    const source = node("page", "page", {}, [
      slotNode("desktop-action", { role: "action", accepts: ["button"] }),
      slotNode("desktop-list", { role: "list", accepts: ["card"] }),
    ]);
    const result = bindComponentToSlot(source, "cmp-card", "story", undefined, undefined, "list");
    const bound = slotsOf(result).find((slot) => slot.props.componentId === "cmp-card");
    expect(bound?.id).toBe("desktop-list");
  });

  it("preserves an authored slot while binding the component and story", () => {
    const source = node("page", "page", {}, [slotNode("hero-slot")]);
    const result = bindComponentToSlot(source, "cmp", "story");
    expect(findSlot(result)?.id).toBe("hero-slot");
    expect(findSlot(result)?.props).toMatchObject({ componentId: "cmp", storyId: "story" });
  });

  it("maps legacy template slugs to curated workspace identities", () => {
    expect(curatedWorkspaceIdForTemplate("aperture")).toBe("ws_curated_aperture");
    expect(curatedWorkspaceIdForTemplate("ws_curated_marlow")).toBe("ws_curated_marlow");
  });

  it("forks the curated snapshot and saves the selected story inside its slot", async () => {
    const source = curatedWorkspace();
    const fork = { ...source, id: "ws_fork", provenance: WorkspaceProvenance.FORKED };
    mockedFork.mockResolvedValue(fork);
    mockedSave.mockImplementation((workspace) => Promise.resolve(workspace));

    const result = await createComponentWorkspace({
      templateWorkspace: source,
      componentId: "cmp-button",
      storyId: "story-primary",
      componentVersion: "2.4.1",
    });

    expect(mockedFork).toHaveBeenCalledWith(source.id);
    expect(mockedSave).toHaveBeenCalledWith(expect.anything(), fork.revision);
    const state = result.pages[0]?.displays[0]?.states[0];
    const savedSpec = JSON.parse(state?.argsJson ?? "{}").spec as TemplateNode;
    expect(findSlot(savedSpec)?.props).toMatchObject({
      componentId: "cmp-button",
      storyId: "story-primary",
      componentVersion: "2.4.1",
    });
    expect(JSON.stringify(savedSpec)).toContain("Original");
  });

  it("binds every saved display and state without mutating the curated source", async () => {
    const source = curatedWorkspace();
    const page = source.pages[0]!;
    const display = page.displays[0]!;
    const alternate = {
      ...display,
      id: "dp_phone",
      states: [
        {
          ...display.states[0]!,
          id: "st_phone",
          argsJson: JSON.stringify({
            theme: "dark",
            spec: node("phone-page", "page", {}, [
              node("phone-copy", "text", { text: "Phone composition" }),
              slotNode("phone-slot"),
            ]),
          }),
        },
      ],
    };
    const sourceWithVariants = {
      ...source,
      pages: [
        {
          ...page,
          displays: [
            {
              ...display,
              states: [
                {
                  ...display.states[0]!,
                  argsJson: JSON.stringify({
                    theme: "dark",
                    spec: node("desktop-page", "page", {}, [
                      node("desktop-copy", "text", { text: "Desktop composition" }),
                      slotNode("desktop-slot"),
                    ]),
                  }),
                },
                {
                  ...display.states[0]!,
                  id: "st_alternate",
                  argsJson: JSON.stringify({
                    spec: node("alternate-page", "page", {}, [slotNode("alternate-slot")]),
                  }),
                },
              ],
            },
            alternate,
          ],
        },
      ],
    };
    const fork = {
      ...sourceWithVariants,
      id: "ws_variant_fork",
      provenance: WorkspaceProvenance.FORKED,
    } as Workspace;
    mockedFork.mockResolvedValue(fork);
    mockedSave.mockImplementation((workspace) => Promise.resolve(workspace));

    const result = await createComponentWorkspace({
      templateWorkspace: sourceWithVariants,
      componentId: "cmp-card",
      storyId: "story-featured",
    });

    const states = result.pages.flatMap((entry) => entry.displays.flatMap((frame) => frame.states));
    expect(states).toHaveLength(3);
    for (const state of states) {
      const spec = JSON.parse(state.argsJson).spec as TemplateNode;
      expect(findSlot(spec)?.props).toMatchObject({
        componentId: "cmp-card",
        storyId: "story-featured",
      });
    }
    expect(JSON.parse(states[0]!.argsJson)).toHaveProperty("theme", "dark");
    expect(JSON.stringify(states.map((state) => state.argsJson))).toContain("Desktop composition");
    expect(JSON.stringify(states.map((state) => state.argsJson))).toContain("Phone composition");
    expect(sourceWithVariants.pages[0]?.displays[0]?.states[0]?.argsJson).toContain(
      "Desktop composition",
    );
    expect(sourceWithVariants.pages[0]?.displays[0]?.states[0]?.argsJson).not.toContain("cmp-card");
  });

  it("materializes an unmatched saved template into a workspace once", async () => {
    const template: Template = {
      id: "legacy-custom",
      name: "Custom page",
      tagline: "Old saved composition",
      category: "operations",
      idea: "A retained custom page",
      slotHint: "Primary action",
      display: "sans",
      radius: 14,
      palette: {
        bg: "#111111",
        surface: "#222222",
        surfaceAlt: "#333333",
        ink: "#ffffff",
        muted: "#aaaaaa",
        line: "#444444",
        accent: "#88aaff",
        accentInk: "#111111",
        accentSoft: "rgba(136,170,255,.1)",
        secondary: "#88ddcc",
      },
      spec: node("legacy-page", "page", {}, [node("legacy-title", "text", { text: "Keep me" })]),
    };
    mockedCreate.mockImplementation((input) =>
      Promise.resolve({
        ...curatedWorkspace(),
        id: "ws_imported_template",
        name: input.name,
        summary: input.summary ?? "",
        provenance: input.provenance ?? WorkspaceProvenance.AUTHORED,
        pages: input.pages ?? [],
        themeJson: input.themeJson ?? "{}",
        optionsJson: input.optionsJson ?? "{}",
      }),
    );

    const result = await createComponentWorkspaceFromTemplate({
      template,
      componentId: "cmp-button",
      storyId: "story-primary",
      componentVersion: "2.4.1",
    });

    const state = result.pages[0]?.displays[0]?.states[0];
    const savedSpec = JSON.parse(state?.argsJson ?? "{}").spec as TemplateNode;
    expect(mockedCreate).toHaveBeenCalledOnce();
    expect(findSlot(savedSpec)?.props).toMatchObject({
      componentId: "cmp-button",
      storyId: "story-primary",
    });
    expect(JSON.stringify(savedSpec)).toContain("Keep me");
  });
});
