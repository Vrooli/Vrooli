import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
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

import { ComponentDetailPage } from "./ComponentDetailPage";
import { componentsClient } from "../api/components";

describe("ComponentDetailPage", () => {
  beforeEach(() => {
    // ensure clean state
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
    await waitFor(() => {
      expect(screen.getByTestId<HTMLTextAreaElement>("monaco-stub").value).toBe("// hi");
    });
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
