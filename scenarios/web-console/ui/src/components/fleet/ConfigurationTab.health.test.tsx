import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import type { Machine } from "../../api/machines";
import { machineHealth, machineIssues } from "../machines/MachineList";
import { strings } from "../../consts/strings";

const apiMocks = vi.hoisted(() => ({
  getConfiguration: vi.fn(),
  listCredentialGrants: vi.fn(),
}));

vi.mock("../../api/machines", async () => {
  const actual = await vi.importActual<typeof import("../../api/machines")>("../../api/machines");
  return { ...actual, ...apiMocks };
});

import { ConfigurationTab } from "./ConfigurationTab";

/** minimouse's health as its agent reports it: 130 leftover onboarding folders, everything else fine. */
function machine(readiness: NonNullable<Machine["target"]["readiness"]>): Machine {
  return {
    target: { id: "bridge-node:minimouse", kind: "bridge-node", label: "minimouse", available: true, readiness },
    grant: { summary: "Read terminal", effects: ["read"], appCount: 1, coversAllApps: false, scopes: [], preset: "read" },
    heartbeatAgeSeconds: 4,
    manageable: true,
    drift: [],
  } as Machine;
}

const minimouse: NonNullable<Machine["target"]["readiness"]> = [
  { key: "capability:agy", label: "Antigravity", passed: true, state: "ready", detail: "" },
  { key: "node_health:disk", label: "Disk space", passed: true, state: "ready", detail: "170.0 GiB free of 233.0 GiB (73%)" },
  { key: "node_health:swap", label: "Swap", passed: false, state: "unknown", detail: "sysctl vm.swapusage could not be read" },
  {
    key: "node_health:bootstrap-artifacts",
    label: "Onboarding leftovers",
    passed: false,
    state: "missing",
    detail:
      "bootstrap_artifacts_large: 130 onboarding artifact folders use 6.6 GiB; only the newest folder is in use; the rest are left over from earlier onboarding runs and can be removed",
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  apiMocks.getConfiguration.mockResolvedValue({ questions: [], readiness: null, detail: null });
  apiMocks.listCredentialGrants.mockResolvedValue({ grants: [] });
});

describe("machine health", () => {
  it("counts a health warning toward the card's total but not as configuration drift", () => {
    const issues = machineIssues(machine(minimouse));
    expect(issues.healthWarnings.map((fact) => fact.key)).toEqual(["node_health:bootstrap-artifacts"]);
    expect(issues.count).toBe(1);
    expect(issues.configurationCount).toBe(0);
    expect(issues.unavailableNodeFeatures).toEqual([]);
  });

  it("lists problems first, then unmeasured readings, then healthy ones", () => {
    expect(machineHealth(machine(minimouse)).map((fact) => fact.key)).toEqual([
      "node_health:bootstrap-artifacts",
      "node_health:swap",
      "node_health:disk",
    ]);
  });

  it("shows each reading with its measurement and the remedy, without the internal code", async () => {
    render(<ConfigurationTab machine={machine(minimouse)} />);

    const section = await screen.findByTestId("machine-health");
    const leftovers = within(section).getByTestId("machine-health-bootstrap-artifacts");
    expect(leftovers).toHaveTextContent("Onboarding leftovers");
    expect(leftovers).toHaveTextContent("130 onboarding artifact folders use 6.6 GiB");
    expect(leftovers).toHaveTextContent("can be removed");
    // Test renders resolve i18n keys to themselves (see renderWithProviders.test.tsx).
    expect(leftovers).toHaveTextContent(strings.machines.healthWarning);
    expect(leftovers).not.toHaveTextContent("bootstrap_artifacts_large");
    expect(within(section).getByTestId("machine-health-disk")).toHaveTextContent(strings.machines.healthOk);
    expect(within(section).getByTestId("machine-health-swap")).toHaveTextContent(strings.machines.healthUnknown);
    expect(screen.queryByTestId("machine-configuration-drift")).toBeNull();
  });

  it("says plainly when a machine's agent is too old to report health", async () => {
    render(<ConfigurationTab machine={machine([{ key: "capability:agy", label: "Antigravity", passed: true, state: "ready", detail: "" }])} />);

    expect(await screen.findByTestId("machine-health-not-reported")).toHaveTextContent(strings.machines.healthNotReported);
  });
});
