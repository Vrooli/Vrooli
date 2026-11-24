import { fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { renderWithProviders } from "../../test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { InstallPosture } from "./InstallPosture";
import { readInstallPosture, writeUpdateControl } from "../../api/installPosture";

vi.mock("../../api/installPosture", () => ({ readInstallPosture: vi.fn(), writeUpdateControl: vi.fn() }));

describe("InstallPosture", () => {
  beforeEach(() => vi.clearAllMocks());

  it("reads enrollment-owned session mode and writes update control through its field contract", async () => {
    vi.mocked(readInstallPosture).mockResolvedValue({ updateControl: "observe", updateControlRevision: "revision-7", trustPosture: "hosted", sessionMode: "shared", sessionModeRevision: "enrollment-r3" });
    vi.mocked(writeUpdateControl).mockResolvedValue({ id: "receipt-1" } as never);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderWithProviders(<InstallPosture />, { queryClient: client });
    const select = await screen.findByRole("combobox", { name: "Host update control" });
    expect(screen.getByTestId("install-session-mode")).toHaveTextContent("shared");
    expect(screen.getByTestId("install-trust-posture")).toHaveTextContent("hosted");
    fireEvent.change(select, { target: { value: "guard" } });
    fireEvent.click(screen.getByRole("button", { name: "Save update control" }));
    await waitFor(() => expect(writeUpdateControl).toHaveBeenCalledWith("guard", "revision-7", expect.any(String)));
  });
});
