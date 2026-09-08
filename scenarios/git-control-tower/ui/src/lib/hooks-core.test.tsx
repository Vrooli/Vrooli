import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useStageFiles, usePendingWorkspaceChanges, useUnstageFiles, useCommit } from "./hooks-core";
import { queryKeys } from "./hooks-query-keys";
import { renderHookWithQueryClient } from "../test-utils";

const mockStageFiles = vi.fn();
const mockUnstageFiles = vi.fn();
const mockCommit = vi.fn();

vi.mock("./api", () => ({
  stageFiles: (...args: unknown[]) => mockStageFiles(...args),
  unstageFiles: (...args: unknown[]) => mockUnstageFiles(...args),
  createCommit: (...args: unknown[]) => mockCommit(...args),
}));

describe("core repo hooks", () => {
  beforeEach(() => {
    mockStageFiles.mockResolvedValue({ success: true, staged: ["src/App.tsx"], timestamp: "t" });
  });

  it("useStageFiles forwards repo context and invalidates repo status", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useStageFiles("repo-1"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({ paths: ["src/App.tsx"] });
    });

    await waitFor(() => {
      expect(mockStageFiles).toHaveBeenCalledWith({ paths: ["src/App.tsx"] }, "repo-1");
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.repoStatus("repo-1") });
    });
  });
});

it("queues index writes while preserving later optimistic changes when an earlier request fails", async () => {
  let rejectFirst: (error: Error) => void = () => { throw new Error("first request not started"); };
  let resolveSecond: (value: { success: boolean }) => void = () => { throw new Error("second request not started"); };
  mockStageFiles.mockReset();
  mockStageFiles.mockImplementationOnce(() => new Promise((_resolve, reject) => { rejectFirst = reject; }))
    .mockImplementationOnce(() => new Promise(resolve => { resolveSecond = resolve; }));
  const { result, queryClient } = renderHookWithQueryClient(() => ({ stage: useStageFiles("queue-repo"), pending: usePendingWorkspaceChanges("queue-repo") }));
  const key = queryKeys.repoStatus("queue-repo");
  queryClient.setQueryData(key, {
    repo_dir: "/repo", branch: {head:"main"}, author:{}, timestamp:"now",
    files: { staged:[], unstaged:["a.ts", "b.ts"], untracked:[], conflicts:[], statuses:{} },
    summary: {staged:0, unstaged:2, untracked:0, conflicts:0},
  });
  const invalidate = vi.spyOn(queryClient, "invalidateQueries");
  act(() => { result.current.stage.mutate({paths:["a.ts"]}); });
  await waitFor(() => expect(mockStageFiles).toHaveBeenCalledTimes(1));
  act(() => { result.current.stage.mutate({paths:["b.ts"]}); });
  await waitFor(() => expect(queryClient.getQueryData<{files:{staged:string[]}}>(key)?.files.staged).toEqual(["a.ts","b.ts"]));
  expect(mockStageFiles).toHaveBeenCalledTimes(1);
  expect([...result.current.pending.paths]).toEqual(["a.ts", "b.ts"]);
  act(() => rejectFirst(new Error("stage rejected")));
  await waitFor(() => expect(mockStageFiles).toHaveBeenCalledTimes(2));
  expect(queryClient.getQueryData<{files:{staged:string[]}}>(key)?.files.staged).toEqual(["b.ts"]);
  expect(invalidate).not.toHaveBeenCalled();
  await waitFor(() => expect(result.current.stage.error?.message).toBe("stage rejected"));
  act(() => result.current.stage.reset());
  expect(result.current.pending.paths.has("b.ts")).toBe(true);
  act(() => resolveSecond({success:true}));
  await waitFor(() => expect(invalidate).toHaveBeenCalledTimes(1));
});


it("treats an explicit server refusal as an error and restores the pending file", async () => {
 mockStageFiles.mockReset().mockResolvedValue({success:false, errors:["ignored path"]});
 const {result, queryClient} = renderHookWithQueryClient(() => useStageFiles("refused"));
 const key=queryKeys.repoStatus("refused");
 queryClient.setQueryData(key, {repo_dir:"/repo", branch:{head:"main"}, author:{}, timestamp:"now", files:{staged:[],unstaged:[],untracked:["ignored"],conflicts:[]}, summary:{staged:0,unstaged:0,untracked:1,conflicts:0}});
 await act(async () => { await expect(result.current.mutateAsync({paths:["ignored"]})).rejects.toThrow("ignored path"); });
 expect(queryClient.getQueryData<{files:{untracked:string[];staged:string[]}}>(key)?.files).toMatchObject({untracked:["ignored"],staged:[]});
});


it("queues unstage behind stage and restores a newly added file to untracked", async () => {
 let finish: (value: {success:boolean}) => void = () => {throw new Error("not started");};
 mockStageFiles.mockReset().mockImplementation(() => new Promise(resolve => {finish=resolve;}));
 mockUnstageFiles.mockReset().mockResolvedValue({success:true});
 const {result, queryClient}=renderHookWithQueryClient(()=>({stage:useStageFiles("same"),unstage:useUnstageFiles("same")}));
 const key=queryKeys.repoStatus("same");
 queryClient.setQueryData(key,{repo_dir:"/repo",branch:{head:"main"},author:{},timestamp:"now",files:{staged:[],unstaged:[],untracked:["new.ts"],conflicts:[]},summary:{staged:0,unstaged:0,untracked:1,conflicts:0}});
 act(()=>result.current.stage.mutate({paths:["new.ts"]}));
 await waitFor(()=>expect(mockStageFiles).toHaveBeenCalledTimes(1));
 act(()=>result.current.unstage.mutate({paths:["new.ts"]}));
 await waitFor(()=>expect(queryClient.getQueryData<{files:{untracked:string[]}}>(key)?.files.untracked).toEqual(["new.ts"]));
 expect(mockUnstageFiles).not.toHaveBeenCalled();
 act(()=>finish({success:true}));
 await waitFor(()=>expect(mockUnstageFiles).toHaveBeenCalledTimes(1));
});

it("refreshes commit history immediately after confirmed success", async () => {
 mockCommit.mockResolvedValue({success:true});
 const {result,queryClient}=renderHookWithQueryClient(()=>useCommit("committed"));
 const invalidate=vi.spyOn(queryClient,"invalidateQueries");
 await act(async()=>{await result.current.mutateAsync({message:"source change"});});
 expect(invalidate).toHaveBeenCalledWith({queryKey:["repo","history","committed"]});
 invalidate.mockClear();mockCommit.mockResolvedValue({success:false});
 await act(async()=>{await result.current.mutateAsync({message:"refused"});});
 expect(invalidate).not.toHaveBeenCalled();
});
