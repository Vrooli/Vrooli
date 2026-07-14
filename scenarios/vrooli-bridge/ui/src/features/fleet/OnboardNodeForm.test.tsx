import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { OnboardingState, OnboardingStepStatus } from "../../api/onboard";
import { makeGetOnboardingResponse, makeStepEvent } from "./mocks/factories";

const { startOnboarding, getOnboarding } = vi.hoisted(() => ({
  startOnboarding: vi.fn(),
  getOnboarding: vi.fn(),
}));

vi.mock("../../api/onboard", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/onboard")>();
  return { ...actual, onboardClient: { startOnboarding, getOnboarding } };
});

import { OnboardNodeForm } from "./OnboardNodeForm";

// Every terminal-failure code the orchestrator can record (mirrors
// api/internal/onboard/types.go); each must render its own distinct message.
const FAILURE_CODES = [
  "ssh_setup_failed",
  "script_push_failed",
  "pairing_issue_failed",
  "bootstrap_usage_error",
  "unsupported_platform",
  "pairing_failed",
  "bootstrap_failed",
  "verify_online_failed",
  "interrupted_by_restart",
  "internal_error",
] as const;

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

async function submitHost(host = "node-01.example.com") {
  const user = userEvent.setup();
  await user.type(screen.getByTestId(selectors.fleet.onboard.host), host);
  await user.click(screen.getByTestId(selectors.fleet.onboard.submit));
  return user;
}

describe("OnboardNodeForm", () => {
  it("renders the connection fields with revision defaulting to @cp and a masked password", () => {
    renderWithProviders(<OnboardNodeForm />);
    expect(screen.getByTestId(selectors.fleet.onboard.host)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.fleet.onboard.user)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.fleet.onboard.revision)).toHaveValue("@cp");
    expect(screen.getByTestId(selectors.fleet.onboard.password)).toHaveAttribute("type", "password");
  });

  it("disables submit until a host is entered", async () => {
    renderWithProviders(<OnboardNodeForm />);
    expect(screen.getByTestId(selectors.fleet.onboard.submit)).toBeDisabled();
    await userEvent.setup().type(screen.getByTestId(selectors.fleet.onboard.host), "h");
    expect(screen.getByTestId(selectors.fleet.onboard.submit)).toBeEnabled();
  });

  it("sends the password in the request body, then clears it and never persists it", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-77" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-77", state: OnboardingState.BOOTSTRAPPING }, [makeStepEvent()]),
    );

    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);
    await user.type(screen.getByTestId(selectors.fleet.onboard.host), "node-01.example.com");
    await user.type(screen.getByTestId(selectors.fleet.onboard.capabilities), "scenario, deploy");
    const passwordField = screen.getByTestId(selectors.fleet.onboard.password);
    await user.type(passwordField, "s3cret-pw");
    await user.click(screen.getByTestId(selectors.fleet.onboard.submit));

    await waitFor(() => expect(startOnboarding).toHaveBeenCalledTimes(1));
    const input = startOnboarding.mock.calls[0]?.[0];
    expect(input?.sshPassword).toBe("s3cret-pw");
    expect(input?.host).toBe("node-01.example.com");
    expect(input?.capabilities).toEqual(["scenario", "deploy"]);
    expect(input?.targetRevision).toBe("@cp");

    // The secret is wiped from the field the moment the request settles...
    await waitFor(() => expect(passwordField).toHaveValue(""));
    // ...and never lands in any browser storage.
    expect(JSON.stringify(window.localStorage)).not.toContain("s3cret-pw");
    expect(JSON.stringify(window.sessionStorage)).not.toContain("s3cret-pw");
  });

  it("renders live step states from the op's event history", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-live" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-live", state: OnboardingState.BOOTSTRAPPING }, [
        makeStepEvent({ sequence: 1n, stepId: "ssh-setup", status: OnboardingStepStatus.OK }),
        makeStepEvent({ sequence: 2n, stepId: "clone", status: OnboardingStepStatus.STARTED, detail: "git clone" }),
      ]),
    );

    renderWithProviders(<OnboardNodeForm />);
    await submitHost();

    expect(await screen.findByTestId(selectors.fleet.onboard.progress)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.fleet.onboardStep({ step: "ssh-setup" }))).toBeInTheDocument();
    const cloneRow = screen.getByTestId(selectors.fleet.onboardStep({ step: "clone" }));
    expect(cloneRow).toHaveTextContent("git clone");
  });

  it("renders a terminal success banner", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-ok" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse(
        { id: "op-ok", state: OnboardingState.SUCCEEDED, nodeId: "node-9", nodeName: "mac-mini" },
        [makeStepEvent({ stepId: "verify-online-confirm", status: OnboardingStepStatus.OK })],
      ),
    );

    renderWithProviders(<OnboardNodeForm />);
    await submitHost();

    const banner = await screen.findByTestId(selectors.fleet.onboard.success);
    expect(banner).toHaveTextContent(strings.fleet.onboard.successHeading);
  });

  it("renders a distinct, actionable message for every failure taxonomy code", async () => {
    const rendered = new Set<string>();
    for (const code of FAILURE_CODES) {
      startOnboarding.mockResolvedValue({ opId: `op-${code}` });
      getOnboarding.mockResolvedValue(
        makeGetOnboardingResponse(
          { id: `op-${code}`, state: OnboardingState.FAILED, failureReason: code, exitCode: 1 },
          [makeStepEvent({ stepId: "pair-redeem", status: OnboardingStepStatus.FAILED })],
        ),
      );

      renderWithProviders(<OnboardNodeForm />);
      await submitHost();

      const banner = await screen.findByTestId(selectors.fleet.onboard.failure);
      const expectedKey = strings.fleet.onboard.failure[code];
      expect(banner, `code ${code} should render its own message`).toHaveTextContent(expectedKey);
      // Under cimode the rendered text is the i18n key path — distinct per code.
      expect(rendered.has(expectedKey), `code ${code} shares a message`).toBe(false);
      rendered.add(expectedKey);

      cleanup();
      vi.clearAllMocks();
    }
    expect(rendered.size).toBe(FAILURE_CODES.length);
  });

  it("surfaces a StartOnboarding error", async () => {
    startOnboarding.mockRejectedValue(new ConnectError("host unreachable", Code.InvalidArgument));

    renderWithProviders(<OnboardNodeForm />);
    await submitHost();

    expect(await screen.findByTestId(selectors.fleet.onboard.error)).toBeInTheDocument();
  });
});
