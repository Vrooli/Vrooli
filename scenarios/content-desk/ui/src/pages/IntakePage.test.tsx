import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";
import { IntakePage } from "./IntakePage";

const mocks = vi.hoisted(() => ({ preview: vi.fn(), admit: vi.fn(), list: vi.fn(), reconcile: vi.fn(), promote: vi.fn() }));
vi.mock("../api/intake", () => ({ intakeClient: mocks }));

function renderIntake() {
  return renderWithProviders(<MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}><IntakePage /></MemoryRouter>, { withoutRouter: true });
}

describe("IntakePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.preview.mockResolvedValue({ resultJson: JSON.stringify({ assessments: [{ external_id: "article-1", status: "admissible" }] }) });
    mocks.admit.mockResolvedValue({ resultJson: JSON.stringify([{ batch_id: "batch-1", item: { external_id: "article-1", variants: [{ id: "linkedin-short", name: "LinkedIn short", channel: "linkedin", body: "Useful professional post" }] }, assessment: { status: "admissible" }, receipt: { id: "receipt-1" } }]) });
    mocks.list.mockResolvedValue({ resultJson: "[]" });
    mocks.reconcile.mockResolvedValue({ resultJson: "[]" });
    mocks.promote.mockResolvedValue({ resultJson: JSON.stringify([{ draft: { id: "draft-1", status: "requested" } }]) });
    vi.stubGlobal("crypto", { randomUUID: () => "op-1" });
  });

  it("resumes promotion from a listed receipt without source re-entry and re-acknowledges changed variants", async () => {
    const receipt = { batch_id: "batch-2", item: { external_id: "article-2", variants: [{ id: "linkedin-short", name: "LinkedIn short", channel: "linkedin", body: "Short" }, { id: "x-short", name: "X short", channel: "x", body: "Short" }] }, assessment: { status: "admissible" }, receipt: { id: "receipt-2" } };
    mocks.list.mockResolvedValue({ resultJson: JSON.stringify([receipt]) });
    renderIntake();
    fireEvent.change(screen.getByLabelText(strings.pages.intake.batch), { target: { value: "batch-2" } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.intake.load }));
    await screen.findByRole("option", { name: /article-2/ });
    fireEvent.change(screen.getByLabelText(strings.pages.intake.postType), { target: { value: "dev-log" } });
    fireEvent.click(screen.getByRole("checkbox", { name: /LinkedIn short/ }));
    const acknowledge = () => fireEvent.click(screen.getAllByRole("checkbox").at(-1)!);
    acknowledge();
    const promote = screen.getByRole("button", { name: strings.pages.intake.promote });
    expect(promote).toBeEnabled();
    fireEvent.click(screen.getByRole("checkbox", { name: /X short/ }));
    expect(promote).toBeDisabled();
    acknowledge();
    expect(promote).toBeEnabled();
    fireEvent.click(promote);
    await waitFor(() => expect(mocks.promote).toHaveBeenCalledWith(expect.objectContaining({ batchId: "batch-2", externalId: "article-2", expectedReceiptId: "receipt-2", variantIds: ["linkedin-short", "x-short"] })));
  });

  it("rejects malformed intake item shapes before calling a provider", async () => {
    renderIntake();
    fireEvent.change(screen.getByLabelText(strings.pages.intake.items), { target: { value: '["text",null]' } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.intake.preview }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Article items must be a JSON array of objects.");
    expect(mocks.preview).not.toHaveBeenCalled();
  });

  it("does not offer promotion for rejected receipts", async () => {
    mocks.list.mockResolvedValue({ resultJson: JSON.stringify([{ batch_id: "batch-rejected", item: { external_id: "rejected-1", variants: [{ id: "v1", name: "Variant", channel: "linkedin", body: "Text" }] }, assessment: { status: "rejected" }, receipt: { id: "receipt-rejected" } }]) });
    renderIntake();
    fireEvent.change(screen.getByLabelText(strings.pages.intake.batch), { target: { value: "batch-rejected" } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.intake.load }));
    await screen.findByRole("option", { name: /rejected-1/ });
    expect(screen.getByRole("button", { name: strings.pages.intake.promote })).toBeDisabled();
  });

  it("promotes source-only content only after channel and exact-receipt acknowledgment", async () => {
    mocks.list.mockResolvedValue({ resultJson: JSON.stringify([{ batch_id: "batch-source", item: { external_id: "source-1" }, assessment: { status: "admissible" }, receipt: { id: "receipt-source" } }]) });
    renderIntake();
    fireEvent.change(screen.getByLabelText(strings.pages.intake.batch), { target: { value: "batch-source" } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.intake.load }));
    await screen.findByRole("option", { name: /source-1/ });
    fireEvent.change(screen.getByLabelText(strings.pages.intake.postType), { target: { value: "dev-log" } });
    const promote = screen.getByRole("button", { name: strings.pages.intake.promote });
    expect(promote).toBeDisabled();
    fireEvent.change(screen.getByLabelText(strings.pages.intake.channel), { target: { value: "linkedin" } });
    fireEvent.click(screen.getAllByRole("checkbox").at(-1)!);
    expect(promote).toBeEnabled();
    fireEvent.click(promote);
    await waitFor(() => expect(mocks.promote).toHaveBeenCalledWith(expect.objectContaining({ batchId: "batch-source", externalId: "source-1", expectedReceiptId: "receipt-source", variantIds: [], channel: "linkedin" })));
  });

  it("requires a matching preview and exact-source acknowledgment before admission", async () => {
    renderIntake();
    const admit = screen.getByRole("button", { name: strings.pages.intake.admit });
    expect(admit).toBeDisabled();
    fireEvent.change(screen.getByLabelText(strings.pages.intake.artifact), { target: { value: "source-1" } });
    fireEvent.change(screen.getByLabelText(strings.pages.intake.revision), { target: { value: "rev-1" } });
    fireEvent.change(screen.getByLabelText(strings.pages.intake.producer), { target: { value: "fixture" } });
    fireEvent.change(screen.getByLabelText(strings.pages.intake.intent), { target: { value: "adapt" } });
    fireEvent.change(screen.getByLabelText(strings.pages.intake.items), { target: { value: '[{"external_id":"article-1","kind":"article","title":"Title","body":"Text"}]' } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.intake.preview }));
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledOnce());
    expect(admit).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox"));
    expect(admit).toBeEnabled();
    fireEvent.change(screen.getByLabelText(strings.pages.intake.items), { target: { value: '[{"external_id":"article-1","body":"Changed"}]' } });
    expect(admit).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: strings.pages.intake.preview }));
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    fireEvent.click(admit);
    await waitFor(() => expect(mocks.admit).toHaveBeenCalledWith(expect.objectContaining({ idempotencyKey: "op-1", source: expect.objectContaining({ revisionId: "rev-1" }) })));
    expect(await screen.findByDisplayValue("batch-1")).toBeInTheDocument();
    expect(screen.getByText(/"batch_id"/)).toHaveTextContent("article-1");
    fireEvent.change(screen.getByLabelText(strings.pages.intake.postType), { target: { value: "dev-log" } });
    fireEvent.click(screen.getByRole("checkbox", { name: /LinkedIn short/ }));
    fireEvent.click(screen.getAllByRole("checkbox").at(-1)!);
    fireEvent.click(screen.getByRole("button", { name: strings.pages.intake.promote }));
    await waitFor(() => expect(mocks.promote).toHaveBeenCalledWith(expect.objectContaining({ batchId: "batch-1", externalId: "article-1", expectedReceiptId: "receipt-1", variantIds: ["linkedin-short"], postTypeId: "dev-log" })));
  });
});
