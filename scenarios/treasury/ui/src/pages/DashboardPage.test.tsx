import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { ApprovalRequestSchema, ApprovalStatus } from "@vrooli/proto-types/treasury/v1/approval/approval_pb";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { DashboardPage } from "./DashboardPage";

const approval = create(ApprovalRequestSchema, {
  id: "approval-1",
  authorizationId: "authorization-1",
  bookId: "book-1",
  mandateId: "mandate-1",
  requestingAgent: "procurement-agent",
  amountMinor: 12_345n,
  currency: "USD",
  counterparty: "Acme Supplies",
  status: ApprovalStatus.QUEUED,
  expiresAt: create(TimestampSchema, { seconds: 1_800_000_000n }),
});

const mocks = vi.hoisted(() => ({
  listPendingApprovals: vi.fn(),
  resolveApproval: vi.fn(),
}));

vi.mock("../api/approvals", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/approvals")>()),
  ...mocks,
}));

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("Approval queue", () => {
  it("supports a keyboard-only approval with an amount-and-counterparty accessible name", async () => {
    mocks.listPendingApprovals.mockResolvedValue([approval]);
    mocks.resolveApproval.mockResolvedValue({ ...approval, status: ApprovalStatus.APPROVED });
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);

    await user.tab();
    expect(screen.getByTestId(selectors.approvals.tokenInput)).toHaveFocus();
    await user.keyboard("operator-secret");
    await user.tab();
    expect(screen.getByTestId(selectors.approvals.openQueueButton)).toHaveFocus();
    await user.keyboard("{Enter}");

    const approve = await screen.findByRole("button", {
      name: "Approve $123.45 payment to Acme Supplies",
    });
    approve.focus();
    await user.keyboard("x");
    expect(mocks.resolveApproval).not.toHaveBeenCalled();
    await user.keyboard("{Enter}");

    await waitFor(() => expect(mocks.resolveApproval).toHaveBeenCalledWith(
      "operator-secret",
      "approval-1",
      ApprovalStatus.APPROVED,
    ));
    expect(await screen.findByRole("status")).toHaveTextContent("Payment to Acme Supplies approved.");
    expect(screen.queryByTestId(selectors.approvals.item({ id: "approval-1" }))).not.toBeInTheDocument();
  });

  it("has no detectable accessibility violations before authentication", async () => {
    const { container } = renderWithProviders(<DashboardPage />);
    await expectNoA11yViolations(container);
  });

  it("shows non-color status text and exposes both decision names", async () => {
    mocks.listPendingApprovals.mockResolvedValue([approval]);
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    await user.type(screen.getByTestId(selectors.approvals.tokenInput), "operator-secret");
    await user.click(screen.getByTestId(selectors.approvals.openQueueButton));

    expect(await screen.findByTestId(selectors.approvals.item({ id: "approval-1" }))).toHaveTextContent("Pending review");
    expect(screen.getByRole("button", { name: "Approve $123.45 payment to Acme Supplies" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Decline $123.45 payment to Acme Supplies" })).toBeVisible();
  });

  // [REQ:TRS-P1-004]
  it("keeps approval chains visibly separated by book", async () => {
    const second = create(ApprovalRequestSchema, {
      ...approval,
      id: "approval-2",
      authorizationId: "authorization-2",
      bookId: "book-2",
      counterparty: "Beta Supplies",
    });
    mocks.listPendingApprovals.mockResolvedValue([approval, second]);
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    await user.type(screen.getByTestId(selectors.approvals.tokenInput), "operator-secret");
    await user.click(screen.getByTestId(selectors.approvals.openQueueButton));

    expect(await screen.findByTestId(selectors.approvals.item({ id: "approval-1" }))).toBeVisible();
    expect(screen.queryByTestId(selectors.approvals.item({ id: "approval-2" }))).not.toBeInTheDocument();
    await user.selectOptions(screen.getByTestId(selectors.approvals.bookSelect), "book-2");
    expect(screen.getByTestId(selectors.approvals.item({ id: "approval-2" }))).toBeVisible();
    expect(screen.queryByTestId(selectors.approvals.item({ id: "approval-1" }))).not.toBeInTheDocument();
  });

  it("reports authentication failures without exposing the token", async () => {
    mocks.listPendingApprovals.mockRejectedValue(new Error("backend included operator-secret"));
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);

    await user.type(screen.getByTestId(selectors.approvals.tokenInput), "operator-secret");
    await user.click(screen.getByTestId(selectors.approvals.openQueueButton));

    expect(await screen.findByRole("alert")).toHaveTextContent("Authentication failed");
    expect(screen.getByRole("alert")).not.toHaveTextContent("operator-secret");
  });

  it("keeps a charge visible when resolution fails and supports Space to decline", async () => {
    mocks.listPendingApprovals.mockResolvedValue([approval]);
    mocks.resolveApproval.mockRejectedValue(new Error("transport failure"));
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    await user.type(screen.getByTestId(selectors.approvals.tokenInput), "operator-secret");
    await user.click(screen.getByTestId(selectors.approvals.openQueueButton));

    const decline = await screen.findByRole("button", {
      name: "Decline $123.45 payment to Acme Supplies",
    });
    decline.focus();
    await user.keyboard(" ");

    await waitFor(() => expect(mocks.resolveApproval).toHaveBeenCalledWith(
      "operator-secret",
      "approval-1",
      ApprovalStatus.DECLINED,
    ));
    expect(await screen.findByRole("alert")).toHaveTextContent("could not be recorded");
    expect(screen.getByTestId(selectors.approvals.item({ id: "approval-1" }))).toBeVisible();
  });
});
