import { renderWithQueryClient, screen } from "../../test-utils";
import { PendingOperatorDecisions } from "./PendingOperatorDecisions";
import { listPendingOperatorDecisions } from "../../api/operatorDecisions";

vi.mock("../../api/operatorDecisions", () => ({
  listPendingOperatorDecisions: vi.fn(),
}));

describe("PendingOperatorDecisions", () => {
  it("shows only safe owner reference metadata and no decision controls", async () => {
    vi.mocked(listPendingOperatorDecisions).mockResolvedValue([{
      permissionId: "onboarding/operator-decision/ref",
      revision: "3",
      decisionRef: "secret-reference",
      ownerId: "operator-1",
      source: "treasury",
      sourceEventType: "treasury.refund.decision_requested.v1",
      state: "pending",
      createdAt: "2026-11-20T10:00:00Z",
      updatedAt: "2026-11-20T10:01:00Z",
      deadline: "",
      urgency: "high",
    }]);
    renderWithQueryClient(<PendingOperatorDecisions />);
    expect(await screen.findByText(/treasury · high urgency/i)).toBeInTheDocument();
    expect(screen.getByText(/revision 3/i)).toBeInTheDocument();
    expect(screen.queryByText("secret-reference")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /approve|deny|resolve/i })).not.toBeInTheDocument();
  });
});
