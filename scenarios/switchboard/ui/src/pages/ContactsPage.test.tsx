import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { bodyOf, defaultRoutes, makeContact, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { ContactsPage } from "./ContactsPage";

const makeContactDetailRoute = () => defaultRoutes()["/api/v1/contacts/c-sam"] as object;

const renderAt = (path: string) =>
  renderWithProviders(
    <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Routes>
        <Route path="/contacts" element={<ContactsPage />} />
        <Route path="/contacts/:contactId" element={<ContactsPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );

describe("ContactsPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  // [REQ:SWBD-P0-009]
  it("lists contacts with their tier as a rank", async () => {
    stubConsoleFetch(defaultRoutes());
    renderAt("/contacts");
    expect(await screen.findByTestId("contacts-row")).toHaveTextContent("Sam");
    expect(screen.getByTestId("contacts-tier-badge")).toHaveAttribute("data-tier", "known");
  });

  it("explains that nobody has reached an agent when the list is empty", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/contacts"] = [];
    stubConsoleFetch(routes);
    renderAt("/contacts");
    await waitFor(() => expect(screen.getByTestId("contacts-contact-region")).toHaveAttribute("data-experience-state", "empty"));
  });

  // [REQ:SWBD-P0-010]
  it("warns about narrowed rooms before a tier is lowered and sends the change", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/contacts/c-sam"] = (init?: RequestInit) =>
      init?.method === "PUT"
        ? { contact: makeContact({ tier: "stranger" }), affected_rooms: [{ thread_id: "thread-2", channel_id: "telegram", thread_key: "chat-9", previous_ceiling: "known", new_ceiling: "stranger" }] }
        : makeContactDetailRoute();
    const { calls } = stubConsoleFetch(routes);
    renderAt("/contacts/c-sam");
    expect(await screen.findByTestId("contacts-tier-control")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("contacts-tier-known")).toHaveAttribute("aria-checked", "true"));
    expect(screen.getByTestId("contacts-tier-effect")).toBeInTheDocument();
    expect(screen.getByTestId("contacts-confirm")).toBeDisabled();
    fireEvent.click(screen.getByTestId("contacts-tier-stranger"));
    expect(screen.getByTestId("contacts-ceiling-warning")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("contacts-confirm"));
    await waitFor(() => expect(calls.some((call) => call.init?.method === "PUT")).toBe(true));
    const put = calls.find((call) => call.init?.method === "PUT");
    expect(bodyOf(put?.init)).toEqual({ tier: "stranger" });
  });
});

describe("ContactsPage edge states", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("shows the phone-width contact strip beside an open contact and survives a 404", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/contacts/ghost"] = new Response("not found", { status: 404 });
    stubConsoleFetch(routes);
    renderAt("/contacts/ghost");
    expect(await screen.findByTestId("contacts-strip")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("contacts-rooms-region")).toHaveAttribute("data-experience-state", "empty"));
  });
});
