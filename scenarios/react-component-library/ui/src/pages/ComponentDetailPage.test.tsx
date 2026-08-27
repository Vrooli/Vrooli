import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";

vi.mock("@monaco-editor/react", () => ({
  __esModule: true,
  default: (props: { value?: string; onChange?: (v: string | undefined) => void }) => {
    const { value, onChange } = props;
    return (
      <textarea
        data-testid="monaco-stub"
        value={value ?? ""}
        onChange={(e) => onChange?.(e.target.value)}
      />
    );
  },
}));

vi.mock("../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/components")>();
  return {
    ...actual,
    getCatalogAsset: vi.fn(),
    getComponentExperience: vi.fn().mockResolvedValue({
      componentId: "cmp-42",
      contractId: "button",
      title: "Button",
      purpose: "Provide an action with an accessible name.",
      evidenceStatus: "available",
      evidenceMessage: "",
      states: [{ id: "primary", exampleName: "primary", description: "Primary action." }],
      claims: [
        {
          id: "action-present",
          type: "element-present",
          statement: "A named action is present.",
          tier: "machine",
          states: ["primary"],
        },
      ],
      evidence: [
        {
          claimId: "action-present",
          verdict: "passed",
          stateId: "primary",
          exampleName: "primary",
          captureRef: "https://example.test/capture",
          checkedAt: "2026-07-15T12:00:00Z",
          message: "claim proven",
          viewport: "desktop",
          viewportWidth: 1280,
          viewportHeight: 720,
        },
      ],
    }),
    componentsClient: {
      listComponents: vi.fn(),
      getComponent: vi.fn().mockResolvedValue({
        component: {
          id: "cmp-42",
          libraryId: "react-component-library:Button",
          displayName: "Button",
          description: "",
          version: "0.1.0",
          sourcePath: "Button.tsx",
          tags: [],
          updatedAt: "",
        },
      }),
      getComponentByLibraryId: vi.fn(),
      indexComponents: vi.fn(),
      getComponentContent: vi.fn().mockResolvedValue({
        content: "// hi",
        sha256: "sha-abc-123",
        sourcePath: "Button.tsx",
      }),
      updateComponentContent: vi.fn(),
    },
  };
});

vi.mock("../api/adoptions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/adoptions")>();
  return {
    ...actual,
    adoptionsClient: {
      listAdoptions: vi.fn().mockResolvedValue({ adoptions: [] }),
      listEffectiveAdoptions: vi.fn().mockResolvedValue({ adoptions: [] }),
      suggestAdoptions: vi.fn().mockResolvedValue({ suggestions: [] }),
      refreshAdoptions: vi.fn(),
    },
  };
});

import { ComponentDetailPage } from "./ComponentDetailPage";
import { componentsClient, getCatalogAsset, getComponentExperience } from "../api/components";

