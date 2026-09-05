import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  useDeleteSSHKey,
  useGitignoreMove,
  useGroupingRules,
  useOpenRepo,
  useSaveCredential,
  useSaveGroupingRules,
} from "./hooks-settings";
import { queryKeys } from "./hooks-query-keys";
import { renderHookWithQueryClient } from "../test-utils";

const mockFetchCapabilities = vi.fn();
const mockFetchCredentials = vi.fn();
const mockSaveCredential = vi.fn();
const mockDeleteCredential = vi.fn();
const mockTestCredential = vi.fn();
const mockUpdateRemoteURL = vi.fn();
const mockFetchRepos = vi.fn();
const mockFetchActiveRepo = vi.fn();
const mockOpenRepo = vi.fn();
const mockCloneRepo = vi.fn();
const mockSetActiveRepo = vi.fn();
const mockRemoveRepo = vi.fn();
const mockFetchSSHKeys = vi.fn();
const mockGenerateSSHKey = vi.fn();
const mockGetSSHPublicKey = vi.fn();
const mockTestSSHConnection = vi.fn();
const mockDeleteSSHKey = vi.fn();
const mockFetchGroupingRules = vi.fn();
const mockSaveGroupingRules = vi.fn();
const mockFetchGitignoreHealth = vi.fn();
const mockMoveGitignoreEntry = vi.fn();

vi.mock("./api", () => ({
  fetchCapabilities: (...args: unknown[]) => mockFetchCapabilities(...args),
  fetchCredentials: (...args: unknown[]) => mockFetchCredentials(...args),
  saveCredential: (...args: unknown[]) => mockSaveCredential(...args),
  deleteCredential: (...args: unknown[]) => mockDeleteCredential(...args),
  testCredential: (...args: unknown[]) => mockTestCredential(...args),
  updateRemoteURL: (...args: unknown[]) => mockUpdateRemoteURL(...args),
  fetchRepos: (...args: unknown[]) => mockFetchRepos(...args),
  fetchActiveRepo: (...args: unknown[]) => mockFetchActiveRepo(...args),
  openRepo: (...args: unknown[]) => mockOpenRepo(...args),
  cloneRepo: (...args: unknown[]) => mockCloneRepo(...args),
  setActiveRepo: (...args: unknown[]) => mockSetActiveRepo(...args),
  removeRepo: (...args: unknown[]) => mockRemoveRepo(...args),
  fetchSSHKeys: (...args: unknown[]) => mockFetchSSHKeys(...args),
  generateSSHKey: (...args: unknown[]) => mockGenerateSSHKey(...args),
  getSSHPublicKey: (...args: unknown[]) => mockGetSSHPublicKey(...args),
  testSSHConnection: (...args: unknown[]) => mockTestSSHConnection(...args),
  deleteSSHKey: (...args: unknown[]) => mockDeleteSSHKey(...args),
  fetchGroupingRules: (...args: unknown[]) => mockFetchGroupingRules(...args),
  saveGroupingRules: (...args: unknown[]) => mockSaveGroupingRules(...args),
  fetchGitignoreHealth: (...args: unknown[]) => mockFetchGitignoreHealth(...args),
  moveGitignoreEntry: (...args: unknown[]) => mockMoveGitignoreEntry(...args),
}));

describe("settings hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSaveCredential.mockResolvedValue({ success: true, id: "cred-1" });
    mockOpenRepo.mockResolvedValue({ success: true });
    mockDeleteSSHKey.mockResolvedValue({ success: true });
    mockFetchGroupingRules.mockResolvedValue({ version: 1, rules: [] });
    mockSaveGroupingRules.mockResolvedValue({ version: 1, rules: [] });
    mockMoveGitignoreEntry.mockResolvedValue({ success: true });
  });

  it("does not fetch grouping rules without an active repo", () => {
    renderHookWithQueryClient(() => useGroupingRules(null));

    expect(mockFetchGroupingRules).not.toHaveBeenCalled();
  });

  it("saves credentials with repo context and invalidates credential queries", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useSaveCredential("repo-1"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({
        remote: "origin",
        url: "https://github.com/example/repo.git",
        username: "octo",
        token: "secret",
      });
    });

    await waitFor(() => {
      expect(mockSaveCredential).toHaveBeenCalledWith({
        remote: "origin",
        url: "https://github.com/example/repo.git",
        username: "octo",
        token: "secret",
      }, "repo-1");
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.credentials("repo-1"),
      });
    });
  });

  it("invalidates repo registry queries after opening a repo", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useOpenRepo());
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({ path: "/tmp/repo" });
    });

    await waitFor(() => {
      expect(mockOpenRepo).toHaveBeenCalledWith({ path: "/tmp/repo" });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.repos });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.activeRepo });
    });
  });

  it("invalidates SSH key queries after deleting a key", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useDeleteSSHKey());
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({ key_path: "/home/user/.ssh/deploy-key" });
    });

    await waitFor(() => {
      expect(mockDeleteSSHKey).toHaveBeenCalledWith({ key_path: "/home/user/.ssh/deploy-key" });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.sshKeys });
    });
  });

  it("saves grouping rules with repo scope and invalidates that repo's rules", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useSaveGroupingRules("repo-2"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const config = {
      enabled: true,
      rules: [
        { id: "docs", label: "Docs", prefixes: ["docs/"], mode: "prefix" },
      ],
    };

    await act(async () => {
      await result.current.mutateAsync(config);
    });

    await waitFor(() => {
      expect(mockSaveGroupingRules).toHaveBeenCalledWith(config, "repo-2");
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.groupingRules("repo-2"),
      });
    });
  });

  it("refreshes gitignore health and status after moving an entry", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useGitignoreMove("repo-3"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({
        line: 3,
        pattern: "coverage/",
        group_dir: "frontend",
        target_pattern: "frontend/coverage/",
      });
    });

    await waitFor(() => {
      expect(mockMoveGitignoreEntry).toHaveBeenCalledWith({
        line: 3,
        pattern: "coverage/",
        group_dir: "frontend",
        target_pattern: "frontend/coverage/",
      }, "repo-3");
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.gitignoreHealth("repo-3"),
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.repoStatus("repo-3"),
      });
    });
  });
});
