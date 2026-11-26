import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { webcrypto } from "node:crypto";

import { ObservedBehaviorPanel } from "./ObservedBehaviorPanel";

function canonical(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "number") return Number.isFinite(value) ? String(value) : "null";
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "string") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map((item) => canonical(item ?? null)).join(",")}]`;
  if (typeof value === "object") return `{${Object.entries(value as Record<string, unknown>).filter(([, item]) => item !== undefined).sort(([left], [right]) => left.localeCompare(right)).map(([key, item]) => `${JSON.stringify(key)}:${canonical(item)}`).join(",")}}`;
  return "null";
}

async function fixtureFiles(build = "sha256:build-a") {
  const session = { id: "session-1", build, startedAt: "2026-11-24T00:00:00.000Z", endedAt: "2026-11-24T00:01:00.000Z" };
  const report: Record<string, unknown> = {
    schemaVersion: "experience-friction-report/v1", scenario: "demo", generatedAt: "2026-11-24T00:01:00.000Z", builds: [build], sessionRefs: [session],
    totals: { sessions: 1, activeMs: 1000 }, recommendations: [{ id: "1", title: "Repeated control", evidence: "The control repeated.", suggestion: "Review the interaction.", severity: "medium" }], window: { from: session.startedAt, to: session.endedAt, sessions: 1 }, flows: [], friction: [{ kind: "repeated_action", count: 1 }],
  };
  const digest = `sha256:${Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(canonical(report)))), (part) => part.toString(16).padStart(2, "0")).join("")}`;
  report.reportDigest = digest;
  const envelope = { kind: "experience-observed-behavior", schemaVersion: "experience-observed-behavior/v1", state: "fresh", scenario: "demo", build, builds: [build], expectedBuild: build, sessions: [session], window: report.window, reportDigest: digest };
  const findings = { schemaVersion: "experience-advisory-findings/v1", findings: [{ schemaVersion: "experience-advisory-finding/v1", id: "adv-1", severity: "medium", kind: "repeated_action", title: "Repeated control", evidence: "The control repeated.", suggestion: "Review the interaction.", route: "/", element: { testId: "save" }, reportDigest: digest, advisory: true }] };
  const candidates = { schemaVersion: "experience-correction-candidates/v1", candidates: [{ schemaVersion: "experience-correction-candidate/v1", id: "cand-1", kind: "repeated_action", severity: "medium", route: "/", element: { testId: "save" }, evidence: "Repeated action on save.", reportDigest: digest, status: "candidate", source: "observed", advisory: true }] };
  const packet = { schemaVersion: "experience-packet-observed/v1", scenario: "demo", generatedAt: "2026-11-24T00:01:00.000Z", build, sessions: [{ id: session.id, build }], window: report.window, headline: "save repeated", topFlow: null, topFriction: { kind: "repeated_action", count: 1 }, timeCostMs: 1000, reportDigest: digest, caveat: "Advisory only." };
  return { report: JSON.stringify(report), envelope: JSON.stringify(envelope), findings: JSON.stringify(findings), candidates: JSON.stringify(candidates), packet: JSON.stringify(packet), build };
}

async function bundleWithEnvelopeChange(change: (envelope: Record<string, unknown>) => void) {
  const files = await fixtureFiles();
  const envelope = JSON.parse(files.envelope) as Record<string, unknown>;
  change(envelope);
  return { ...files, envelope: JSON.stringify(envelope) };
}

async function selectBundle(files: Awaited<ReturnType<typeof fixtureFiles>>) {
  await userEvent.type(screen.getByLabelText("Expected build identity"), files.build);
  for (const [label, key] of [["Friction report file", "report"], ["Observed behavior file", "envelope"], ["Advisory findings file", "findings"], ["Correction candidates file", "candidates"], ["Observed packet file", "packet"]] as const) {
    const input = screen.getByLabelText(label) as HTMLInputElement;
    const file = Object.assign(new File([files[key]], `${key}.json`, { type: "application/json" }), { text: async () => files[key] });
    fireEvent.change(input, { target: { files: [file] } });
  }
  await userEvent.click(screen.getByRole("button", { name: "Evaluate observed evidence" }));
}

describe("ObservedBehaviorPanel", () => {
  beforeAll(() => {
    if (!globalThis.crypto?.subtle) Object.defineProperty(globalThis, "crypto", { value: webcrypto });
  });

  it("renders digest-bound findings as advisory with optional packet context", async () => {
    const files = await fixtureFiles();
    render(<ObservedBehaviorPanel scenario="demo" />);
    await selectBundle(files);
    expect(await screen.findByRole("region", { name: "Evaluated advisory evidence" })).toBeInTheDocument();
    expect(screen.getByText("Repeated control")).toBeInTheDocument();
    expect(screen.getByText("medium · advisory only")).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Optional observed packet block" })).toBeInTheDocument();
    expect(screen.getByText(/Not a verdict/)).toBeInTheDocument();
    expect(screen.getByText("Correction candidates · review required")).toBeInTheDocument();
    expect(screen.getByText("Repeated action on save.")).toBeInTheDocument();
  });

  it("fails closed on a missing build identity", async () => {
    const files = await fixtureFiles();
    render(<ObservedBehaviorPanel scenario="demo" />);
    await selectBundle({ ...files, build: "sha256:other" });
    expect(await screen.findByRole("status")).toHaveTextContent("Not evaluated");
    expect(screen.queryByRole("region", { name: "Evaluated advisory evidence" })).not.toBeInTheDocument();
  });

  it("fails closed when the report scenario differs", async () => {
    const files = await fixtureFiles();
    render(<ObservedBehaviorPanel scenario="other-scenario" />);
    await selectBundle(files);
    expect(await screen.findByRole("status")).toHaveTextContent("scenario does not match");
  });

  it("fails closed when the session reference differs from the report", async () => {
    const files = await bundleWithEnvelopeChange((envelope) => {
      (envelope.sessions as Array<Record<string, unknown>>)[0]!.id = "other-session";
    });
    render(<ObservedBehaviorPanel scenario="demo" />);
    await selectBundle(files);
    expect(await screen.findByRole("status")).toHaveTextContent("Session identity");
  });

  it("fails closed when the report digest has been altered", async () => {
    const files = await fixtureFiles();
    const report = JSON.parse(files.report) as Record<string, unknown>;
    report.scenario = "tampered";
    render(<ObservedBehaviorPanel scenario="demo" />);
    await selectBundle({ ...files, report: JSON.stringify(report) });
    expect(await screen.findByRole("status")).toHaveTextContent("digest does not match");
  });

  it("exports an explicit review disposition without a corpus write", async () => {
    const files = await fixtureFiles();
    Object.defineProperty(URL, "createObjectURL", { configurable: true, value: vi.fn(() => "blob:review") });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: vi.fn() });
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    render(<ObservedBehaviorPanel scenario="demo" />);
    await selectBundle(files);
    await userEvent.selectOptions(await screen.findByLabelText("Review decision cand-1"), "decline");
    await userEvent.type(screen.getByLabelText("Review reason cand-1"), "Repeated action alone does not establish an interface defect.");
    const exportButton = screen.getByRole("button", { name: "Download review disposition (no corpus write)" });
    expect(exportButton).toBeEnabled();
    await userEvent.click(exportButton);
    expect(URL.createObjectURL).toHaveBeenCalledOnce();
    expect(click).toHaveBeenCalledOnce();
  });
});
