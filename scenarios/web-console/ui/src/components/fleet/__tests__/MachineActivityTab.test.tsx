import { renderWithProviders as render } from "../../../test-utils";
import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import MachineActivityTab from "../MachineActivityTab";
import { getConfiguration, type Machine } from "../../../api/machines";

vi.mock("../../../api/machines", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../../api/machines")>()),
  getConfiguration: vi.fn(),
}));

const machine = { target: { id: "machine-1" } } as Machine;

describe("MachineActivityTab", () => {
  beforeEach(() => vi.mocked(getConfiguration).mockReset());

  it("shows an explicit empty state when configuration has no audit events", async () => {
    vi.mocked(getConfiguration).mockResolvedValue({ targetId: "machine-1", questions: [], readiness: null, detail: null });
    render(<MachineActivityTab machine={machine} />);
    expect(await screen.findByText("machines.activityEmpty")).toBeInTheDocument();
  });

  it("renders audit events with a safe system actor fallback", async () => {
    vi.mocked(getConfiguration).mockResolvedValue({
      targetId: "machine-1",
      questions: [],
      readiness: null,
      detail: { auditEvents: [{ actor: "", action: "connected", detail: "ok" }] } as never,
    });
    render(<MachineActivityTab machine={machine} />);
    await waitFor(() => expect(screen.getByTestId("machine-activity")).toBeInTheDocument());
    expect(screen.getByText("machines.activityHeading")).toBeInTheDocument();
  });
});
