import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings.generated";
import { renderWithProviders } from "../../test-utils";
import type { ComponentBundle } from "../data";

const api = vi.hoisted(() => ({ runComponentTest: vi.fn(), getComponentTestReport: vi.fn() }));

vi.mock("../../api/componentTests", () => ({
  runComponentTest: api.runComponentTest,
  getComponentTestReport: api.getComponentTestReport,
}));

import { AssetDataDrawer } from "./AssetDataDrawer";

function bundle(reports: ComponentBundle["testReports"]): ComponentBundle {
  return {
    asset: {
      id: "asset-1",
      libraryId: "react-component-library:Button/2",
      catalogId: "controls.button",
      slug: "button",
      displayName: "Button",
      latestVersion: "2.1.0",
      version: "2.1.0",
      draftVersion: "",
      sourcePath: "components/Button/versions/2.1.0/Button.tsx",
      manifestPath: "components/Button/component.json",
      dependencies: [],
      tags: [],
      designStyles: [],
    },
    versions: [],
    stories: [],
    testReports: reports,
    partial: [],
  } as unknown as ComponentBundle;
}

beforeEach(async () => {
  await setLocale("en");
  api.runComponentTest.mockResolvedValue({ id: "report-new", verdict: "passed", results: [] });
  api.getComponentTestReport.mockResolvedValue(undefined);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AssetDataDrawer test evidence", () => {
  it("renders the stored report history with stage results", () => {
    const reports = [
      {
        id: "report-failed-2026",
        verdict: "failed",
        createdAt: "2026-11-20T12:00:00Z",
        results: [{ stage: "contract", subject: "Button@2.1.0", verdict: "failed" }],
      },
      {
        id: "report-passed-2026",
        verdict: "passed",
        createdAt: "2026-11-19T12:00:00Z",
        results: [{ stage: "preview", subject: "Button@2.1.0", verdict: "passed" }],
      },
    ] as unknown as ComponentBundle["testReports"];

    renderWithProviders(
      <AssetDataDrawer
        bundle={bundle(reports)}
        tab="tests"
        onTab={vi.fn()}
        version="2.1.0"
        onVersion={vi.fn()}
      />,
    );

    expect(screen.getByText(/report-faile/)).toBeInTheDocument();
    expect(screen.getByText(/report-faile/).nextSibling).toHaveTextContent("failed");
    expect(screen.getByText(/report-passe/)).toBeInTheDocument();
    expect(screen.getByText(/^contract$/)).toBeInTheDocument();
    expect(screen.getByText(/^preview$/)).toBeInTheDocument();
  });

  it("runs a focused test for the selected asset version", async () => {
    renderWithProviders(
      <AssetDataDrawer
        bundle={bundle([])}
        tab="tests"
        onTab={vi.fn()}
        version="2.1.0"
        onVersion={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Run test" }));

    await waitFor(() =>
      expect(api.runComponentTest).toHaveBeenCalledWith({
        componentId: "asset-1",
        version: "2.1.0",
        includeClosure: false,
      }),
    );
  });

  it("shows a useful error when a focused test cannot run", async () => {
    api.runComponentTest.mockRejectedValueOnce(new Error("test service unavailable"));

    renderWithProviders(
      <AssetDataDrawer
        bundle={bundle([])}
        tab="tests"
        onTab={vi.fn()}
        version="2.1.0"
        onVersion={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Run test" }));

    expect(await screen.findByText(/test service unavailable/)).toBeInTheDocument();
  });

  it("preserves an unavailable deep-linked report and shows its error state", async () => {
    api.getComponentTestReport.mockRejectedValueOnce(new Error("report missing"));

    renderWithProviders(
      <AssetDataDrawer
        bundle={bundle([])}
        tab="tests"
        onTab={vi.fn()}
        version="2.1.0"
        onVersion={vi.fn()}
        selectedTestReportID="report-unavailable"
      />,
    );

    expect(
      await screen.findByText(i18n.t(strings.workbench.component.reportUnavailable)),
    ).toBeInTheDocument();
    expect(api.getComponentTestReport).toHaveBeenCalledWith("report-unavailable");
  });

  it("loads a deep-linked report, shows its remediation, and reruns the pinned request", async () => {
    const expectedRemediation = "Add an accessible label to the button.";
    const report = {
      id: "report-deep-linked",
      rootLibraryId: "react-component-library:Button/2",
      rootVersion: "2.0.4",
      includeClosure: true,
      verdict: "failed",
      createdAt: "2026-11-20T12:00:00Z",
      results: [
        {
          stage: "contract",
          subject: "Button@2.0.4",
          verdict: "failed",
          message: "The accessible name is missing.",
          remediation: expectedRemediation,
        },
      ],
    } as unknown as ComponentBundle["testReports"][number];
    api.getComponentTestReport.mockResolvedValue(report);
    api.runComponentTest.mockResolvedValue({ id: "report-rerun", verdict: "failed", results: [] });
    const onSelectTestReport = vi.fn();

    renderWithProviders(
      <AssetDataDrawer
        bundle={bundle([])}
        tab="tests"
        onTab={vi.fn()}
        version="2.1.0"
        onVersion={vi.fn()}
        selectedTestReportID="report-deep-linked"
        onSelectTestReport={onSelectTestReport}
      />,
    );

    expect(await screen.findByText(expectedRemediation)).toBeInTheDocument();
    expect(api.getComponentTestReport).toHaveBeenCalledWith("report-deep-linked");
    fireEvent.click(screen.getByRole("button", { name: /report-deep-linked/ }));
    await waitFor(() =>
      expect(api.runComponentTest).toHaveBeenCalledWith({
        componentId: "react-component-library:Button/2",
        version: "2.0.4",
        includeClosure: true,
      }),
    );
    await waitFor(() => expect(onSelectTestReport).toHaveBeenCalledWith("report-rerun"));
  });

  it("lets a user select a report from the history for URL deep-linking", () => {
    const onSelectTestReport = vi.fn();
    const reports = [
      { id: "report-history", verdict: "passed", createdAt: "2026-11-20T12:00:00Z", results: [] },
    ] as unknown as ComponentBundle["testReports"];

    renderWithProviders(
      <AssetDataDrawer
        bundle={bundle(reports)}
        tab="tests"
        onTab={vi.fn()}
        version="2.1.0"
        onVersion={vi.fn()}
        onSelectTestReport={onSelectTestReport}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /^View report/ }));
    expect(onSelectTestReport).toHaveBeenCalledWith("report-history");
  });
});
