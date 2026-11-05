import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import type { ApprovalDecision } from "../../../api/approvals";
import { parseApprovalsDeepLink, useApprovalsStore } from "../../../stores/useApprovalsStore";
import {
  approveOptionKey,
  isActionable,
  recommendedOption,
  relativeTime,
  safeContextUrl,
  secondaryOptions,
  sortDecisions,
  urgencyRank,
} from "../approvalsFormat";
import ApprovalContextPanel from "../ApprovalContextPanel";
import { renderWithProviders } from "../../../test-utils";
import { setLocale } from "../../../i18n";
import ApprovalsDrawer from "../ApprovalsDrawer";
import { useApprovals } from "../../../hooks/useApprovals";
import { answerApproval, getApproval, ApprovalAlreadyResolvedError, ApprovalsUnauthorizedError, ApprovalsUnavailableError } from "../../../api/approvals";

vi.mock("../../../hooks/useApprovals", () => ({ useApprovals: vi.fn() }));
vi.mock("../../../api/approvals", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../api/approvals")>();
  return { ...actual, answerApproval: vi.fn(), getApproval: vi.fn() };
});
vi.mock("../../../hooks/useMediaQuery", () => ({ useMediaQuery: () => false }));
afterEach(() => {
  vi.clearAllMocks();
  useApprovalsStore.setState({ pendingDeepLinkId: null, shouldOpenDrawer: false });
});

function decision(over: Partial<ApprovalDecision>): ApprovalDecision {
  return {
    id: "a",
    question: "q",
    options: [
      { key: "yes", label: "Yes" },
      { key: "no", label: "No" },
    ],
    recommended: "yes",
    recommendationReason: "",
    defaultAnswer: "",
    reversible: true,
    urgency: "normal",
    contextUrl: "",
    bigPicture: "",
    technicalDetails: "",
    deadline: "",
    state: "pending",
    answer: "",
    answerLabel: "",
    note: "",
    answeredBy: "",
    answeredAt: "",
    late: false,
    defaultAppliedAt: "",
    resolvedAt: "",
    createdAt: new Date().toISOString(),
    updatedAt: "",
    source: "agent-manager",
    sourceEventType: "agent-manager.run-review.decision_requested.v1",
    kind: "run-review",
    ...over,
  };
}

describe("approvalsFormat", () => {
  it("ranks urgency highest-first", () => {
    expect(urgencyRank("critical")).toBeGreaterThan(urgencyRank("high"));
    expect(urgencyRank("high")).toBeGreaterThan(urgencyRank("normal"));
    expect(urgencyRank("")).toBe(0);
  });

  it("sorts by urgency then soonest deadline", () => {
    const now = Date.now();
    const low = decision({ id: "low", urgency: "low" });
    const crit = decision({ id: "crit", urgency: "critical" });
    const highSoon = decision({ id: "soon", urgency: "high", deadline: new Date(now + 60_000).toISOString() });
    const highLate = decision({ id: "late", urgency: "high", deadline: new Date(now + 600_000).toISOString() });
    const order = sortDecisions([low, highLate, crit, highSoon]).map((d) => d.id);
    expect(order).toEqual(["crit", "soon", "late", "low"]);
  });

  it("resolves the recommended option and the rest", () => {
    const d = decision({});
    expect(recommendedOption(d)).toEqual({ key: "yes", label: "Yes" });
    expect(secondaryOptions(d).map((o) => o.key)).toEqual(["no"]);
  });

  it("marks only a pending decision actionable", () => {
    expect(isActionable(decision({ state: "pending" }))).toBe(true);
    expect(isActionable(decision({ state: "expired" }))).toBe(false);
    expect(isActionable(decision({ state: "answered" }))).toBe(false);
  });

  it("formats relative time for past and future", () => {
    const now = Date.now();
    expect(relativeTime(new Date(now + 2 * 3600_000).toISOString(), now)).toBe("in 2h");
    expect(relativeTime(new Date(now - 6 * 60_000).toISOString(), now)).toBe("6m ago");
    expect(relativeTime("", now)).toBe("");
  });

  it("approveOptionKey returns the recommended option only when valid", () => {
    expect(approveOptionKey(decision({ recommended: "yes" }))).toBe("yes");
    // An invalid recommendation must NOT fall back to options[0] for approve.
    expect(approveOptionKey(decision({ recommended: "nonexistent" }))).toBeNull();
    expect(approveOptionKey(decision({ recommended: "" }))).toBeNull();
  });
});

