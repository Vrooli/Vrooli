import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  browserVisibleArtifactUrl,
  ComponentTestPanel,
  readBoundedEvidenceText,
  summarizePerformance,
  tone,
  verdictLabel,
} from "./ComponentTestPanel";
import { renderWithProviders } from "../../test-utils";
import type { ComponentExperience } from "../../api/components";

const api = vi.hoisted(() => ({
  getComponentTestReport: vi.fn(),
  listComponentTestReports: vi.fn(),
  runComponentTest: vi.fn(),
}));
vi.mock("../../api/componentTests", () => api);

describe("ComponentTestPanel", () => {
  it("keeps evidence classification and artifact URL helpers deterministic", () => {
    expect(tone("passed")).toBe("success");
    expect(tone("failed")).toBe("danger");
    expect(tone("blocked")).toBe("warning");
    expect(tone("unmeasured")).toBe("warning");
    expect(tone("unknown")).toBe("neutral");
    expect(verdictLabel("passed")).toBe("Passed");
    expect(verdictLabel("failed")).toBe("Needs attention");
    expect(verdictLabel("blocked")).toBe("Blocked");
    expect(verdictLabel("unmeasured")).toBe("Unmeasured");
    expect(verdictLabel("unknown")).toBe("Inconclusive");
    expect(browserVisibleArtifactUrl("/embedded/browser-automation-studio/api/v1/artifacts/a"))
      .toContain("/embedded/browser-automation-studio/api/v1/artifacts/a");
    expect(browserVisibleArtifactUrl("http://127.0.0.1:17116/api/v1/artifacts/a.png"))
      .toContain("/embedded/browser-automation-studio/api/v1/artifacts/a.png");
    expect(browserVisibleArtifactUrl("provenance:capture")).toBe("provenance:capture");
    expect(summarizePerformance(null).summary.value).toBeNull();
    expect(summarizePerformance([]).summary.value).toEqual([]);
  });

  it("bounds streamed evidence before a large performance trace can freeze the workspace", async () => {
    const response = new Response(
      new ReadableStream({
        start(controller) {
          controller.enqueue(new TextEncoder().encode("1234"));
          controller.enqueue(new TextEncoder().encode("5678"));
          controller.enqueue(new TextEncoder().encode("90"));
          controller.close();
        },
      }),
    );

    await expect(readBoundedEvidenceText(response, "Performance", 8)).rejects.toThrow(
      "Performance is too large to render safely",
    );
  });

  it("defaults to the selected version and renders durable remediation", async () => {
    api.listComponentTestReports.mockResolvedValue([]);
    api.runComponentTest.mockResolvedValue({
      id: "ctr_123",
      verdict: "failed",
      results: [
        {
          stage: "contract_validation",
          assetLibraryId: "rcl:Button",
          version: "1.0.0",
          verdict: "failed",
          message: "invalid",
          remediation: "fix contract",
        },
      ],
    });
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    await screen.findByText("No component test evidence yet");
    expect(screen.getByRole("checkbox", { name: "Include dependency closure" })).not.toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Run component tests" }));
    await waitFor(() =>
      expect(api.runComponentTest).toHaveBeenCalledWith({
        componentId: "button-id",
        version: "1.0.0",
        includeClosure: false,
      }),
    );
    expect(await screen.findAllByText("Recommended next step:")).toHaveLength(2);
    expect(screen.getAllByText("fix contract")).toHaveLength(2);
    expect(screen.getAllByText("Needs attention")).toHaveLength(3);
  });

  it("provides a deep link for historical durable evidence", async () => {
    api.listComponentTestReports.mockResolvedValue([
      { id: "ctr_latest", verdict: "passed", results: [] },
      { id: "ctr_history", verdict: "failed", results: [] },
    ]);
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />, {
      routerEntries: ["/assets/button-id?tab=tests"],
    });
    expect(
      await screen.findByRole("link", { name: "Open component test report ctr_history" }),
    ).toHaveAttribute("href", "/assets/button-id?tab=tests&testReport=ctr_history");
  });

  it("shows a structural loading skeleton while history is being retrieved", () => {
    api.listComponentTestReports.mockReturnValue(new Promise(() => undefined));
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    expect(screen.getByTestId("component-test-history-skeleton")).toBeInTheDocument();
  });

  it("explains when history cannot be retrieved instead of presenting an empty history", async () => {
    api.listComponentTestReports.mockRejectedValue(new Error("legacy data could not be decoded"));
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Test history could not be loaded");
    expect(screen.queryByText("No component test evidence yet")).not.toBeInTheDocument();
  });

  it("surfaces BAS artifacts as readable evidence states", async () => {
    api.listComponentTestReports.mockResolvedValue([
      {
        id: "ctr_capture",
        verdict: "passed",
        results: [],
        artifacts: [
          {
            kind: "bas-screenshot",
            label: "story:screenshot",
            assetLibraryId: "rcl:Button",
            version: "1.0.0",
            reference: "http://127.0.0.1:17116/api/v1/artifacts/capture.png",
          },
        ],
      },
    ]);
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    expect(await screen.findByRole("tab", { name: "Screenshot: Captured" })).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: "Accessibility Tree: Not captured" }),
    ).toBeInTheDocument();
    expect(screen.queryByText("screenshotmissing")).not.toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: "Captured component screenshot" }).getAttribute("src"),
    ).toContain("/embedded/browser-automation-studio/api/v1/artifacts/capture.png");
    expect(screen.queryByRole("link", { name: /open.*capture/i })).not.toBeInTheDocument();
  });

  it("explains when the embedded screenshot artifact cannot be loaded", async () => {
    api.listComponentTestReports.mockResolvedValue([
      {
        id: "ctr_capture_error",
        verdict: "passed",
        results: [],
        artifacts: [
          {
            kind: "bas-screenshot",
            label: "story:screenshot",
            assetLibraryId: "rcl:Button",
            version: "1.0.0",
            reference: "/embedded/browser-automation-studio/api/v1/artifacts/missing.png",
          },
        ],
      },
    ]);
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    const image = await screen.findByRole("img", { name: "Captured component screenshot" });
    fireEvent.error(image);
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Screenshot could not be displayed",
    );
  });

  it("reviews measured claims and structured evidence without rendering the raw trace", async () => {
    const fetchMock = vi.fn().mockImplementation(
      () =>
        Promise.resolve(
          new Response(
        JSON.stringify({
          traceEvents: [
            { name: "paint", ts: 1000, dur: 2000 },
            { name: "paint", ts: 4000, dur: 1000 },
            {},
            null,
          ],
          metadata: { lcp: 120 },
        }),
        { headers: { "content-type": "application/json" } },
          ),
        ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const experience = {
      claims: [
        { id: "claim-1", statement: "The result stays readable", type: "visual", tier: "machine", states: [] },
        { id: "claim-2", statement: "The fallback is explicit", type: "behavior", tier: "manual", states: [] },
      ],
      evidence: [
        {
          claimId: "claim-1",
          verdict: "failed",
          stateId: "default",
          exampleName: "Default result",
          captureRef: "capture-1",
          checkedAt: "2026-08-24T00:00:00Z",
          message: "text clipped",
          viewport: "desktop",
          viewportWidth: 1440,
          viewportHeight: 900,
          measurement: {
            observed: "12",
            required: "16",
            unit: "px",
            metric: "font-size",
            subjects: [{ testId: "result", value: "12px", bounds: { x: 4, y: 8, width: 100, height: 20 } }],
          },
        },
        {
          claimId: "claim-2",
          verdict: "failed",
          stateId: "error",
          exampleName: "Error result",
          captureRef: "capture-2",
          checkedAt: "2026-08-24T00:00:00Z",
          message: "no measurement",
          viewport: "mobile",
          viewportWidth: 390,
          viewportHeight: 844,
        },
      ],
    } as unknown as ComponentExperience;
    api.listComponentTestReports.mockResolvedValue([
      {
        id: "ctr_measured",
        rootLibraryId: "rcl:Button",
        rootVersion: "2.2.0",
        includeClosure: false,
        verdict: "failed",
        results: [
          { stage: "contract", assetLibraryId: "rcl:Button", version: "2.2.0", subject: "", verdict: "passed", message: "", remediation: "" },
          { stage: "behavior", assetLibraryId: "rcl:Button", version: "2.2.0", subject: "", verdict: "passed", message: "", remediation: "" },
          { stage: "experience", assetLibraryId: "rcl:Button", version: "2.2.0", subject: "", verdict: "passed", message: "", remediation: "" },
          { stage: "performance", assetLibraryId: "rcl:Button", version: "2.2.0", subject: "", verdict: "passed", message: "", remediation: "" },
        ],
        artifacts: [
          "bas-screenshot",
          "bas-accessibility",
          "bas-console_logs",
          "bas-performance",
        ].map((kind) => ({
          kind,
          label: kind,
          assetLibraryId: "rcl:Button",
          version: "2.2.0",
          reference: `/embedded/browser-automation-studio/api/v1/artifacts/${kind}.json`,
        })),
      },
    ]);

    renderWithProviders(
      <ComponentTestPanel componentId="button-id" version="2.2.0" experience={experience} />,
    );

    expect(await screen.findAllByText("The result stays readable")).toHaveLength(2);
    expect(screen.getByText("12")).toBeInTheDocument();
    expect(await screen.findByRole("tab", { name: "Screenshot: Captured" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "Show measured overlay" }));
    expect(screen.getByRole("checkbox", { name: "Show measured overlay" })).not.toBeChecked();
    fireEvent.click(screen.getByRole("tab", { name: "Performance: Captured" }));
    expect(await screen.findByText("Trace events")).toBeInTheDocument();
    expect(screen.getByText("Captured duration")).toBeInTheDocument();
    expect(screen.getByText("4 ms")).toBeInTheDocument();
    expect(screen.getByText("paint")).toBeInTheDocument();
    expect(screen.getByText("unnamed")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalled();

    fireEvent.click(screen.getByRole("tab", { name: "Accessibility Tree: Captured" }));
    await waitFor(() =>
      expect(screen.queryByText(/trace events are summarized/i)).not.toBeInTheDocument(),
    );
    expect(await screen.findByText(/traceEvents/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: /claim-2/ }));
    expect(screen.getByTestId("claim-overlay-unmeasured")).toBeInTheDocument();
  });

  it("covers bounded evidence responses without a stream and rejects oversized headers", async () => {
    const noBody = new Response("plain text");
    Object.defineProperty(noBody, "body", { value: null });
    await expect(readBoundedEvidenceText(noBody, "Console")).resolves.toBe("plain text");

    const oversized = new Response("ok", { headers: { "content-length": "99" } });
    await expect(readBoundedEvidenceText(oversized, "Console", 8)).rejects.toThrow(
      "Console is too large to render safely",
    );
  });

  it("renders an explicit empty capture workspace and reports run failures", async () => {
    api.listComponentTestReports.mockResolvedValue([
      {
        id: "ctr_empty",
        verdict: "unmeasured",
        results: [
          {
            stage: "contract",
            assetLibraryId: "rcl:Button",
            version: "1.0.0",
            subject: "",
            verdict: "blocked",
            message: "waiting for contract evidence",
            remediation: "publish the contract",
          },
          {
            stage: "behavior",
            assetLibraryId: "rcl:Button",
            version: "1.0.0",
            subject: "",
            verdict: "unmeasured",
            message: "not measured",
            remediation: "run the behavior checks",
          },
          {
            stage: "performance",
            assetLibraryId: "rcl:Button",
            version: "1.0.0",
            subject: "",
            verdict: "inconclusive",
            message: "no timing sample",
            remediation: "capture a trace",
          },
        ],
        artifacts: [],
      },
    ]);
    api.runComponentTest.mockRejectedValue(new Error("component contract is unavailable"));
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);

    expect(await screen.findByText("No BAS capture bundle in this report.")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Screenshot: Not captured" })).toBeInTheDocument();
    expect(screen.getByText("Screenshot is not captured")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Run component tests" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "component contract is unavailable",
    );
  });

  it("keeps selected-report failures and malformed structured artifacts legible", async () => {
    api.listComponentTestReports.mockResolvedValue([]);
    api.getComponentTestReport.mockRejectedValue(new Error("report was deleted"));
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />, {
      routerEntries: ["/assets/button-id?tab=tests&testReport=missing"],
    });
    expect(await screen.findByRole("alert")).toHaveTextContent("requested report is unavailable");

    cleanup();
    api.getComponentTestReport.mockResolvedValue({
      id: "ctr-text",
      verdict: "passed",
      results: [],
      artifacts: [
        {
          kind: "bas-console_logs",
          label: "Console",
          assetLibraryId: "rcl:Button",
          version: "1.0.0",
          reference: "/embedded/browser-automation-studio/api/v1/artifacts/console.txt",
        },
      ],
    });
    const fetchMock = vi.fn().mockResolvedValue(new Response("not-json"));
    vi.stubGlobal("fetch", fetchMock);
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />, {
      routerEntries: ["/assets/button-id?tab=tests&testReport=text"],
    });
    fireEvent.click(await screen.findByRole("tab", { name: "Console: Captured" }));
    expect(await screen.findByText("not-json")).toBeInTheDocument();
  });

  it("shows the bounded empty state for a captured performance trace", async () => {
    api.listComponentTestReports.mockResolvedValue([
      {
        id: "ctr-performance",
        verdict: "passed",
        results: [],
        artifacts: [
          {
            kind: "bas-performance",
            label: "Performance",
            assetLibraryId: "rcl:Button",
            version: "1.0.0",
            reference: "/embedded/browser-automation-studio/api/v1/artifacts/performance.json",
          },
        ],
      },
    ]);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ metadata: {} }))));
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    fireEvent.click(await screen.findByRole("tab", { name: "Performance: Captured" }));
    expect(await screen.findByText("No trace events were recorded.")).toBeInTheDocument();
    expect(screen.getByText("Captured duration")).toBeInTheDocument();
  });
});
