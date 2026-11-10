import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { conversationsClient } from "../../api/notifications";
import { renderWithProviders, expectNoA11yViolations } from "../../test-utils";
import { NativeConsents } from "./NativeConsents";
import { setLocale } from "../../i18n";

vi.mock("../../api/notifications", () => ({ conversationsClient: { listNativeConsents: vi.fn(), answer: vi.fn() } }));
const list = vi.mocked(conversationsClient.listNativeConsents);
const consent = { authorizationId: "consent-1", notificationId: "notification-1", source: "agent-manager", title: "Agent needs approval", body: "Ask and send for operator", state: "pending", reviewUrl: "https://agent-manager.example.test/api/v1/authorizations/consent-1/approve", observedAt: "2026-11-09T14:00:00Z", reviewExpiresAt: "2026-11-15T14:00:00Z" };
describe("Native permission visibility", () => {
 beforeEach(async () => { vi.resetAllMocks(); await setLocale("en"); });
 it("Given a pending native permission, when Needs You is read, then its source link is available without creating or answering an ask", async () => {
  list.mockResolvedValue({ consents: [consent], nextCursor: "" } as never);
  const { container } = renderWithProviders(<NativeConsents query="" />);
  expect(await screen.findByRole("link", { name: "Review in Agent Manager" })).toHaveAttribute("href", consent.reviewUrl);
  expect(screen.getByText(/Opening a link grants nothing/)).toBeInTheDocument();
  expect(conversationsClient.answer).not.toHaveBeenCalled(); await expectNoA11yViolations(container);
 });
 it("Given an unreachable source, when reading fails, then the queue is not declared clear and a read-only retry is offered", async () => {
  list.mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({ consents: [{ ...consent, state: "unavailable", observedAt: "" }], nextCursor: "" } as never);
  renderWithProviders(<NativeConsents query="" />); await screen.findByRole("alert");
  await userEvent.setup().click(screen.getByRole("button", { name: "Try again" }));
  await screen.findByText(/Current status is unverified/); expect(conversationsClient.answer).not.toHaveBeenCalled();
 });
 it("Given a closed candidate page, when older candidates remain, then the next page stays reachable", async () => {
  list.mockResolvedValueOnce({ consents: [], nextCursor: "next" } as never).mockResolvedValueOnce({ consents: [consent], nextCursor: "" } as never);
  renderWithProviders(<NativeConsents query="permission" />);
  await userEvent.setup().click(await screen.findByRole("button", { name: "Check older permission requests" }));
  await screen.findByRole("link", { name: "Review in Agent Manager" });
  expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "next", query: "permission" }), expect.anything());
 });
});
