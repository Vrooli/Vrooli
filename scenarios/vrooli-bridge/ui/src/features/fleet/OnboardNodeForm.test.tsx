import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { OnboardingState, OnboardingStepStatus, SourceMode } from "../../api/onboard";
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

const s = selectors.fleet.onboard;

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

// Wizard navigation helpers. Step 1 Connect (host/user) → Step 2 Unlock
// (password/sudo) → Step 3 Review (Start).
async function fillHostToUnlock(user: ReturnType<typeof userEvent.setup>, host = "node-01.example.com") {
  await user.type(screen.getByTestId(s.host), host);
  await user.click(screen.getByTestId(s.next)); // Connect → Unlock
}
async function toReview(user: ReturnType<typeof userEvent.setup>, host = "node-01.example.com") {
  await fillHostToUnlock(user, host);
  await user.click(screen.getByTestId(s.next)); // Unlock → Review
}
async function openAdvanced(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId(s.advancedToggle));
}
/** Drive the whole wizard to the Review step and press Start. */
async function startOnboard(host = "node-01.example.com") {
  const user = userEvent.setup();
  await toReview(user, host);
  await user.click(screen.getByTestId(s.submit));
  return user;
}

describe("OnboardNodeForm wizard", () => {
  it("walks Connect → Unlock → Review, surfacing the fields for each step", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);

    // Step 1 Connect: address + username, no password yet.
    expect(screen.getByTestId(s.host)).toBeInTheDocument();
    expect(screen.getByTestId(s.user)).toBeInTheDocument();
    expect(screen.queryByTestId(s.password)).not.toBeInTheDocument();

    await fillHostToUnlock(user);

    // Step 2 Unlock: masked password + the (checked-by-default) admin toggle.
    expect(screen.getByTestId(s.password)).toHaveAttribute("type", "password");
    expect(screen.getByTestId(s.provisionSudo)).toBeChecked();

    await user.click(screen.getByTestId(s.next));

    // Step 3 Review: advanced options collapsed until asked for; revision @cp.
    expect(screen.queryByTestId(s.revision)).not.toBeInTheDocument();
    await openAdvanced(user);
    expect(screen.getByTestId(s.revision)).toHaveValue("@cp");
  });

  it("gates Next on a non-empty address and lets Back return to it", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);

    expect(screen.getByTestId(s.next)).toBeDisabled();
    await user.type(screen.getByTestId(s.host), "node-01.example.com");
    expect(screen.getByTestId(s.next)).toBeEnabled();

    await user.click(screen.getByTestId(s.next));
    expect(screen.getByTestId(s.password)).toBeInTheDocument();
    await user.click(screen.getByTestId(s.back));
    // Back returns to Connect with the address preserved.
    expect(screen.getByTestId(s.host)).toHaveValue("node-01.example.com");
  });

  it("preserves values entered on earlier steps across navigation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);

    await user.type(screen.getByTestId(s.host), "node-01.example.com");
    await user.click(screen.getByTestId(s.next));
    await user.type(screen.getByTestId(s.password), "s3cret-pw");
    await user.click(screen.getByTestId(s.back));
    await user.click(screen.getByTestId(s.next));
    // The password typed on Unlock survives a round-trip to Connect and back.
    expect(screen.getByTestId(s.password)).toHaveValue("s3cret-pw");
  });

  it("moves focus to the step heading when advancing", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);
    await fillHostToUnlock(user);
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByRole("heading", { level: 3 })),
    );
  });
});

