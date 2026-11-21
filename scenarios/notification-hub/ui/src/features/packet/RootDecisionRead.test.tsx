import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { en } from "../../i18n/catalog";
import { RootDecisionRead } from "./RootDecisionRead";

beforeEach(async () => { await setLocale("en"); Object.defineProperty(window, "isSecureContext", { configurable: true, value: true }); });
afterEach(() => vi.unstubAllGlobals());
it("reads the requested root decision after password and MFA", async () => {
  const fetchMock = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ mfaRequired: true, mfaChallenge: "challenge" }))).mockResolvedValueOnce(new Response(JSON.stringify({ mfaRequired: false, ask: { id: "decision-1", source: "root-rebaseline", sourceEventType: "root.rebaseline.decision_requested.v1" } })));
  vi.stubGlobal("fetch", fetchMock);
  const onRead = vi.fn(); const user = userEvent.setup();
  renderWithProviders(<RootDecisionRead askId="decision-1" onRead={onRead} />);
  await user.click(screen.getByRole("button", { name: en.packet.rootRead.action }));
  await user.type(screen.getByLabelText(en.session.email), "owner@example.test");
  await user.type(screen.getByLabelText(en.session.password), "secret");
  await user.click(screen.getByRole("button", { name: en.packet.rootRead.action }));
  await user.type(await screen.findByLabelText(en.session.totpCode), "123456");
  await user.click(screen.getByRole("button", { name: en.packet.rootRead.action }));
  await waitFor(() => expect(onRead).toHaveBeenCalledWith(expect.objectContaining({ id: "decision-1" })));
  expect(JSON.parse(String(fetchMock.mock.calls[1]![1]!.body))).toMatchObject({ askId: "decision-1", mfaChallenge: "challenge", totpCode: "123456" });
  expect(screen.getByLabelText(en.session.password)).toHaveValue("");
});
it("does not accept a different decision or offer credentials on an insecure origin", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ mfaRequired: false, ask: { id: "other" } }))));
  const onRead = vi.fn(); const user = userEvent.setup();
  const { unmount } = renderWithProviders(<RootDecisionRead askId="decision-1" onRead={onRead} />);
  await user.click(screen.getByRole("button", { name: en.packet.rootRead.action }));
  await user.type(screen.getByLabelText(en.session.email), "owner@example.test");
  await user.type(screen.getByLabelText(en.session.password), "secret");
  await user.click(screen.getByRole("button", { name: en.packet.rootRead.action }));
  expect(await screen.findByRole("alert")).toHaveTextContent(en.packet.rootRead.error);
  expect(onRead).not.toHaveBeenCalled();
  unmount(); Object.defineProperty(window, "isSecureContext", { configurable: true, value: false });
  renderWithProviders(<RootDecisionRead askId="decision-1" onRead={onRead} />);
  expect(screen.queryByRole("button", { name: en.packet.rootRead.action })).not.toBeInTheDocument();
});
