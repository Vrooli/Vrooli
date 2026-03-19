// [REQ:REQ-P0-004] Config Generation UI
import { screen, waitFor, fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithQueryClient, mockFetchSuccess, mockFetchError, mockFetchPending } from "../../test-utils";
import { StepComplete } from "./StepComplete";

const MOCK_CONFIG = { config: { resources: { postgres: { enabled: true } } } };
const MOCK_CONFIG_MULTI = { config: { resources: { postgres: { enabled: true }, redis: { enabled: true } } } };

function renderComponent(selected: Set<string>, onStartOver?: () => void) {
  return renderWithQueryClient(<StepComplete selected={selected} onStartOver={onStartOver} />);
}

describe("StepComplete", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows loading state while generating config", () => {
    mockFetchPending();
    renderComponent(new Set(["postgres"]));
    expect(screen.getByTestId("config-loading")).toBeInTheDocument();
  });

  it("shows generated config on success", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("config-output")).toBeInTheDocument();
    });
    expect(screen.getByText(/configuration ready/i)).toBeInTheDocument();
  });

  it("shows error on API failure", async () => {
    mockFetchError();
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("config-error")).toBeInTheDocument();
    });
  });

  it("shows download button when config is generated", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("download-config")).toBeInTheDocument();
    });
    expect(screen.getByTestId("download-config")).toHaveAttribute("aria-label", "Download JSON config file");
  });

  it("config output is keyboard accessible", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("config-output")).toBeInTheDocument();
    });
    expect(screen.getByTestId("config-output")).toHaveAttribute("tabindex", "0");
    expect(screen.getByTestId("config-output")).toHaveAttribute("aria-label", "Generated configuration JSON");
  });

  it("shows copy button when config is generated", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("copy-config")).toBeInTheDocument();
    });
  });

  it("shows empty state when no resources selected", () => {
    renderComponent(new Set());
    expect(screen.getByText(/no resources were selected/i)).toBeInTheDocument();
  });

  it("displays resource count in success message", async () => {
    mockFetchSuccess(MOCK_CONFIG_MULTI);
    renderComponent(new Set(["postgres", "redis"]));
    await waitFor(() => {
      expect(screen.getByText(/2 resources/i)).toBeInTheDocument();
    });
  });

  it("shows Start Over button when onStartOver is provided", async () => {
    const onStartOver = vi.fn();
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]), onStartOver);
    await waitFor(() => {
      expect(screen.getByTestId("start-over")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("start-over"));
    // Now requires confirmation
    expect(onStartOver).not.toHaveBeenCalled();
    expect(screen.getByTestId("start-over-confirm")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("start-over-confirm"));
    expect(onStartOver).toHaveBeenCalled();
  });

  it("config output uses line numbers for readability", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("config-output")).toBeInTheDocument();
    });
    const pre = screen.getByTestId("config-output");
    expect(pre.classList.contains("config-line-numbers")).toBe(true);
    // Each line of JSON should be wrapped in a span
    const spans = pre.querySelectorAll("span");
    expect(spans.length).toBeGreaterThan(0);
  });

  it("cancels start over when Cancel is clicked", async () => {
    const onStartOver = vi.fn();
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]), onStartOver);
    await waitFor(() => {
      expect(screen.getByTestId("start-over")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("start-over"));
    fireEvent.click(screen.getByTestId("start-over-cancel"));
    expect(onStartOver).not.toHaveBeenCalled();
    expect(screen.getByTestId("start-over")).toBeInTheDocument();
  });
});
