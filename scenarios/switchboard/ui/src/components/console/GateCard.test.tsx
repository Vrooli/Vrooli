import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { makeGate, stubConsoleFetch } from "../../test-utils/consoleFixtures";
import { GateCard } from "./GateCard";

describe("GateCard", () => {
  afterEach(() => vi.unstubAllGlobals());

  // [REQ:SWBD-P1-005]
  it("lets the owner grant once and posts the answer", async () => {
    const { calls } = stubConsoleFetch({ "/api/v1/gates/gate-1/answer": makeGate({ status: "granted" }) });
    const onAnswered = vi.fn();
    renderWithProviders(<GateCard gate={makeGate()} onAnswered={onAnswered} />);
    expect(screen.getByTestId("capability-gate-withheld")).toHaveTextContent("calendar.write");
    expect(screen.getByTestId("capability-gate-expiry")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("capability-gate-grant"));
    await waitFor(() => expect(onAnswered).toHaveBeenCalled());
    expect(calls[0]?.init?.method).toBe("POST");
  });

  it("hides the controls from anyone who is not the owner", () => {
    stubConsoleFetch({});
    renderWithProviders(<GateCard gate={makeGate()} viewerIsOwner={false} />);
    expect(screen.getByTestId("capability-gate-permission-denied")).toBeInTheDocument();
    expect(screen.queryByTestId("capability-gate-grant")).not.toBeInTheDocument();
  });

  it("renders an expired gate as a fact with no actions", () => {
    stubConsoleFetch({});
    renderWithProviders(<GateCard gate={makeGate({ status: "expired", expires_at: new Date(Date.now() - 60_000).toISOString() })} />);
    expect(screen.queryByTestId("capability-gate-grant")).not.toBeInTheDocument();
    expect(screen.getByTestId("capability-gate")).toHaveAttribute("data-gate-status", "expired");
  });

  it("surfaces an answer failure inline", async () => {
    stubConsoleFetch({ "/api/v1/gates/gate-1/answer": new Response("nope", { status: 409 }) });
    renderWithProviders(<GateCard gate={makeGate()} />);
    fireEvent.click(screen.getByTestId("capability-gate-deny"));
    expect(await screen.findByRole("alert", { name: "" })).toBeInTheDocument();
  });
});
