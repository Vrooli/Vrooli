/**
 * BackendsCard tests — the on-demand host-tool backend install affordance.
 *
 * Renders <BackendsCard /> directly and drives the doctorBackends → install
 * affordance → ensureBackend flow through the canonical mock-builder pattern.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  makeBackendStatus,
  makeDoctorBackendsResponse,
  makeEnsureBackendResponse,
} from "./mocks/factories";
import { makeModelsMocks } from "./mocks/models";

vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { BackendsCard } from "./BackendsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("BackendsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows an Install affordance only for a not-installed host-tool backend", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.doctorBackends).mockResolvedValueOnce(
      makeDoctorBackendsResponse({
        backends: [
          makeBackendStatus({ name: "realesrgan-ncnn-vulkan", hostTool: "realesrgan-ncnn-vulkan", hostToolReady: false }),
          makeBackendStatus({ name: "sd-ready", hostTool: "sd", hostToolReady: true }),
          // cloud/in-process provider: no host tool → not installable here.
          makeBackendStatus({ name: "builtin", hostTool: "", hostToolReady: true, remediation: "" }),
        ],
      }),
    );

    renderWithProviders(<BackendsCard />);
    await waitFor(() => expect(screen.getByTestId(selectors.backends.list)).toBeInTheDocument());

    // The no-host-tool builtin row is filtered out entirely: only the two
    // host-tool backends render, and exactly one (the not-ready realesrgan)
    // offers an install button.
    const names = screen.getAllByTestId(selectors.backends.name).map((n) => n.textContent);
    expect(names).toEqual(["realesrgan-ncnn-vulkan", "sd-ready"]);
    expect(screen.getAllByTestId(selectors.backends.installButton)).toHaveLength(1);
  });

  it("installs a not-ready backend via ensureBackend and surfaces the job notice", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.doctorBackends).mockResolvedValue(
      makeDoctorBackendsResponse({
        backends: [makeBackendStatus({ name: "realesrgan-ncnn-vulkan", hostTool: "realesrgan-ncnn-vulkan", hostToolReady: false })],
      }),
    );
    vi.mocked(modelsClient.ensureBackend).mockResolvedValueOnce(
      makeEnsureBackendResponse({ tool: "realesrgan-ncnn-vulkan", jobId: "ensure-7", etaSeconds: 90 }),
    );

    renderWithProviders(<BackendsCard />);
    await waitFor(() => expect(screen.getByTestId(selectors.backends.installButton)).toBeInTheDocument());

    await userEvent.click(screen.getByTestId(selectors.backends.installButton));

    await waitFor(() =>
      expect(modelsClient.ensureBackend).toHaveBeenCalledWith({ tool: "realesrgan-ncnn-vulkan" }),
    );
    await waitFor(() => {
      const notice = screen.getByTestId(selectors.backends.installNotice);
      expect(notice).toHaveTextContent("ensure-7");
    });
  });

  it("shows manual guidance (no job notice) when a backend cannot be auto-fetched", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.doctorBackends).mockResolvedValue(
      makeDoctorBackendsResponse({
        backends: [makeBackendStatus({ name: "iopaint", hostTool: "iopaint", hostToolReady: false })],
      }),
    );
    vi.mocked(modelsClient.ensureBackend).mockResolvedValueOnce(
      makeEnsureBackendResponse({
        tool: "iopaint",
        jobId: "",
        manual: true,
        state: "manual_action_required",
        detail: "pipx install iopaint",
      }),
    );

    renderWithProviders(<BackendsCard />);
    await waitFor(() => expect(screen.getByTestId(selectors.backends.installButton)).toBeInTheDocument());
    await userEvent.click(screen.getByTestId(selectors.backends.installButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.backends.manualHint)).toHaveTextContent("pipx install iopaint");
    });
    expect(screen.queryByTestId(selectors.backends.installNotice)).not.toBeInTheDocument();
  });

  it("renders the empty state when no host-tool backends are reported", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.doctorBackends).mockResolvedValueOnce(makeDoctorBackendsResponse({ backends: [] }));

    renderWithProviders(<BackendsCard />);
    await waitFor(() => expect(screen.getByTestId(selectors.backends.empty)).toBeInTheDocument());
  });
});
