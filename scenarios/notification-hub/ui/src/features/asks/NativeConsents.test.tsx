import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import { conversationsClient } from "../../api/notifications";
import { renderWithProviders, expectNoA11yViolations } from "../../test-utils";
import { NativeConsents } from "./NativeConsents";
import { useNativeConsents } from "./useNativeConsents";
import { setLocale } from "../../i18n";

vi.mock("../../api/notifications", () => ({ conversationsClient: { listNativeConsents: vi.fn(), answer: vi.fn() } }));
const list = vi.mocked(conversationsClient.listNativeConsents);
const consent = { authorizationId: "consent-1", notificationId: "notification-1", source: "agent-manager", title: "Agent needs approval", body: "Ask and send for operator", state: "pending", reviewUrl: "https://agent-manager.example.test/api/v1/authorizations/consent-1/approve", observedAt: "2026-11-09T14:00:00Z", reviewExpiresAt: "2026-11-15T14:00:00Z" };
/** The section as the queue page mounts it: the page owns the read. */
function Section({ query }: { query: string }) {
  return <NativeConsents list={useNativeConsents(query, true)} />;
}

describe("Native permission visibility", () => {
 beforeEach(async () => { vi.resetAllMocks(); await setLocale("en"); });
 it("Given a pending native permission, when Needs You is read, then its source link is available without creating or answering an ask", async () => {
  list.mockResolvedValue({ consents: [consent], nextCursor: "" } as never);
  const { container } = renderWithProviders(<Section query="" />);
  expect(await screen.findByRole("link", { name: "Review in Agent Manager" })).toHaveAttribute("href", consent.reviewUrl);
  expect(screen.getByText(/Opening a link grants nothing/)).toBeInTheDocument();
  expect(conversationsClient.answer).not.toHaveBeenCalled(); await expectNoA11yViolations(container);
 });
 it("Given an unreachable source, when reading fails, then this section stays out of the way for the page to report it", async () => {
  list.mockRejectedValue(new Error("offline"));
  const { container } = renderWithProviders(<Section query="" />);
  await waitFor(() => expect(list).toHaveBeenCalled());
  await waitFor(() => expect(container).toBeEmptyDOMElement());
 });
 it("Given a closed candidate page, when older candidates remain, then the next page stays reachable", async () => {
  list.mockResolvedValueOnce({ consents: [], nextCursor: "next" } as never).mockResolvedValueOnce({ consents: [consent], nextCursor: "" } as never);
  renderWithProviders(<Section query="permission" />);
  await userEvent.setup().click(await screen.findByRole("button", { name: "Check older permission requests" }));
  await screen.findByRole("link", { name: "Review in Agent Manager" });
  expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "next", query: "permission" }), expect.anything());
 });
 it("Given no Hub session, when permission requests are read, then the list owns the sign-in prompt and this section stays out of the way", async () => {
  list.mockRejectedValue(new ConnectError("no session", Code.Unauthenticated));
  const { container } = renderWithProviders(<Section query="" />);
  await waitFor(() => expect(container).toBeEmptyDOMElement());
  expect(list).toHaveBeenCalled();
 });
});
