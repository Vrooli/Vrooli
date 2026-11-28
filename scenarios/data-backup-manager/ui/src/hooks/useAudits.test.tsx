import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../api/audits", () => ({
  listAudits: vi.fn(),
  getAudit: vi.fn(),
  runSnapshotAudit: vi.fn(),
  isTerminalAudit: vi.fn((status: number) => status === 1),
}));

import * as api from "../api/audits";
import { useAudits } from "./useAudits";

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    {children}
  </QueryClientProvider>
);

beforeEach(() => vi.clearAllMocks());

describe("useAudits", () => {
  it("loads audits for a target", async () => {
    vi.mocked(api.listAudits).mockResolvedValue([]);
    const { result } = renderHook(() => useAudits("target-1"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.listAudits).toHaveBeenCalledWith("target-1");
  });
});
