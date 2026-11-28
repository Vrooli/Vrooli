import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders as render } from "../test-utils";
import userEvent from "@testing-library/user-event";
import ActivationSetupCard from "./ActivationSetupCard";

describe("ActivationSetupCard", () => {
  afterEach(() => { vi.restoreAllMocks(); window.localStorage.clear(); });

  it("shows durable activation progress without implying an account is required", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ first_command_run: true, first_ai_command_accepted: false }), { status: 200 })));
    render(<ActivationSetupCard />);
    expect(await screen.findByText(/you can use Aquila without an account/i)).toBeTruthy();
    expect(screen.getByText(/Run your first command/).textContent).toContain("✓");
    expect(screen.getByText(/Try an AI command/).textContent).toContain("○");
  });

  it("offers retry after a status failure and persists dismissal", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response("unavailable", { status: 503 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ first_command_run: false, first_ai_command_accepted: false }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    const view = render(<ActivationSetupCard />);
    expect(await screen.findByText("Setup progress is temporarily unavailable.")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByText(/Run your first command/)).toBeTruthy();
    await user.click(await screen.findByRole("button", { name: "Dismiss setup guide" }));
    expect(window.localStorage.getItem("wc-activation-setup-dismissed-v1")).toBe("true");
    view.unmount();
    render(<ActivationSetupCard />);
    expect(screen.queryByRole("heading", { name: "Get started with Aquila" })).toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
