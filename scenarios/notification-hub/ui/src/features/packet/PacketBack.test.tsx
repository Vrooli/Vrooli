import { useState } from "react";
import { beforeEach, describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import userEvent from "@testing-library/user-event";

import { setLocale } from "../../i18n";
import { en } from "../../i18n/catalog";
import { renderWithProviders } from "../../test-utils";
import { PacketBack } from "./PacketBack";
import { usePacketOrigin } from "./usePacketOrigin";

function Where() {
  const location = useLocation();
  return <output data-testid="where">{`${location.pathname}${location.search}|${JSON.stringify(location.state)}`}</output>;
}

function Origin({ state }: { state?: string }) {
  const { returnTo, section } = usePacketOrigin(state === undefined ? undefined : { state });
  return <output data-testid="origin">{`${returnTo}|${section}`}</output>;
}

function renderIn(entry: string | { pathname: string; state: unknown }, children: React.ReactNode) {
  return renderWithProviders(
    <MemoryRouter initialEntries={[entry as string]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      {children}
      <Routes>
        <Route path="*" element={<Where />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

const origin = () => screen.getByTestId("origin").textContent;

describe("PacketBack", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("goes where it is told, names that list, and asks it to restore this decision's row", async () => {
    renderIn("/asks/ask-1", <PacketBack returnTo="/asks" askId="ask-1" />);
    await userEvent.click(screen.getByRole("link", { name: en.layout.nav.asks }));
    expect(screen.getByTestId("where")).toHaveTextContent('/asks|{"restoreAskId":"ask-1"}');
  });

  it("is named History when it returns to History, search included", async () => {
    renderIn("/asks/ask-1", <PacketBack returnTo="/history?q=goal" askId="ask-1" />);
    await userEvent.click(screen.getByRole("link", { name: en.layout.nav.history }));
    expect(screen.getByTestId("where")).toHaveTextContent('/history?q=goal|{"restoreAskId":"ask-1"}');
  });

  it("renders compact mode as an accessible arrow without a text label", () => {
    renderIn("/asks/ask-1", <PacketBack returnTo="/asks" askId="ask-1" compact />);
    const link = screen.getByRole("link", { name: en.layout.nav.asks });
    expect(link).toHaveAttribute("title", en.layout.nav.asks);
    expect(link).toHaveClass("min-w-11", "min-h-11");
    expect(link.textContent).toBe("");
  });
});

describe("where a decision sits", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("follows the list it was opened from, search included", () => {
    renderIn({ pathname: "/asks/ask-1", state: { from: "/history?q=goal" } }, <Origin state="pending" />);
    expect(origin()).toBe("/history?q=goal|history");
  });

  it("follows Needs you when it was opened from there, even if the decision has closed", () => {
    renderIn({ pathname: "/asks/ask-1", state: { from: "/asks?urgency=high" } }, <Origin state="answered" />);
    expect(origin()).toBe("/asks?urgency=high|asks");
  });

  it("ignores a return address that is not one of the Hub's lists", () => {
    renderIn({ pathname: "/asks/ask-1", state: { from: "https://evil.example/asks" } }, <Origin state="pending" />);
    expect(origin()).toBe("/asks|asks");
  });

  it("belongs to Needs you when it is waiting and reached directly (a push link)", () => {
    renderIn("/asks/ask-1", <Origin state="pending" />);
    expect(origin()).toBe("/asks|asks");
  });

  it("treats an escalated decision as waiting", () => {
    renderIn("/asks/ask-1", <Origin state="escalated" />);
    expect(origin()).toBe("/asks|asks");
  });

  it.each(["answered", "defaulted", "expired", "withdrawn", "dismissed"])("belongs to History when it is %s and reached directly", (state) => {
    renderIn("/asks/ask-1", <Origin state={state} />);
    expect(origin()).toBe("/history|history");
  });

  it("claims nothing before the decision is known", () => {
    renderIn("/asks/ask-1", <Origin />);
    expect(origin()).toBe("/asks|null");
  });

  it("keeps the section it opened in when the decision closes while it is read", async () => {
    function Closing() {
      const [state, setState] = useState("pending");
      return (
        <>
          <button type="button" onClick={() => setState("answered")}>answer</button>
          <Origin state={state} />
        </>
      );
    }
    renderIn("/asks/ask-1", <Closing />);
    await userEvent.click(screen.getByRole("button", { name: "answer" }));
    expect(origin()).toBe("/asks|asks");
  });
});