describe("safeContextUrl", () => {
  it("allows http(s) and root-relative URLs", () => {
    expect(safeContextUrl("https://vrooli.com/x")).toBe("https://vrooli.com/x");
    expect(safeContextUrl("http://localhost:8080/y")).toBe("http://localhost:8080/y");
    expect(safeContextUrl("/scenarios/deployment-manager/release/1")).toBe("/scenarios/deployment-manager/release/1");
  });

  it("rejects dangerous or protocol-relative schemes (XSS guard)", () => {
    expect(safeContextUrl("javascript:alert(1)")).toBeNull();
    expect(safeContextUrl("JavaScript:alert(1)")).toBeNull();
    expect(safeContextUrl("data:text/html,<script>alert(1)</script>")).toBeNull();
    expect(safeContextUrl("vbscript:msgbox(1)")).toBeNull();
    expect(safeContextUrl("//evil.com/x")).toBeNull();
    expect(safeContextUrl("")).toBeNull();
    expect(safeContextUrl("   ")).toBeNull();
    expect(safeContextUrl("not a url")).toBeNull();
  });
});

describe("ApprovalContextPanel", () => {
  it("shows independent purpose and technical explanations as plain text", async () => {
    await setLocale("en");
    renderWithProviders(
      createElement(ApprovalContextPanel, {
        decision: decision({
          bigPicture: "Protect the deployment window.",
          technicalDetails: "The image digest changed.",
        }),
        contextUrl: "",
      }),
    );

    expect(screen.getByText("Protect the deployment window.")).toBeInTheDocument();
    expect(screen.getByText("The image digest changed.")).toBeInTheDocument();
    expect(screen.getByText("Why this matters")).toBeInTheDocument();
    expect(screen.getByText("Technical details")).toBeInTheDocument();
  });
});

