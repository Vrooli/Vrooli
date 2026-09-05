import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ConsoleApiError } from "../../api/console";
import { session } from "../../api/session";
import { renderWithProviders } from "../../test-utils";
import { SessionProvider, useSession } from "./SessionProvider";

function Probe() {
  const { session: current, withSession, signOut } = useSession();
  return (
    <div>
      <span data-testid="who">{current?.email ?? "anonymous"}</span>
      <button
        type="button"
        data-testid="write"
        onClick={() => {
          let attempts = 0;
          void withSession(() => {
            attempts += 1;
            if (attempts === 1) return Promise.reject(new ConsoleApiError("unauthenticated", 401));
            return Promise.resolve("ok");
          }).then((value) => {
            document.title = `result:${value}:${attempts}`;
          });
        }}
      >
        write
      </button>
      <button type="button" data-testid="out" onClick={signOut}>
        out
      </button>
    </div>
  );
}

describe("SessionProvider", () => {
  beforeEach(() => {
    session.clear();
    document.title = "";
  });
  afterEach(() => vi.unstubAllGlobals());

  it("prompts for sign-in on 401, then retries the write once", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() => Promise.resolve(new Response(JSON.stringify({ token: "t", subject: "owner-1", email: "o@example.com" }), { status: 200, headers: { "Content-Type": "application/json" } }))),
    );
    renderWithProviders(
      <SessionProvider>
        <Probe />
      </SessionProvider>,
    );
    expect(screen.getByTestId("who")).toHaveTextContent("anonymous");
    fireEvent.click(screen.getByTestId("write"));
    const email = await screen.findByTestId("session-email");
    fireEvent.change(email, { target: { value: "o@example.com" } });
    fireEvent.change(screen.getByTestId("session-password"), { target: { value: "secret" } });
    fireEvent.click(screen.getByTestId("session-submit"));
    await waitFor(() => expect(document.title).toBe("result:ok:2"));
    expect(screen.getByTestId("who")).toHaveTextContent("o@example.com");
    expect(session.token()).toBe("t");
    fireEvent.click(screen.getByTestId("out"));
    await waitFor(() => expect(screen.getByTestId("who")).toHaveTextContent("anonymous"));
  });

  it("reports bad credentials without storing anything", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response("denied", { status: 401 }))));
    renderWithProviders(
      <SessionProvider>
        <Probe />
      </SessionProvider>,
    );
    fireEvent.click(screen.getByTestId("write"));
    fireEvent.change(await screen.findByTestId("session-email"), { target: { value: "o@example.com" } });
    fireEvent.change(screen.getByTestId("session-password"), { target: { value: "wrong" } });
    fireEvent.click(screen.getByTestId("session-submit"));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(session.get()).toBeNull();
  });
});