describe("OnboardNodeForm submission", () => {
  it("sends the password in the request body and never persists it", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-77" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-77", state: OnboardingState.BOOTSTRAPPING }, [makeStepEvent()]),
    );

    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);
    await user.type(screen.getByTestId(s.host), "node-01.example.com");
    await user.click(screen.getByTestId(s.next));
    await user.type(screen.getByTestId(s.password), "s3cret-pw");
    await user.click(screen.getByTestId(s.next));
    await openAdvanced(user);
    await user.type(screen.getByTestId(s.capabilities), "scenario, deploy");
    await user.click(screen.getByTestId(s.submit));

    await waitFor(() => expect(startOnboarding).toHaveBeenCalledTimes(1));
    const input = startOnboarding.mock.calls[0]?.[0];
    expect(input?.sshPassword).toBe("s3cret-pw");
    expect(input?.host).toBe("node-01.example.com");
    expect(input?.capabilities).toEqual(["scenario", "deploy"]);
    expect(input?.targetRevision).toBe("@cp");

    // Starting swaps the wizard for the live progress view...
    expect(await screen.findByTestId(s.progress)).toBeInTheDocument();
    // ...and the secret never lands in any browser storage.
    expect(JSON.stringify(window.localStorage)).not.toContain("s3cret-pw");
    expect(JSON.stringify(window.sessionStorage)).not.toContain("s3cret-pw");
  });

  it("reveals and re-masks the password via the visibility toggle", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);
    await fillHostToUnlock(user);
    const passwordField = screen.getByTestId(s.password);
    const toggle = screen.getByTestId(s.passwordToggle);

    await user.type(passwordField, "s3cret-pw");
    expect(passwordField).toHaveAttribute("type", "password");
    expect(toggle).toHaveAttribute("aria-pressed", "false");

    await user.click(toggle);
    expect(passwordField).toHaveAttribute("type", "text");
    expect(toggle).toHaveAttribute("aria-pressed", "true");
    expect(passwordField).toHaveValue("s3cret-pw");

    await user.click(toggle);
    expect(passwordField).toHaveAttribute("type", "password");
  });

  it("provisions passwordless sudo by default and lets the operator opt out", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-sudo" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-sudo", state: OnboardingState.BOOTSTRAPPING }, [makeStepEvent()]),
    );

    // Default: the toggle is on and the request carries provisionSudo=true.
    renderWithProviders(<OnboardNodeForm />);
    await startOnboard();
    await waitFor(() => expect(startOnboarding).toHaveBeenCalledTimes(1));
    expect(startOnboarding.mock.calls[0]?.[0]?.provisionSudo).toBe(true);

    cleanup();
    vi.clearAllMocks();
    startOnboarding.mockResolvedValue({ opId: "op-sudo-off" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-sudo-off", state: OnboardingState.BOOTSTRAPPING }, [makeStepEvent()]),
    );

    // Opt out: unchecking the toggle on Unlock sends provisionSudo=false.
    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);
    await fillHostToUnlock(user);
    await user.click(screen.getByTestId(s.provisionSudo));
    expect(screen.getByTestId(s.provisionSudo)).not.toBeChecked();
    await user.click(screen.getByTestId(s.next));
    await user.click(screen.getByTestId(s.submit));
    await waitFor(() => expect(startOnboarding).toHaveBeenCalledTimes(1));
    expect(startOnboarding.mock.calls[0]?.[0]?.provisionSudo).toBe(false);
  });

  it("threads the advanced setup profile into the StartOnboarding request", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-prof" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-prof", state: OnboardingState.BOOTSTRAPPING }, [makeStepEvent()]),
    );

    const user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);
    await toReview(user);
    await openAdvanced(user);
    await user.type(screen.getByTestId(s.setupEnvironment), "production");
    await user.type(screen.getByTestId(s.setupResources), "enabled");
    await user.type(screen.getByTestId(s.setupScenarios), "none");
    await user.type(screen.getByTestId(s.controlPlaneUrl), "http://control-plane.example.com:8080");
    await user.click(screen.getByTestId(s.includeOptional));
    await user.click(screen.getByTestId(s.submit));

    await waitFor(() => expect(startOnboarding).toHaveBeenCalledTimes(1));
    const input = startOnboarding.mock.calls[0]?.[0];
    expect(input?.setupEnvironment).toBe("production");
    expect(input?.setupResources).toBe("enabled");
    expect(input?.setupScenarios).toBe("none");
    expect(input?.includeOptional).toBe(true);
    expect(input?.controlPlaneUrl).toBe("http://control-plane.example.com:8080");
  });

  it("omits the setup profile (blank fields, optional off) by default", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-def" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-def", state: OnboardingState.BOOTSTRAPPING }, [makeStepEvent()]),
    );

    renderWithProviders(<OnboardNodeForm />);
    await startOnboard();
    await waitFor(() => expect(startOnboarding).toHaveBeenCalledTimes(1));
    const input = startOnboarding.mock.calls[0]?.[0];
    expect(input?.setupEnvironment).toBe("");
    expect(input?.setupResources).toBe("");
    expect(input?.setupScenarios).toBe("");
    expect(input?.includeOptional).toBe(false);
    // Blank control-plane URL falls through to the server-side default.
    expect(input?.controlPlaneUrl).toBe("");
  });

  it("defaults to working-tree source and switches to pinned (with a push note) when unchecked", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-src" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-src", state: OnboardingState.BOOTSTRAPPING }, [makeStepEvent()]),
    );

    // Default: working-tree (ship this computer's files) — checked, no pinned push note.
    let user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);
    await toReview(user);
    await openAdvanced(user);
    expect(screen.getByTestId(s.sourceWorkingTree)).toBeChecked();
    expect(screen.queryByTestId(s.sourceWarning)).toBeNull();
    await user.click(screen.getByTestId(s.submit));
    await waitFor(() => expect(startOnboarding).toHaveBeenCalledTimes(1));
    expect(startOnboarding.mock.calls[0]?.[0]?.sourceMode).toBe(SourceMode.WORKING_TREE);

    cleanup();
    vi.clearAllMocks();
    startOnboarding.mockResolvedValue({ opId: "op-pinned" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-pinned", state: OnboardingState.BOOTSTRAPPING }, [makeStepEvent()]),
    );

    // Pinned: unchecking reveals the "must be pushed" note and sends PINNED_REVISION.
    user = userEvent.setup();
    renderWithProviders(<OnboardNodeForm />);
    await toReview(user);
    await openAdvanced(user);
    await user.click(screen.getByTestId(s.sourceWorkingTree));
    expect(screen.getByTestId(s.sourceWarning)).toBeInTheDocument();
    await user.click(screen.getByTestId(s.submit));
    await waitFor(() => expect(startOnboarding).toHaveBeenCalledTimes(1));
    expect(startOnboarding.mock.calls[0]?.[0]?.sourceMode).toBe(SourceMode.PINNED_REVISION);
  });
});