describe("ComponentDetailPage", () => {
  beforeEach(() => {
    vi.mocked(componentsClient.getComponentByLibraryId).mockResolvedValue(
      {} as Awaited<ReturnType<typeof componentsClient.getComponentByLibraryId>>,
    );
    vi.mocked(componentsClient.listComponents).mockResolvedValue({
      components: [],
    } as unknown as Awaited<ReturnType<typeof componentsClient.listComponents>>);
    vi.mocked(getCatalogAsset).mockResolvedValue({
      component: {
        id: "cmp-42",
        libraryId: "react-component-library:DrawerShell",
        displayName: "DrawerShell",
        dependencies: [
          { libraryId: "react-component-library:useFocusTrap", version: "1.0.0" },
          { libraryId: "react-component-library:useEscapeKey", version: "1.0.0" },
        ],
      },
    } as Awaited<ReturnType<typeof getCatalogAsset>>);
  });
  afterEach(() => cleanup());

  it("renders the editor for the component resolved by route id", async () => {
    renderWithProviders(
      <Routes>
        <Route path="/components/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components/cmp-42"] },
    );

    await waitFor(() => {
      expect(screen.getByTestId("component-detail-page")).toBeInTheDocument();
    });
    expect(screen.getByRole("tab", { name: "components.editor.files" })).toBeInTheDocument();
    expect(screen.getAllByRole("tab")).toHaveLength(6);
    expect(screen.queryByTestId("monaco-stub")).not.toBeInTheDocument();
  });

  it("shows declared behavior, evidence tier, verdict, and capture link", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route path="/components/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components/cmp-42"] },
    );

    await user.click(await screen.findByRole("tab", { name: "componentDetail.info.experience" }));
    expect(await screen.findByTestId("component-experience-panel")).toHaveTextContent(
      "A named action is present.",
    );
    expect(screen.getByTestId("component-experience-panel")).toHaveTextContent("machine");
    expect(screen.getByTestId("component-experience-panel")).toHaveTextContent("passed");
    expect(screen.getByTestId("component-experience-panel")).toHaveTextContent(
      "componentDetail.experience.identity",
    );
    expect(screen.getByTestId("component-experience-panel")).toHaveTextContent(
      "componentDetail.experience.stale",
    );
    expect(
      screen.getByRole("link", { name: "componentDetail.experience.openCapture" }),
    ).toHaveAttribute("href", "https://example.test/capture");
    expect(getComponentExperience).toHaveBeenCalledWith("cmp-42");
  });

  it("renders a missing-id message when the route has no component id", () => {
    renderWithProviders(
      <Routes>
        <Route path="/components" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components"] },
    );

    expect(screen.getByTestId("component-detail-missing-id")).toBeInTheDocument();
  });

  it("switches the information panel between overview and versions", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route path="/components/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components/cmp-42"] },
    );

    await screen.findByRole("tab", { name: "componentDetail.info.versions" });
    expect(screen.getByRole("tab", { name: "components.editor.previewMode" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await user.click(screen.getByRole("tab", { name: "componentDetail.info.versions" }));
    expect(screen.getByTestId("versions-card")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "componentDetail.info.adoptions" })).not.toBeInTheDocument();
  });

  it("shows version and adoption totals in the detail tab notification bubbles", async () => {
    vi.mocked(getCatalogAsset).mockResolvedValueOnce({
      component: {
        id: "cmp-42",
        libraryId: "react-component-library:DrawerShell",
        displayName: "DrawerShell",
        metrics: { versionCount: 3, directAdoptionCount: 2 },
      },
    } as Awaited<ReturnType<typeof getCatalogAsset>>);

    renderWithProviders(
      <Routes>
        <Route path="/components/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components/cmp-42"] },
    );

    expect(
      await screen.findByRole("tab", { name: "componentDetail.info.versions" }),
    ).toHaveTextContent("3");
    expect(screen.queryByRole("tab", { name: "componentDetail.info.adoptions" })).not.toBeInTheDocument();
  });

  it("links a component's shared hook dependencies from its overview", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route path="/components/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components/cmp-42"] },
    );

    await user.click(await screen.findByRole("tab", { name: "componentDetail.info.overview" }));
    expect(
      await screen.findByRole("link", { name: "react-component-library:useFocusTrap" }),
    ).toHaveAttribute("href", "/assets/react-component-library%3AuseFocusTrap?tab=overview");
    expect(
      screen.getByRole("link", { name: "react-component-library:useEscapeKey" }),
    ).toHaveAttribute("href", "/assets/react-component-library%3AuseEscapeKey?tab=overview");
  });

  it("restores an asset's saved tab and story when the URL has neither", async () => {
    window.localStorage.setItem(
      "rcl.asset-navigation.cmp-42",
      JSON.stringify({ tab: "files", story: "disabled" }),
    );
    try {
      renderWithProviders(
        <Routes>
          <Route path="/components/:id" element={<ComponentDetailPage />} />
        </Routes>,
        { routerEntries: ["/components/cmp-42"] },
      );
      expect(await screen.findByRole("tab", { name: "components.editor.files" })).toHaveAttribute(
        "aria-selected",
        "true",
      );
      await waitFor(() =>
        expect(window.localStorage.getItem("rcl.asset-navigation.cmp-42")).toContain("disabled"),
      );
    } finally {
      window.localStorage.removeItem("rcl.asset-navigation.cmp-42");
    }
  });

  it("opens hooks in Overview and omits the render preview tab", async () => {
    const user = userEvent.setup();
    vi.mocked(getCatalogAsset).mockResolvedValueOnce({
      component: {
        id: "hook-42",
        libraryId: "react-component-library:useFocusTrap",
        displayName: "useFocusTrap",
        assetKind: 2,
        sourcePath: "useFocusTrap.ts",
        metrics: { versionCount: 2, directAdoptionCount: 0, effectiveAdoptionCount: 1 },
      },
    } as Awaited<ReturnType<typeof getCatalogAsset>>);
    vi.mocked(componentsClient.getComponent).mockResolvedValueOnce({
      component: { id: "hook-42", libraryId: "react-component-library:useFocusTrap" },
    } as Awaited<ReturnType<typeof componentsClient.getComponent>>);

    renderWithProviders(
      <Routes>
        <Route path="/assets/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/assets/hook-42"] },
    );

    await screen.findByTestId("hook-detail-page");
    expect(
      await screen.findByRole("tab", { name: "componentDetail.info.overview" }),
    ).toHaveAttribute("aria-selected", "true");
    expect(
      screen.queryByRole("tab", { name: "components.editor.preview" }),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("hook-workspace-details")).toBeInTheDocument();
    await user.click(await screen.findByRole("tab", { name: "components.editor.files" }));
    expect(await screen.findByTestId("monaco-stub")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "componentDetail.info.versions" }));
    expect(screen.getByTestId("hook-workspace-details")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "componentDetail.info.adoptions" })).not.toBeInTheDocument();
  });

  it("renders loading while the component lookup is pending", () => {
    vi.mocked(componentsClient.getComponent).mockReturnValueOnce(new Promise(() => {}));

    renderWithProviders(
      <Routes>
        <Route path="/components/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components/cmp-pending"] },
    );

    expect(screen.getByTestId("component-detail-loading")).toBeInTheDocument();
  });

  it("resolves a bare slug through the catalog when direct lookups miss", async () => {
    vi.mocked(componentsClient.getComponent).mockRejectedValueOnce(new Error("id is not a uuid"));
    vi.mocked(componentsClient.getComponentByLibraryId).mockRejectedValueOnce(
      new Error("unknown library id"),
    );
    vi.mocked(componentsClient.listComponents).mockResolvedValueOnce({
      components: [{ id: "cmp-42", slug: "Button" }],
    } as unknown as Awaited<ReturnType<typeof componentsClient.listComponents>>);

    renderWithProviders(
      <Routes>
        <Route path="/components/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components/Button"] },
    );

    await waitFor(() => {
      expect(screen.getByTestId("component-detail-page")).toBeInTheDocument();
    });
  });

  it("renders an error when the component lookup returns no component", async () => {
    vi.mocked(componentsClient.getComponent).mockResolvedValueOnce(
      {} as Awaited<ReturnType<typeof componentsClient.getComponent>>,
    );

    renderWithProviders(
      <Routes>
        <Route path="/components/:id" element={<ComponentDetailPage />} />
      </Routes>,
      { routerEntries: ["/components/missing"] },
    );

    await waitFor(() => {
      expect(screen.getByTestId("component-detail-error")).toBeInTheDocument();
    });
  });
});
