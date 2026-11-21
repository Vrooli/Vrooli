/**
 * Tests for GeneratorLayout responsive behaviour.
 *
 * We mock useIsMobile to test both desktop and mobile layouts
 * without depending on actual viewport dimensions.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { createRef } from "react";
import {
  useSidebarStore,
  SECTION_IDS,
  type SectionId,
} from "../../store/sidebarStore";

// Mock the responsive hook — default to desktop
const mockUseIsMobile = vi.fn(() => false);
vi.mock("../../hooks/useMediaQuery", () => ({
  useIsMobile: () => mockUseIsMobile(),
  useMediaQuery: vi.fn(() => false),
  MOBILE_QUERY: "(max-width: 767px)",
}));

// Mock PipelineSidebar and MobilePipelineSummary to keep tests focused on layout
vi.mock("./PipelineSidebar", () => ({
  PipelineSidebar: ({
    onSectionClick,
  }: {
    onSectionClick: (s: string) => void;
  }) => (
    <div data-testid="pipeline-sidebar">
      <button
        type="button"
        onClick={() => {
          onSectionClick("build");
        }}
      >
        Go to build
      </button>
    </div>
  ),
}));

vi.mock("./MobilePipelineSummary", () => ({
  MobilePipelineSummary: ({ onOpenDrawer }: { onOpenDrawer: () => void }) => (
    <button type="button" data-testid="mobile-summary" onClick={onOpenDrawer}>
      Open sidebar
    </button>
  ),
}));

// Lazy import so mocks are resolved first
const { GeneratorLayout } = await import("./GeneratorLayout");

function buildSectionRefs(): Record<
  SectionId,
  React.RefObject<HTMLDivElement>
> {
  const refs = {} as Record<SectionId, React.RefObject<HTMLDivElement>>;
  for (const id of SECTION_IDS) {
    refs[id] = createRef<HTMLDivElement>();
  }
  return refs;
}

function renderLayout() {
  const refs = buildSectionRefs();
  return render(
    <GeneratorLayout sectionRefs={refs}>
      <div data-testid="main-content">Hello</div>
    </GeneratorLayout>,
  );
}

beforeEach(() => {
  mockUseIsMobile.mockReturnValue(false);
  useSidebarStore.setState({
    collapsed: false,
    activeSection: "configuration",
    mobileDrawerOpen: false,
  });
});

/* ── Desktop ─────────────────────────────────────────────────────── */

describe("GeneratorLayout — desktop", () => {
  it("renders the sidebar and main content side by side", () => {
    renderLayout();
    expect(screen.getByTestId("pipeline-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("main-content")).toBeInTheDocument();
  });

  it("does not render the mobile summary bar", () => {
    renderLayout();
    expect(screen.queryByTestId("mobile-summary")).not.toBeInTheDocument();
  });
});

/* ── Mobile ──────────────────────────────────────────────────────── */

describe("GeneratorLayout — mobile", () => {
  beforeEach(() => {
    mockUseIsMobile.mockReturnValue(true);
  });

  it("renders the mobile summary bar instead of the inline sidebar", () => {
    renderLayout();
    expect(screen.getByTestId("mobile-summary")).toBeInTheDocument();
    expect(screen.getByTestId("main-content")).toBeInTheDocument();
  });

  it("opens the drawer when the summary bar is tapped", () => {
    renderLayout();
    expect(useSidebarStore.getState().mobileDrawerOpen).toBe(false);

    fireEvent.click(screen.getByTestId("mobile-summary"));
    expect(useSidebarStore.getState().mobileDrawerOpen).toBe(true);
  });

  it("renders the sidebar inside the drawer when open", () => {
    useSidebarStore.setState({ mobileDrawerOpen: true });
    renderLayout();
    expect(screen.getByTestId("pipeline-sidebar")).toBeInTheDocument();
  });

  it("closes the drawer when a section is clicked inside it", () => {
    useSidebarStore.setState({ mobileDrawerOpen: true });
    renderLayout();

    fireEvent.click(screen.getByText("Go to build"));
    expect(useSidebarStore.getState().mobileDrawerOpen).toBe(false);
  });
});
