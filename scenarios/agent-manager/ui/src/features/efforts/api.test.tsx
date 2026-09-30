import { create } from "@bufbuild/protobuf";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { PropsWithChildren } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { EffortBoardSchema } from "@vrooli/proto-types/agent-manager/v1/domain/effort_pb";
import { effortBoardClient, useEffortBoard } from "./api";

afterEach(() => vi.restoreAllMocks());
function wrapper() {
  const client = new QueryClient();
  return ({ children }: PropsWithChildren) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

test("owner cursor selects distinct evidence pages without discovery or mutation", async () => {
  const read = vi.spyOn(effortBoardClient, "getEffortBoard")
    .mockResolvedValueOnce(create(EffortBoardSchema, { nextPageToken: "opaque", rows: [{ enrollment: { effortRef: "one" } }] }))
    .mockResolvedValueOnce(create(EffortBoardSchema, { rows: [{ enrollment: { effortRef: "two" } }] }));
  const discover = vi.spyOn(effortBoardClient, "reconcileEffortDiscovery");
  const enroll = vi.spyOn(effortBoardClient, "enrollEffort");
  const { result, rerender } = renderHook(({ cursor }) => useEffortBoard(cursor), { initialProps: { cursor: "" }, wrapper: wrapper() });
  await waitFor(() => expect(result.current.data?.nextPageToken).toBe("opaque"));
  expect(result.current.data?.rows[0].enrollment?.effortRef).toBe("one");
  rerender({ cursor: "opaque" });
  await waitFor(() => expect(result.current.data?.rows[0].enrollment?.effortRef).toBe("two"));
  expect(read).toHaveBeenNthCalledWith(2, { pageSize: 25, pageToken: "opaque" }, { signal: expect.any(AbortSignal), timeoutMs: 30_000 });
  expect(discover).not.toHaveBeenCalled();
  expect(enroll).not.toHaveBeenCalled();
});

test("an exact effort link asks the owner for that identity outside the current page", async () => {
  const read = vi.spyOn(effortBoardClient, "getEffortBoard")
    .mockResolvedValue(create(EffortBoardSchema, { rows: [{ enrollment: { effortRef: "effort:alpha/beta?x=1" } }] }));
  const { result } = renderHook(() => useEffortBoard("", "effort:alpha/beta?x=1"), { wrapper: wrapper() });
  await waitFor(() => expect(result.current.isSuccess).toBe(true));
  expect(read).toHaveBeenCalledWith({ pageSize: 25, pageToken: "", effortRef: "effort:alpha/beta?x=1" }, { signal: expect.any(AbortSignal), timeoutMs: 30_000 });
});

test("obsolete observations are cancelled and owner failure remains an error", async () => {
  const signals: AbortSignal[] = [];
  const read = vi.spyOn(effortBoardClient, "getEffortBoard").mockImplementationOnce((_request, options) => {
    signals.push(options!.signal!); return new Promise(() => undefined);
  }).mockRejectedValueOnce(new Error("owner unavailable"));
  const { result, rerender } = renderHook(({ cursor }) => useEffortBoard(cursor), { initialProps: { cursor: "" }, wrapper: wrapper() });
  await waitFor(() => expect(signals).toHaveLength(1));
  rerender({ cursor: "next" });
  await waitFor(() => expect(result.current.isError).toBe(true));
  expect(signals[0].aborted).toBe(true);
  expect(read).toHaveBeenCalledTimes(2);
  expect(result.current.data).toBeUndefined();
});
