import { afterEach, expect, it, vi } from "vitest";
import { startSignIn } from "./auth";
import { LANDING_PAGE_URL } from "../shared/upgradeDestination";

afterEach(() => {
  vi.unstubAllGlobals();
  sessionStorage.removeItem("vrooli.web.auth-state");
});

it("identifies Aquila to sign-in while retaining the callback and technical state key", async () => {
  const assign = vi.fn();
  const callback = "https://console.example.test/?workspace=current";
  vi.stubGlobal("window", { location: { href: callback, assign } });

  await startSignIn();

  expect(assign).toHaveBeenCalledTimes(1);
  const destination = new URL(String(assign.mock.calls[0]?.[0]));
  expect(destination.origin).toBe(new URL(LANDING_PAGE_URL).origin);
  expect(destination.pathname).toBe("/auth/login");
  expect(destination.searchParams.get("app")).toBe("Aquila");
  expect(destination.searchParams.get("redirect_uri")).toBe(callback);
  const state = sessionStorage.getItem("vrooli.web.auth-state");
  expect(state).toBeTruthy();
  expect(destination.searchParams.get("state")).toBe(state);
});
