import { afterEach, describe, expect, it, vi } from "vitest";

const auth = vi.hoisted(() => ({
  completeWebAuthCallback: vi.fn(),
  getAccessToken: vi.fn(),
  startSignIn: vi.fn(),
  signOut: vi.fn(),
}));

vi.mock("../lib/auth", () => auth);

import { useAuthStore } from "./authStore";

function clearDesktop(): void {
  delete (window as Window & { desktop?: unknown }).desktop;
}

afterEach(() => {
  clearDesktop();
  vi.clearAllMocks();
  useAuthStore.setState({ accessToken: null, loading: true, error: null });
});

describe("useAuthStore", () => {
  it("initializes from the current web access token", async () => {
    auth.completeWebAuthCallback.mockResolvedValue(false);
    auth.getAccessToken.mockResolvedValue("web-token");

    await useAuthStore.getState().initialize();

    expect(useAuthStore.getState()).toMatchObject({ accessToken: "web-token", loading: false, error: null });
  });

  it("syncs desktop auth changes and reports session expiry", async () => {
    let listener: ((event: { event: string }) => void) | undefined;
    Object.defineProperty(window, "desktop", {
      configurable: true,
      value: { auth: { onAuthChanged: (next: typeof listener) => { listener = next; } } },
    });
    auth.completeWebAuthCallback.mockResolvedValue(false);
    auth.getAccessToken.mockResolvedValueOnce("first-token").mockResolvedValueOnce(null);

    await useAuthStore.getState().initialize();
    await listener?.({ event: "session-expired" });

    expect(useAuthStore.getState()).toMatchObject({ accessToken: null, loading: false, error: "Subscription session expired" });
  });

  it("completes sign-in and clears the loading state", async () => {
    auth.startSignIn.mockResolvedValue(undefined);

    await useAuthStore.getState().signIn();

    expect(auth.startSignIn).toHaveBeenCalledOnce();
    expect(useAuthStore.getState()).toMatchObject({ loading: false, error: null });
  });

  it("surfaces sign-in and sign-out failures without losing the session state", async () => {
    auth.startSignIn.mockRejectedValue(new Error("provider unavailable"));
    await useAuthStore.getState().signIn();
    expect(useAuthStore.getState()).toMatchObject({ loading: false, error: "provider unavailable" });

    auth.signOut.mockRejectedValue(new Error("sign-out failed"));
    useAuthStore.setState({ accessToken: "keep-me" });
    await useAuthStore.getState().signOut();
    expect(useAuthStore.getState()).toMatchObject({ accessToken: "keep-me", loading: false, error: "sign-out failed" });
  });

  it("signs out successfully and removes the access token", async () => {
    auth.signOut.mockResolvedValue(undefined);
    useAuthStore.setState({ accessToken: "active" });

    await useAuthStore.getState().signOut();

    expect(auth.signOut).toHaveBeenCalledOnce();
    expect(useAuthStore.getState()).toMatchObject({ accessToken: null, loading: false, error: null });
  });

  it("converts callback failures into an authentication error", async () => {
    auth.completeWebAuthCallback.mockRejectedValue(new Error("bad callback"));

    await useAuthStore.getState().initialize();

    expect(useAuthStore.getState()).toMatchObject({ loading: false, error: "bad callback" });
  });
});
