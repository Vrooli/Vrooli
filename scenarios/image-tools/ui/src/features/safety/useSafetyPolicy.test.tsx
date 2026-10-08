import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, renderHook, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  DeploymentTier,
  SafetyPolicySchema,
} from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

const mocks = vi.hoisted(() => ({ getPolicy: vi.fn() }));
vi.mock("../../api/safety", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/safety")>();
  return { ...actual, getPolicy: mocks.getPolicy };
});

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { SAFETY_POLICY_QUERY_KEY, useSafetyPolicy } from "./useSafetyPolicy";

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("useSafetyPolicy", () => {
  it("exposes a stable query key", () => {
    expect(SAFETY_POLICY_QUERY_KEY).toEqual(["safety-policy"]);
  });

  it("fetches and returns the resolved policy", async () => {
    const policy = create(SafetyPolicySchema, { tier: DeploymentTier.LOCAL });
    mocks.getPolicy.mockResolvedValue(policy);

    const { result } = renderHook(() => useSafetyPolicy(), { wrapper });

    await waitFor(() => expect(result.current.data).toBe(policy));
    expect(mocks.getPolicy).toHaveBeenCalledTimes(1);
  });
});