describe("OnboardNodeForm progress", () => {
  it("renders live step states from the op's event history", async () => {
    startOnboarding.mockResolvedValue({ opId: "op-live" });
    getOnboarding.mockResolvedValue(
      makeGetOnboardingResponse({ id: "op-live", state: OnboardingState.BOOTSTRAPPING }, [
        makeStepEvent({ sequence: 1n, stepId: "ssh-setup", status: OnboardingStepStatus.OK }),
        makeStepEvent({ sequence: 2n, stepId: "clone", status: OnboardingStepStatus.STARTED, detail: "git clone" }),
      ]),
    );

    renderWithProviders(<OnboardNodeForm />);
    await startOnboard();

    expect(await screen.findByTestId(s.progress)).toBeInTheDocument();
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
    await startOnboard();

    const banner = await screen.findByTestId(s.success);
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
      await startOnboard();

      const banner = await screen.findByTestId(s.failure);
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

  it("surfaces a StartOnboarding error and stays on the wizard", async () => {
    startOnboarding.mockRejectedValue(new ConnectError("host unreachable", Code.InvalidArgument));

    renderWithProviders(<OnboardNodeForm />);
    await startOnboard();

    expect(await screen.findByTestId(s.error)).toBeInTheDocument();
    // The wizard is still shown (Start button present) so the operator can retry.
    expect(screen.getByTestId(s.submit)).toBeInTheDocument();
  });
});
