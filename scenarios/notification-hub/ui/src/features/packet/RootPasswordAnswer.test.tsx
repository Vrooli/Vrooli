import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { AskSchema, AskOptionSchema } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { conversationsClient } from "../../api/notifications";
import { i18n } from "../../i18n";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { RootPasswordAnswer } from "./RootPasswordAnswer";

vi.mock("../../api/notifications", () => ({ conversationsClient: { answerRootWithPassword: vi.fn(), getAsk: vi.fn() } }));
const ask = create(AskSchema, { id: "decision-1", revisionCount: 3, options: [create(AskOptionSchema, { key: "approve", label: "Approve" }), create(AskOptionSchema, { key: "reject", label: "Reject" })] });
const setup = () => renderWithProviders(<RootPasswordAnswer ask={ask} email="" onCancel={vi.fn()} onRecorded={vi.fn()} onChanged={vi.fn()} />);
async function credentials() {
  const user = userEvent.setup();
  await user.click(screen.getByRole("radio", { name: "Approve" }));
  await user.type(screen.getByLabelText(i18n.t(strings.packet.rootPassword.email)), "owner@example.test");
  await user.type(screen.getByLabelText(i18n.t(strings.packet.rootPassword.password)), "example-secret");
  await user.click(screen.getByRole("button", { name: "Verify and send answer" }));
  return user;
}
beforeEach(async () => { vi.resetAllMocks(); await i18n.changeLanguage("en"); });
describe("decision-bound password answer", () => {
  it("does not offer or send credentials in an insecure context", () => {
    Object.defineProperty(window, "isSecureContext", { configurable: true, value: false });
    setup();
    expect(screen.getByRole("alert")).toHaveTextContent(i18n.t(strings.session.secureRequired));
    expect(screen.queryByLabelText(i18n.t(strings.packet.rootPassword.password))).not.toBeInTheDocument();
    expect(conversationsClient.answerRootWithPassword).not.toHaveBeenCalled();
  });
  it("binds the answer and revision through MFA, without storing credentials", async () => {
    vi.mocked(conversationsClient.answerRootWithPassword).mockResolvedValueOnce({ mfaRequired: true, mfaChallenge: "challenge", answeredAt: "" } as never).mockResolvedValueOnce({ mfaRequired: false, mfaChallenge: "", answeredAt: "now" } as never);
    setup(); const user = await credentials();
    expect(await screen.findByLabelText(i18n.t(strings.packet.rootPassword.authenticatorCode))).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Approve" })).toBeDisabled();
    await user.type(screen.getByLabelText(i18n.t(strings.packet.rootPassword.authenticatorCode)), "123456");
    await user.click(screen.getByRole("button", { name: "Verify code and send answer" }));
    await screen.findByText(i18n.t(strings.packet.rootPassword.recorded));
    expect(conversationsClient.answerRootWithPassword).toHaveBeenLastCalledWith(expect.objectContaining({ askId: "decision-1", revisionCount: 3, answer: "approve", mfaChallenge: "challenge", totpCode: "123456" }));
    expect(JSON.stringify(localStorage)).not.toContain("example-secret");
  });
  it("checks status before allowing a retry when the answer response is lost", async () => {
    vi.mocked(conversationsClient.answerRootWithPassword).mockRejectedValueOnce(new Error("lost response"));
    vi.mocked(conversationsClient.getAsk).mockResolvedValueOnce({ ask: create(AskSchema, { id: ask.id, state: "pending", revisionCount: 3 }) } as never);
    setup(); const user = await credentials();
    await screen.findByRole("alert");
    expect(screen.queryByRole("button", { name: "Verify and send answer" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Check decision status" }));
    await waitFor(() => expect(screen.getByText(/still open/)).toBeInTheDocument());
    expect(screen.getByLabelText(i18n.t(strings.packet.rootPassword.password))).toHaveValue("");
    expect(conversationsClient.answerRootWithPassword).toHaveBeenCalledTimes(1);
  });
});
