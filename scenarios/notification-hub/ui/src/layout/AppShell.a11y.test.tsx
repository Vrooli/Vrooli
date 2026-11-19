/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + persistent phone tabs). Feature cards keep
 * their own a11y tests.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";

vi.mock("../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { listAsks: vi.fn(() => Promise.resolve({ asks: [], totalAwaiting: 0 })), listNativeConsents: vi.fn(() => Promise.resolve({ consents: [] })) },
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
      <TestAppRouter initialEntries={["/asks"]} />,
      { withoutRouter: true },
    );
    await screen.findByRole("heading", { name: "All caught up" });
    await expectNoA11yViolations(container);
  });

  it("keeps the four phone destinations as accessible links in the tab bar", async () => {
    setPhoneViewport();
    const { container } = renderWithProviders(
      <>
        {/* jsdom does not evaluate the shared shell's viewport media queries. */}
        <style>{"[data-rcl-app-shell-sidebar] { display: none; }"}</style>
        <TestAppRouter initialEntries={["/asks"]} />
      </>,
      { withoutRouter: true },
    );

    await screen.findByRole("heading", { name: "All caught up" });

    const bottomBar = screen.getByTestId(`${selectors.layout.shell}-tabs`);
    expect(within(bottomBar).getAllByRole("link", { hidden: true })).toHaveLength(4);
    expect(screen.queryByTestId(`${selectors.layout.shell}-menu`)).not.toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
