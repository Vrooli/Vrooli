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

  it("marks onboarding complete after config generation succeeds", async () => {
    globalThis.fetch = vi.fn()
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(MOCK_CONFIG),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ status: "ok" }),
      });

    renderComponent(new Set(["postgres"]));

    await waitFor(() => {
      expect(screen.getByTestId("config-output")).toBeInTheDocument();
    });

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenNthCalledWith(
        2,
        expect.stringContaining("/api/v1/complete"),
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ user_id: "default" }),
        }),
      );
    });
  });

  it("shows error on API failure with alert role and message", async () => {
    mockFetchError();
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("config-error")).toBeInTheDocument();
    });
    // Error should be announced via role="alert" for screen readers
    expect(screen.getByRole("alert")).toBeInTheDocument();
    // Should contain a meaningful error message
    expect(screen.getByTestId("config-error")).toHaveTextContent(/failed|error|problem/i);
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

  it("shows singular 'resource' when exactly one selected", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByText(/1 resource\./i)).toBeInTheDocument();
    });
  });

  it("does not show Start Over button when onStartOver is not provided", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("config-output")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("start-over")).not.toBeInTheDocument();
  });

  it("copy button calls clipboard.writeText with config JSON", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("copy-config")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("copy-config"));
    expect(writeText).toHaveBeenCalledWith(JSON.stringify(MOCK_CONFIG, null, 2));
  });

  it("config output renders full JSON structure including nested keys", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("config-output")).toBeInTheDocument();
    });
    const output = screen.getByTestId("config-output");
    // Verify the JSON contains key structural elements
    expect(output).toHaveTextContent("postgres");
    expect(output).toHaveTextContent("enabled");
    expect(output).toHaveTextContent("resources");
  });

  it("download button creates and clicks a temporary link", async () => {
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("download-config")).toBeInTheDocument();
    });

    const createObjectURL = vi.fn().mockReturnValue("blob:test-url");
    const revokeObjectURL = vi.fn();
    Object.assign(URL, { createObjectURL, revokeObjectURL });

    const fakeLink = document.createElement("a");
    const clickSpy = vi.spyOn(fakeLink, "click");
    vi.spyOn(document, "createElement").mockImplementation((tag: string) => {
      if (tag === "a") return fakeLink;
      return document.createElement(tag);
    });

    fireEvent.click(screen.getByTestId("download-config"));

    expect(createObjectURL).toHaveBeenCalledWith(expect.any(Blob));
    expect(clickSpy).toHaveBeenCalled();
    expect(fakeLink.download).toBe("vrooli-config.json");
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:test-url");
  });

  it("copy button shows 'Copied' text after clicking", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    mockFetchSuccess(MOCK_CONFIG);
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("copy-config")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("copy-config"));
    await waitFor(() => {
      expect(screen.getByText("Copied")).toBeInTheDocument();
    });
  });
});