describe("ApprovalsDrawer context interactions", () => {
  it("loads a deep-linked resolved decision outside the pending list", async () => {
    await setLocale("en");
    const resolved = decision({ id: "history", question: "Previously decided?", state: "answered", answer: "yes", answerLabel: "Yes" });
    vi.mocked(useApprovals).mockReturnValue({ data: [], isLoading: false, error: null, refetch: vi.fn() } as never);
    vi.mocked(getApproval).mockResolvedValue(resolved);
    useApprovalsStore.getState().requestOpen("history");
    renderWithProviders(createElement(ApprovalsDrawer, { open: true, onClose: vi.fn() }));
    expect(await screen.findByText("Previously decided?")).toBeInTheDocument();
    expect(screen.getByTestId("approvals-resolved-note")).toBeInTheDocument();
    expect(screen.queryByTestId("approvals-answer-recommended")).not.toBeInTheDocument();
    expect(getApproval).toHaveBeenCalledWith("history");
  });

  it("offers a retry when the approvals service is unavailable", async () => {
    await setLocale("en");
    const user = userEvent.setup();
    const refetch = vi.fn();
    vi.mocked(useApprovals).mockReturnValue({ data: [], isLoading: false, error: new ApprovalsUnavailableError(), refetch } as never);
    renderWithProviders(createElement(ApprovalsDrawer, { open: true, onClose: vi.fn() }));
    const unavailable = screen.getByTestId("approvals-unavailable");
    await user.click(within(unavailable).getByRole("button"));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("hides stale answer controls when the session expires", async () => {
    await setLocale("en");
    const user = userEvent.setup();
    vi.mocked(useApprovals).mockReturnValue({ data: [decision({})], isLoading: false, error: null, refetch: vi.fn() } as never);
    vi.mocked(answerApproval).mockRejectedValue(new ApprovalsUnauthorizedError());
    renderWithProviders(createElement(ApprovalsDrawer, { open: true, onClose: vi.fn() }));
    await user.click(await screen.findByTestId("approvals-answer-recommended"));
    expect(await screen.findByTestId("approvals-session-expired")).toBeInTheDocument();
    expect(screen.queryByTestId("approvals-answer-recommended")).not.toBeInTheDocument();
  });

  it.each(["answered", "resolved elsewhere"])("does not leave a stale actionable detail after %s", async (outcome) => {
    await setLocale("en");
    const user = userEvent.setup();
    const refetch = vi.fn().mockResolvedValue({});
    vi.mocked(useApprovals).mockReturnValue({ data: [decision({ question: "Approve the release?" })], isLoading: false, error: null, refetch } as never);
    vi.mocked(answerApproval).mockReset();
    if (outcome === "answered") vi.mocked(answerApproval).mockResolvedValue(undefined);
    else vi.mocked(answerApproval).mockRejectedValue(new ApprovalAlreadyResolvedError());
    renderWithProviders(createElement(ApprovalsDrawer, { open: true, onClose: vi.fn() }));

    await user.click(await screen.findByTestId("approvals-answer-recommended"));
    await waitFor(() => expect(refetch).toHaveBeenCalled());
    expect(screen.queryByTestId("approvals-answer-recommended")).not.toBeInTheDocument();
    if (outcome === "resolved elsewhere") expect(screen.getByTestId("approvals-already-resolved")).toBeInTheDocument();
  });

  it("keeps the selected decision and answer note while both disclosures toggle", async () => {
    await setLocale("en");
    const user = userEvent.setup();
    const selectedDecision = decision({
      id: "ask-context",
      question: "Keep the release open?",
      bigPicture: "Protect the deployment window.",
      technicalDetails: "The image digest changed.",
    });
    vi.mocked(useApprovals).mockReturnValue({
      data: [selectedDecision],
      isLoading: false,
      error: null,
      refetch: vi.fn().mockResolvedValue({}),
    } as never);
    vi.mocked(answerApproval).mockResolvedValue(undefined);
    const { renderWithProviders: render } = await import("../../../test-utils");
    render(createElement(ApprovalsDrawer, { open: true, onClose: vi.fn() }));

    const detail = await screen.findByTestId("approvals-detail");
    expect(within(detail).getByText("Keep the release open?")).toBeInTheDocument();
    const note = within(detail).getByTestId("approvals-note");
    await user.type(note, "inspect the rebuilt artifact");
    for (const label of ["Why this matters", "Technical details"]) {
      const summary = within(detail).getByText(label);
      const disclosure = summary.closest("details");
      expect(disclosure).not.toBeNull();
      await user.click(summary);
      expect(disclosure).toHaveProperty("open", true);
      expect(note).toHaveValue("inspect the rebuilt artifact");
      expect(within(detail).getByText("Keep the release open?")).toBeInTheDocument();
      expect(answerApproval).not.toHaveBeenCalled();
      await user.click(summary);
      expect(disclosure).toHaveProperty("open", false);
      expect(note).toHaveValue("inspect the rebuilt artifact");
      expect(answerApproval).not.toHaveBeenCalled();
    }

    await user.click(within(detail).getByTestId("approvals-answer-recommended"));
    await waitFor(() => expect(answerApproval).toHaveBeenCalledTimes(1));
    expect(answerApproval).toHaveBeenCalledWith("ask-context", "yes", "inspect the rebuilt artifact");
  });
});

describe("ApprovalsDrawer irreversible confirmation", () => {
  it("keeps the confirmation named and modal, and only submits after confirm", async () => {
    await setLocale("en");
    const user = userEvent.setup();
    vi.mocked(answerApproval).mockClear();
    const irreversibleDecision = decision({
      id: "irreversible-approval",
      question: "Delete the deployment record?",
      reversible: false,
    });
    vi.mocked(useApprovals).mockReturnValue({
      data: [irreversibleDecision],
      isLoading: false,
      error: null,
      refetch: vi.fn().mockResolvedValue({}),
    } as never);
    vi.mocked(answerApproval).mockResolvedValue(undefined);
    renderWithProviders(createElement(ApprovalsDrawer, { open: true, onClose: vi.fn() }));

    const detail = await screen.findByTestId("approvals-detail");
    const answerButton = within(detail).getByTestId("approvals-answer-recommended");
    await user.click(answerButton);

    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    const labelledBy = dialog.getAttribute("aria-labelledby");
    const describedBy = dialog.getAttribute("aria-describedby");
    expect(document.getElementById(labelledBy as string)?.textContent).toBe("Confirm your decision");
    expect(document.getElementById(describedBy as string)?.textContent).toContain("This can't be undone.");
    expect(document.getElementById(describedBy as string)?.textContent).toContain("Yes for: Delete the deployment record?");
    expect(answerApproval).not.toHaveBeenCalled();

    await user.click(within(dialog).getByTestId("approvals-confirm-cancel"));
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(answerApproval).not.toHaveBeenCalled();

    await user.click(answerButton);
    const reopenedDialog = await screen.findByRole("alertdialog");
    await user.click(within(reopenedDialog).getByTestId("approvals-confirm-confirm"));
    await waitFor(() => expect(answerApproval).toHaveBeenCalledTimes(1));
    expect(answerApproval).toHaveBeenCalledWith("irreversible-approval", "yes", "");
  });
});

describe("parseApprovalsDeepLink", () => {
  it("opens on an explicit approval id", () => {
    expect(parseApprovalsDeepLink("?approval=ask-1")).toEqual({ open: true, decisionId: "ask-1" });
  });
  it("accepts the agent-manager authorization/run aliases", () => {
    expect(parseApprovalsDeepLink("?authorization=ask-9")).toEqual({ open: true, decisionId: "ask-9" });
    expect(parseApprovalsDeepLink("?run=ask-7")).toEqual({ open: true, decisionId: "ask-7" });
  });
  it("opens with no specific decision on the flag", () => {
    expect(parseApprovalsDeepLink("?approvals")).toEqual({ open: true, decisionId: null });
    expect(parseApprovalsDeepLink("?approvals=1")).toEqual({ open: true, decisionId: null });
  });
  it("does not open without an approvals param", () => {
    expect(parseApprovalsDeepLink("?foo=bar")).toEqual({ open: false, decisionId: null });
    expect(parseApprovalsDeepLink("")).toEqual({ open: false, decisionId: null });
  });
});
