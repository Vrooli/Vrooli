import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import type { ComponentBundle, WorkbenchStory } from "../data";
import { useComponentBundle } from "../data";
import { PreviewPage } from "./PreviewPage";

vi.mock("../data", () => ({
  assetVersion: (asset: { latestVersion?: string; version?: string }) => asset.latestVersion || asset.version || "",
  useComponentBundle: vi.fn(),
}));
vi.mock("../component/DeviceFrame", () => ({
  DeviceFrame: (props: { device: string; width: number; height: number; scale: number; label: string; children: React.ReactNode }) => (
    <section data-testid="device-frame" aria-label={props.label} data-device={props.device} data-width={props.width} data-height={props.height} data-scale={props.scale}>
      {props.children}
    </section>
  ),
}));
vi.mock("../component/FixtureFrame", () => ({
  FixtureFrame: (props: { assetId: string; storyId: string; version: string; title: string }) => (
    <output data-testid="preview-fixture" data-asset={props.assetId} data-story={props.storyId} data-version={props.version} aria-label={props.title} />
  ),
}));

const stories: WorkbenchStory[] = [
  { key: "asset@1:primary", id: "primary", name: "Primary", role: "main", args: {}, fieldsJson: "{}", version: "1.0.0", sourcePath: "Button.tsx", contractId: "asset@1" },
  { key: "asset@1:secondary", id: "secondary", name: "Secondary", role: "alternate", args: {}, fieldsJson: "{}", version: "1.0.0", sourcePath: "Button.tsx", contractId: "asset@1" },
];
const bundle: ComponentBundle = {
  asset: { id: "asset-7", libraryId: "react-component-library:Button", displayName: "Button", version: "1.0.0", latestVersion: "1.2.0" } as ComponentBundle["asset"],
  versions: [],
  stories,
  testReports: [],
  partial: [],
  experience: {
    componentId: "asset-7", libraryId: "react-component-library:Button", version: "1.2.0", contractId: "asset@1", title: "Button", purpose: "Action", evidenceStatus: "fresh", evidenceMessage: "",
    states: [], claims: [], evidence: [],
  },
};

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{`${location.pathname}${location.search}`}</output>;
}

function renderPreview(path = "/assets/asset-7/preview") {
  return renderWithProviders(
    <>
      <LocationProbe />
      <Routes>
        <Route path="/assets/:id/preview" element={<PreviewPage />} />
        <Route path="/assets/:id" element={<div data-testid="asset-route">Asset route</div>} />
      </Routes>
    </>,
    { routerEntries: [path] },
  );
}

describe("PreviewPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useComponentBundle).mockReturnValue({ data: bundle, isLoading: false, isError: false, error: null } as never);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders the selected story and viewport from the shared URL, preserving other query state", () => {
    renderPreview("/assets/asset-7/preview?device=tablet&story=secondary&keep=1");

    expect(screen.getByTestId("device-frame")).toHaveAttribute("data-device", "tablet");
    expect(screen.getByTestId("device-frame")).toHaveAttribute("data-width", "834");
    expect(screen.getByTestId("preview-fixture")).toHaveAttribute("data-story", "secondary");
    expect(screen.getByTestId("preview-fixture")).toHaveAttribute("data-version", "1.2.0");

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.preview.phone }));
    expect(screen.getByTestId("device-frame")).toHaveAttribute("data-device", "phone");
    expect(screen.getByTestId("device-frame")).toHaveAttribute("data-width", "390");
    expect(screen.getByTestId("location")).toHaveTextContent("device=phone");
    expect(screen.getByTestId("location")).toHaveTextContent("story=secondary");
    expect(screen.getByTestId("location")).toHaveTextContent("keep=1");

    fireEvent.change(screen.getByRole("combobox", { name: strings.workbench.preview.story }), {
      target: { value: "primary" },
    });
    expect(screen.getByTestId("preview-fixture")).toHaveAttribute("data-story", "primary");
    expect(screen.getByTestId("location")).toHaveTextContent("story=primary");
    expect(screen.getByTestId("location")).toHaveTextContent("device=phone");
  });

  it("copies the current preview URL and exposes the copied state", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", Object.assign(Object.create(null), { clipboard: { writeText } }));
    renderPreview("/assets/asset-7/preview?device=desktop&story=primary");

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.preview.share }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(window.location.href));
    expect(screen.getByRole("button", { name: strings.workbench.preview.copied })).toBeInTheDocument();
  });

  it("shows a specific empty state when the loaded asset has no stories", () => {
    vi.mocked(useComponentBundle).mockReturnValue({ data: { ...bundle, stories: [] }, isLoading: false, isError: false, error: null } as never);
    renderPreview();
    expect(screen.getByText(strings.workbench.preview.noStories)).toBeInTheDocument();
    expect(screen.queryByTestId("device-frame")).not.toBeInTheDocument();
  });

  it("distinguishes pending and failed bundle loads", () => {
    vi.mocked(useComponentBundle).mockReturnValue({ data: undefined, isLoading: true, isError: false, error: null } as never);
    const { unmount } = renderPreview();
    expect(document.querySelector(".wb-skeleton")).toBeInTheDocument();
    unmount();

    vi.mocked(useComponentBundle).mockReturnValue({ data: undefined, isLoading: false, isError: true, error: new Error("offline") } as never);
    renderPreview();
    expect(screen.getByRole("alert")).toHaveTextContent(strings.workbench.preview.error);
  });
});
