import { describe, expect, it, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import {
  buildAgentRunUrl,
  buildAgentTaskUrl,
  buildAgentProfileUrl,
  buildSkillUrl,
  useAgentRunUrl,
  useAgentTaskUrl,
  useAgentProfileUrl,
  useSkillUrl,
} from "./external-links";

vi.mock("../services", () => ({
  embeddedService: {
    getExternalUrl: vi.fn(),
  },
}));

import { embeddedService } from "../services";

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe("external-links pure builders", () => {
  describe("buildAgentRunUrl", () => {
    it("composes the URL when both base and runId are provided", () => {
      expect(buildAgentRunUrl("https://agent.test", "abc-123")).toBe(
        "https://agent.test/runs/abc-123",
      );
    });

    it("returns null when runId is missing", () => {
      expect(buildAgentRunUrl("https://agent.test", null)).toBeNull();
      expect(buildAgentRunUrl("https://agent.test", undefined)).toBeNull();
      expect(buildAgentRunUrl("https://agent.test", "")).toBeNull();
    });

    it("returns null when base is missing", () => {
      expect(buildAgentRunUrl(null, "abc-123")).toBeNull();
      expect(buildAgentRunUrl(undefined, "abc-123")).toBeNull();
      expect(buildAgentRunUrl("", "abc-123")).toBeNull();
    });

    it("trims a single trailing slash on the base", () => {
      expect(buildAgentRunUrl("https://agent.test/", "abc")).toBe("https://agent.test/runs/abc");
    });

    it("URL-encodes the runId", () => {
      expect(buildAgentRunUrl("https://agent.test", "run/with/slashes")).toBe(
        "https://agent.test/runs/run%2Fwith%2Fslashes",
      );
    });
  });

  describe("buildAgentTaskUrl", () => {
    it("deep-links to a selected task when base and taskId are provided", () => {
      expect(buildAgentTaskUrl("https://agent.test", "task-123")).toBe(
        "https://agent.test/tasks?taskId=task-123",
      );
    });

    it("returns null when taskId is missing", () => {
      expect(buildAgentTaskUrl("https://agent.test", null)).toBeNull();
      expect(buildAgentTaskUrl("https://agent.test", "")).toBeNull();
    });

    it("returns null without a base", () => {
      expect(buildAgentTaskUrl(null, "task-123")).toBeNull();
    });

    it("URL-encodes the taskId", () => {
      expect(buildAgentTaskUrl("https://agent.test", "task/with/slashes")).toBe(
        "https://agent.test/tasks?taskId=task%2Fwith%2Fslashes",
      );
    });
  });

  describe("buildAgentProfileUrl", () => {
    it("encodes the profileKey including its slash", () => {
      expect(buildAgentProfileUrl("https://agent.test", "swarm-manager/deep-work")).toBe(
        "https://agent.test/profiles?profileKey=swarm-manager%2Fdeep-work",
      );
    });

    it("returns null without a profileKey", () => {
      expect(buildAgentProfileUrl("https://agent.test", null)).toBeNull();
      expect(buildAgentProfileUrl("https://agent.test", "")).toBeNull();
    });

    it("returns null without a base", () => {
      expect(buildAgentProfileUrl(null, "swarm-manager/deep-work")).toBeNull();
    });
  });

  describe("buildSkillUrl", () => {
    it("encodes the skillId including its slash", () => {
      expect(buildSkillUrl("https://prompt.test", "swarm-manager/holistic-loop-investigate")).toBe(
        "https://prompt.test/skills/swarm-manager%2Fholistic-loop-investigate",
      );
    });

    it("returns null without a skillId", () => {
      expect(buildSkillUrl("https://prompt.test", undefined)).toBeNull();
    });

    it("returns null without a base", () => {
      expect(buildSkillUrl(null, "swarm-manager/holistic-loop-investigate")).toBeNull();
    });
  });
});

describe("external-links hooks", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    vi.mocked(embeddedService.getExternalUrl).mockReset();
  });

  it("useAgentRunUrl returns the resolved URL once the embedded-service URL loads", async () => {
    vi.mocked(embeddedService.getExternalUrl).mockImplementation(async (svc: string) =>
      svc === "agent-manager" ? "https://agent.test" : null,
    );
    queryClient.setQueryData(["embedded-service-url", "agent-manager"], "https://agent.test");

    const { result } = renderHook(() => useAgentRunUrl("run-1"), {
      wrapper: wrapper(queryClient),
    });
    expect(result.current).toBe("https://agent.test/runs/run-1");
  });

  it("useAgentRunUrl returns null while the embedded-service URL is still loading", () => {
    queryClient.setQueryData(["embedded-service-url", "agent-manager"], null);
    const { result } = renderHook(() => useAgentRunUrl("run-1"), {
      wrapper: wrapper(queryClient),
    });
    expect(result.current).toBeNull();
  });

  it("useAgentTaskUrl deep-links via ?taskId", () => {
    queryClient.setQueryData(["embedded-service-url", "agent-manager"], "https://agent.test");
    const { result } = renderHook(() => useAgentTaskUrl("task-1"), {
      wrapper: wrapper(queryClient),
    });
    expect(result.current).toBe("https://agent.test/tasks?taskId=task-1");
  });

  it("useSkillUrl resolves against the prompt-manager base", () => {
    queryClient.setQueryData(["embedded-service-url", "prompt-manager"], "https://prompt.test");
    const { result } = renderHook(() => useSkillUrl("swarm-manager/holistic-loop-investigate"), {
      wrapper: wrapper(queryClient),
    });
    expect(result.current).toBe(
      "https://prompt.test/skills/swarm-manager%2Fholistic-loop-investigate",
    );
  });

  it("useAgentProfileUrl deep-links via ?profileKey", () => {
    queryClient.setQueryData(["embedded-service-url", "agent-manager"], "https://agent.test");
    const { result } = renderHook(() => useAgentProfileUrl("swarm-manager/deep-work"), {
      wrapper: wrapper(queryClient),
    });
    expect(result.current).toBe(
      "https://agent.test/profiles?profileKey=swarm-manager%2Fdeep-work",
    );
  });
});
