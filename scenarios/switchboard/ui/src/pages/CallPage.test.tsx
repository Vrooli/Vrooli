import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { CallPage } from "./CallPage";

describe("CallPage", () => {
  it("keeps the declared transcript region present and points back at the thread", () => {
    renderWithProviders(
      <MemoryRouter initialEntries={["/call/thread-9"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <Routes>
          <Route path="/call/:threadId" element={<CallPage />} />
        </Routes>
      </MemoryRouter>,
      { withoutRouter: true },
    );
    expect(screen.getByTestId("call-transcript-region")).toHaveAttribute("data-experience-state", "ready");
    expect(screen.getByRole("link")).toHaveAttribute("href", "/conversations/thread-9");
  });
});
