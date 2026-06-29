import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

vi.mock("../../api/health", () => ({
  fetchHealth: vi.fn(),
}));

vi.mock("../../api/healthStatus", () => ({
  getProviderHealth: vi.fn(),
}));

import { ServiceStatusPill } from "./ServiceStatusPill";
import { fetchHealth } from "../../api/health";
import { getProviderHealth } from "../../api/healthStatus";
import { State } from "@vrooli/proto-types/audio-tools/v1/health_status/health_status_pb";

const SERVICE_NAME = "audio-tools";

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

function makeHealthyProviders() {
  return {
    capabilities: [
      {
        providers: [{ state: State.AVAILABLE }],
      },
    ],
    cacheTtlSeconds: 30,
  };
}

function makeDegradedProviders(downCount: number) {
  return {
    capabilities: [
      {
        providers: Array.from({ length: downCount }, () => ({ state: State.UNAVAILABLE })),
      },
    ],
    cacheTtlSeconds: 30,
  };
}

function render() {
  return renderWithProviders(
    <MemoryRouter future={routerFuture}>
      <ServiceStatusPill />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.mocked(fetchHealth).mockResolvedValue({ status: "ok", service: "audio-tools" } as never);
  vi.mocked(getProviderHealth).mockResolvedValue(makeHealthyProviders() as never);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ServiceStatusPill", () => {
  it("shows checking state while loading", () => {
    // make the health call never resolve
    vi.mocked(fetchHealth).mockImplementation(() => new Promise(() => {}));
    render();
    expect(screen.getByText(strings.status.checking)).toBeInTheDocument();
  });

  it("shows offline when REST health fails", async () => {
    vi.mocked(fetchHealth).mockRejectedValue(new Error("network error"));
    render();
    await waitFor(() => {
      expect(screen.getByText(strings.status.offline)).toBeInTheDocument();
    });
  });

  it("shows healthy when REST and providers are all ok", async () => {
    render();
    await waitFor(() => {
      expect(screen.getByText(SERVICE_NAME)).toBeInTheDocument();
    });
  });

  it("falls back to summaryApi key when service name is empty", async () => {
    vi.mocked(fetchHealth).mockResolvedValue({ status: "ok", service: "" } as never);
    render();
    await waitFor(() => {
      expect(screen.getByText(strings.overview.summaryApi)).toBeInTheDocument();
    });
  });

  it("shows degraded when 1 provider is UNAVAILABLE", async () => {
    vi.mocked(getProviderHealth).mockResolvedValue(makeDegradedProviders(1) as never);
    render();
    await waitFor(() => {
      expect(screen.getByText(/Degraded.*1 down/)).toBeInTheDocument();
    });
  });

  it("shows degraded count for multiple UNAVAILABLE providers", async () => {
    vi.mocked(getProviderHealth).mockResolvedValue(makeDegradedProviders(3) as never);
    render();
    await waitFor(() => {
      expect(screen.getByText(/Degraded.*3 down/)).toBeInTheDocument();
    });
  });
});
