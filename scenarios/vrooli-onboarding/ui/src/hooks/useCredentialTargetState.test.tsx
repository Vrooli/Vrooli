import { act, renderHook } from "../test-utils";
import { describe, expect, it } from "vitest";
import { useCredentialTargetState } from "./useCredentialTargetState";

describe("credential target drafts", () => {
  it("keeps unsaved secret text through a wizard route unmount without browser storage", () => {
    const target = `local-${crypto.randomUUID()}`;
    const firstRoute = renderHook(() => useCredentialTargetState(target));

    act(() => firstRoute.result.current.updateTargetState(target, (state) => ({
      ...state,
      values: { "mail/api_key": "synthetic-secret-for-test" },
    })));
    expect(firstRoute.result.current.state.values["mail/api_key"]).toBe("synthetic-secret-for-test");
    expect(window.localStorage.length).toBe(0);
    expect(window.sessionStorage.length).toBe(0);
    firstRoute.unmount();

    const resumedRoute = renderHook(() => useCredentialTargetState(target));
    expect(resumedRoute.result.current.state.values["mail/api_key"]).toBe("synthetic-secret-for-test");
    resumedRoute.unmount();
  });

  it("keeps each target's unsaved value separate", () => {
    const localTarget = `local-${crypto.randomUUID()}`;
    const remoteTarget = `remote-${crypto.randomUUID()}`;
    const local = renderHook(() => useCredentialTargetState(localTarget));
    const remote = renderHook(() => useCredentialTargetState(remoteTarget));

    act(() => local.result.current.updateTargetState(localTarget, (state) => ({
      ...state,
      values: { "mail/api_key": "local-synthetic-value" },
    })));

    expect(local.result.current.state.values["mail/api_key"]).toBe("local-synthetic-value");
    expect(remote.result.current.state.values["mail/api_key"]).toBeUndefined();
    local.unmount();
    remote.unmount();
  });
});
