/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + persistent phone tabs). Feature cards keep
 * their own a11y tests.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";

vi.mock("../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { listAsks: vi.fn(() => Promise.resolve({ asks: [] })) },
  deliveryClient: { getTimeline: vi.fn(() => Promise.resolve({ notifications: [] })) },
}));

const setPhoneViewport = () => {
  Object.defineProperty(window, "innerWidth", { configurable: true, value: 390 });
  vi.spyOn(window, "matchMedia").mockImplementation((media) => {
    const match = /min-width:\s*([\d.]+)(rem|px)/i.exec(media);
    const minimum = match?.[1] && match[2] ? Number(match[1]) * (match[2].toLowerCase() === "rem" ? 16 : 1) : Infinity;
    return {
      media,
      matches: 390 >= minimum,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: () => false,
    } as MediaQueryList;
  });
};

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 1024 });
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );
    await screen.findByText(/Nothing needs an answer/);
    await expectNoA11yViolations(container);
  });

  it("retains accessible phone destinations while the compact menu opens and closes", async () => {
    setPhoneViewport();
    const user = userEvent.setup();
    const { container } = renderWithProviders(
      <>
        {/* jsdom does not evaluate the shared shell's viewport media queries. */}
        <style>{"[data-rcl-app-shell-sidebar] { display: none; }"}</style>
        <TestAppRouter initialEntries={["/"]} />
      </>,
      { withoutRouter: true },
    );

    await screen.findByText(/Nothing needs an answer/);

    expect(screen.queryByTestId(`${selectors.layout.shell}-menu-panel`)).not.toBeInTheDocument();
    const bottomBar = screen.getByTestId(`${selectors.layout.shell}-tabs`);
    expect(within(bottomBar).getAllByRole("link", { hidden: true })).toHaveLength(5);

    await user.click(screen.getByTestId(`${selectors.layout.shell}-menu`));
    const menu = await screen.findByTestId(`${selectors.layout.shell}-menu-panel`);
    expect(menu).toHaveAttribute("aria-label", "Primary navigation");
    expect(within(menu).getAllByRole("link")).toHaveLength(5);
    await expectNoA11yViolations(container);
  });
});
