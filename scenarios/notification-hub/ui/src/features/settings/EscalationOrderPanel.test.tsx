import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { recipientsClient } from "../../api/notifications";
import { i18n } from "../../i18n";
import { EscalationOrderPanel } from "./EscalationOrderPanel";

vi.mock("../../api/notifications", () => ({ recipientsClient: {
  listDevices: vi.fn().mockResolvedValue({ devices: [{ id: "d1", channels: ["email", "webhook"] }] }),
  getEscalationChain: vi.fn().mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }, { ordinal: 1, channel: "future-channel" }] }),
  setEscalationChain: vi.fn().mockResolvedValue({ steps: [] }),
} }));

describe("EscalationOrderPanel", () => {
  it("opens the channel-choice screen and stages the selected channel without saving", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.getEscalationChain).mockReset().mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }] } as never);
    vi.mocked(recipientsClient.listDevices).mockReset().mockResolvedValue({ devices: [{ id: "d1", channels: ["email", "webhook"] }] } as never);
    vi.mocked(recipientsClient.setEscalationChain).mockReset();
    renderWithProviders(<EscalationOrderPanel focused />);
    await screen.findByRole("list", { name: "Escalation order" });
    expect(screen.queryByTestId("escalation-order-channel-picker")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add channel" }));
    expect(await screen.findByRole("heading", { name: "Add a channel" })).toBeInTheDocument();
    expect(screen.getByText("Availability is checked separately")).toBeInTheDocument();
    expect(screen.getByText("Availability is checked separately").closest("aside")).toHaveStyle({ borderInlineStartColor: "var(--reach-guidance-rail)" });
    expect(screen.getByTestId("escalation-order-add-back")).toHaveClass("min-h-11");
    expect(screen.getByTestId("escalation-order-add-back").parentElement).toHaveClass("mt-4");
    await user.click(screen.getByRole("button", { name: /Webhook/ }));
    expect(screen.queryByTestId("escalation-order-channel-picker")).not.toBeInTheDocument();
    expect(screen.getByTestId("escalation-order-channel-webhook")).toBeInTheDocument();
    expect(vi.mocked(recipientsClient.setEscalationChain)).not.toHaveBeenCalled();
  });

  it("returns to the saved order without changing it from channel choices", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.getEscalationChain).mockReset().mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }] } as never);
    vi.mocked(recipientsClient.setEscalationChain).mockReset();
    renderWithProviders(<EscalationOrderPanel focused />);
    await screen.findByRole("list", { name: "Escalation order" });
    await user.click(screen.getByRole("button", { name: "Add channel" }));
    await user.click(screen.getByRole("button", { name: "Back to order" }));
    expect(screen.getByRole("list", { name: "Escalation order" })).toBeInTheDocument();
    expect(screen.getByText("Saved order:").parentElement).toHaveTextContent("Email");
    expect(vi.mocked(recipientsClient.setEscalationChain)).not.toHaveBeenCalled();
  });

  it("keeps unknown keys visible and blocks saving until their removal is staged explicitly", async () => {
    const user = userEvent.setup();
    const onEditorHeaderChange = vi.fn();
    const get = vi.mocked(recipientsClient.getEscalationChain);
    get.mockReset().mockResolvedValueOnce({ steps: [{ ordinal: 0, channel: "email" }, { ordinal: 1, channel: "future-channel" }] } as never)
      .mockResolvedValueOnce({ steps: [{ ordinal: 0, channel: "email" }] } as never);
    const set = vi.mocked(recipientsClient.setEscalationChain);
    set.mockReset().mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }] } as never);
    renderWithProviders(<EscalationOrderPanel focused onEditorHeaderChange={onEditorHeaderChange} />);
    const orderedList = await screen.findByRole("list", { name: "Escalation order" });
    expect(screen.getByRole("heading", { name: "Review this saved channel" })).toBeInTheDocument();
    await waitFor(() => expect(onEditorHeaderChange).toHaveBeenLastCalledWith(expect.objectContaining({
      title: "Escalation order",
      backLabel: "Order",
    })));
    expect(screen.getByText("Escalation order · unsupported entry")).toBeInTheDocument();
    expect(within(orderedList).getByText(/future-channel/)).toBeInTheDocument();
    expect(screen.getByText("Unrecognized saved identifier. Preserved unchanged; target availability is unknown.")).toBeInTheDocument();
    expect(await screen.findByTestId("escalation-order-review-required")).toHaveTextContent("A target needs review");
    expect(screen.getByTestId("escalation-order-policy")).toHaveTextContent("Only overdue critical asks use this order");
    expect(within(orderedList).getByText(/1\. Email/)).toBeInTheDocument();
    expect(within(orderedList).getByText(/2\. future-channel/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save order" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Keep draft" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add channel" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Clear order/ })).not.toBeInTheDocument();
    expect(set).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Remove future-channel" }));
    expect(screen.queryByTestId("escalation-order-channel-future-channel")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save order" })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "Save order" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Order saved and confirmed");
    expect(set).toHaveBeenCalledWith({ channels: ["email"] });
    expect(get).toHaveBeenCalledTimes(2);
  });

  it("requires an explicit confirmation before staging a full clear", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.getEscalationChain).mockReset().mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }] } as never);
    vi.mocked(recipientsClient.setEscalationChain).mockReset().mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }] } as never);
    renderWithProviders(<EscalationOrderPanel focused />);
    await user.click(await screen.findByRole("button", { name: /Clear order/ }));
    expect(screen.getByRole("heading", { name: "Clear escalation order?" })).toBeInTheDocument();
    expect(screen.getByText("Current order").nextElementSibling).toHaveTextContent("Email");
    expect(screen.getByTestId("escalation-order-policy")).toHaveTextContent("Only overdue critical asks use this order");
    expect(vi.mocked(recipientsClient.setEscalationChain)).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Keep editing" }));
    expect(screen.queryByRole("heading", { name: "Clear escalation order?" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Clear order/ }));
    await user.click(screen.getByRole("button", { name: "Clear this draft order" }));
    expect(screen.getByRole("heading", { name: "No escalation channels" })).toBeInTheDocument();
    expect(screen.getByTestId("escalation-order-empty-draft")).toHaveTextContent("The cleared order is not saved yet");
    expect(screen.getByText("Your saved order is still Email.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save empty order" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Restore saved order" })).toBeEnabled();
    expect(vi.mocked(recipientsClient.setEscalationChain)).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Restore saved order" }));
    expect(screen.getByRole("list", { name: "Escalation order" })).toHaveTextContent("Email");
    expect(screen.queryByRole("button", { name: "Save empty order" })).not.toBeInTheDocument();
    expect(vi.mocked(recipientsClient.setEscalationChain)).not.toHaveBeenCalled();
  });

  it.each([
    ["en", "Only overdue critical asks use this order. Non-critical asks expire unless a separately governed reversible default applies.", "Timing is fixed at 5 minutes per response step; target availability and content approval are checked separately."],
    ["ja", "期限超過の重大な依頼のみがこの順序を使用します。非重大な依頼は、別途管理される取消可能な既定対応が適用されない限り期限切れになります。", "応答の間隔は5分固定です。通知先の利用可否と内容の承認は別途確認されます。"],
    ["ar", "ينطبق هذا الترتيب على الطلبات الحرجة المتأخرة فقط. تنتهي الطلبات غير الحرجة ما لم ينطبق خيار افتراضي قابل للعكس يخضع لإدارة منفصلة.", "التوقيت ثابت عند 5 دقائق لكل خطوة؛ ويتم التحقق بشكل منفصل من توفر الهدف وموافقة المحتوى."],
  ])("shows the operative critical-only policy in every rendered settings state (%s)", async (locale, policy, timing) => {
    const user = userEvent.setup();
    const noStaleHold = () => expect(screen.queryByText(/Eligibility rules remain under product review|対象となる条件は製品判断待ち|قواعد الأهلية قيد مراجعة المنتج/)).not.toBeInTheDocument();
    await i18n.changeLanguage(locale);
    vi.mocked(recipientsClient.getEscalationChain).mockReset().mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }] } as never);
    vi.mocked(recipientsClient.setEscalationChain).mockReset();

    const overview = renderWithProviders(<EscalationOrderPanel />);
    expect(await screen.findByTestId("escalation-order-policy")).toHaveTextContent(policy);
    noStaleHold();
    overview.unmount();
    const editor = renderWithProviders(<EscalationOrderPanel focused />);
    expect(await screen.findByTestId("escalation-order-policy")).toHaveTextContent(policy);
    noStaleHold();
    expect(await screen.findByTestId("escalation-order-list")).toBeInTheDocument();
    expect(screen.getByText(timing)).toBeInTheDocument();
    expect(screen.getByTestId("escalation-order-policy")).toHaveTextContent(policy);

    await user.click(screen.getByTestId("escalation-order-clear"));
    expect(screen.getByTestId("escalation-order-policy")).toHaveTextContent(policy);
    noStaleHold();
    await user.click(screen.getByTestId("escalation-order-confirm-clear"));
    expect(screen.getByTestId("escalation-order-empty-draft")).toBeInTheDocument();
    expect(screen.getByTestId("escalation-order-policy")).toHaveTextContent(policy);
    noStaleHold();
    expect(vi.mocked(recipientsClient.setEscalationChain)).not.toHaveBeenCalled();
    editor.unmount();
    await i18n.changeLanguage("en");
  });

  it("retains previous and attempted orders after an unknown save and checks without replaying", async () => {
    const user = userEvent.setup();
    const get = vi.mocked(recipientsClient.getEscalationChain);
    get.mockReset().mockResolvedValueOnce({ steps: [{ ordinal: 0, channel: "email" }, { ordinal: 1, channel: "webhook" }] } as never)
      .mockRejectedValueOnce(new Error("read unavailable"))
      .mockResolvedValueOnce({ steps: [{ ordinal: 0, channel: "webhook" }, { ordinal: 1, channel: "email" }] } as never);
    const set = vi.mocked(recipientsClient.setEscalationChain);
    set.mockReset().mockRejectedValueOnce(new Error("response lost"));
    renderWithProviders(<EscalationOrderPanel focused />);
    await screen.findByRole("list", { name: "Escalation order" });
    await user.click(screen.getByRole("button", { name: "Move down email" }));
    await user.click(screen.getByRole("button", { name: "Save order" }));
    expect(await screen.findByRole("status")).toHaveTextContent("The save result is unknown");
    expect(screen.getByTestId("escalation-order-unknown-retained")).toHaveTextContent("Your attempted order is retained. No automatic retry or change to quiet hours has been made.");
    expect(screen.getByRole("heading", { name: "The save result is unknown" })).toBeInTheDocument();
    expect(screen.getByText("Read the saved order before making another change.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Keep draft" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add channel" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Clear order/ })).not.toBeInTheDocument();
    expect(screen.queryByText(/Eligibility rules remain under product review/)).not.toBeInTheDocument();
    expect(screen.getByText("Previously loaded").nextElementSibling).toHaveTextContent("Email → Webhook");
    expect(screen.getByText("Attempted order").nextElementSibling).toHaveTextContent("Webhook → Email");
    expect(screen.queryByRole("list", { name: "Escalation order" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Check saved order" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Order saved and confirmed");
    expect(set).toHaveBeenCalledTimes(1);
    expect(get).toHaveBeenCalledTimes(3);
  });

  it("discards an unsaved reorder when the owner cancels back to the overview", async () => {
    const user = userEvent.setup();
    vi.mocked(recipientsClient.getEscalationChain).mockReset().mockResolvedValue({ steps: [{ ordinal: 0, channel: "email" }, { ordinal: 1, channel: "webhook" }] } as never);
    vi.mocked(recipientsClient.setEscalationChain).mockReset();
    function ControlledPanel() {
      const [focused, setFocused] = useState(true);
      return <EscalationOrderPanel focused={focused} onFocusedChange={setFocused} />;
    }
    renderWithProviders(<ControlledPanel />);
    const list = await screen.findByRole("list", { name: "Escalation order" });
    await user.click(screen.getByRole("button", { name: "Move down email" }));
    expect(within(list).getByText("1. Webhook")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("list", { name: "Escalation order" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Edit order" }));
    const restored = await screen.findByRole("list", { name: "Escalation order" });
    expect(within(restored).getByText("1. Email")).toBeInTheDocument();
    expect(within(restored).getByText("2. Webhook")).toBeInTheDocument();
    expect(vi.mocked(recipientsClient.setEscalationChain)).not.toHaveBeenCalled();
  });

  it("publishes the localized contextual shell header while the order editor is focused", () => {
    const onEditorHeaderChange = vi.fn();
    const onFocusedChange = vi.fn();
    const { rerender } = renderWithProviders(
      <EscalationOrderPanel focused onFocusedChange={onFocusedChange} onEditorHeaderChange={onEditorHeaderChange} />,
    );
    const header = onEditorHeaderChange.mock.calls.map(([value]) => value).find((value) => value !== null);
    expect(header).toMatchObject({ title: "Channel order", backLabel: "Settings" });
    expect(screen.queryByRole("button", { name: "Back to Settings" })).not.toBeInTheDocument();
    header.onBack();
    expect(onFocusedChange).toHaveBeenCalledWith(false);

    rerender(<EscalationOrderPanel focused={false} onFocusedChange={onFocusedChange} onEditorHeaderChange={onEditorHeaderChange} />);
    expect(onEditorHeaderChange).toHaveBeenLastCalledWith(null);
  });
});
