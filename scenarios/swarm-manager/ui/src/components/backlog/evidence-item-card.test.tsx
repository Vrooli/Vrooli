import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { createElement } from "react";
import { EvidenceItemCard } from "./evidence-item-card";
import type { EvidenceItem } from "../../services/review-service";

// Mock useCaptureContent
const mockCaptureResult = {
  content: null as string | null,
  isLoading: false,
  error: null as string | null,
  isTruncated: false,
  captureUrl: "http://test/capture",
};

vi.mock("../../hooks/useCaptureContent", () => ({
  useCaptureContent: () => mockCaptureResult,
}));

// Mock buildApiUrl
vi.mock("@vrooli/api-base", () => ({
  buildApiUrl: (path: string) => `http://test${path}`,
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function makeEvidence(overrides: Partial<EvidenceItem> = {}): EvidenceItem {
  return {
    id: "ev-1",
    type: "cli_output",
    title: "Test Evidence",
    description: "Test description",
    capture_path: "output.txt",
    verified: false,
    ...overrides,
  };
}

function renderCard(item: EvidenceItem) {
  return render(
    <EvidenceItemCard
      item={item}
      backlogKind="fix"
      backlogName="my-item"
      onVerify={vi.fn()}
    />,
    { wrapper: createWrapper() },
  );
}

/** Click the expand/collapse chevron button (last button in the card). */
function expandCard() {
  const buttons = screen.getAllByRole("button");
  const expandBtn = buttons[buttons.length - 1] as HTMLElement;
  fireEvent.click(expandBtn);
}

beforeEach(() => {
  mockCaptureResult.content = null;
  mockCaptureResult.isLoading = false;
  mockCaptureResult.error = null;
  mockCaptureResult.isTruncated = false;
  mockCaptureResult.captureUrl = "http://test/capture";
});

describe("EvidenceItemCard", () => {
  describe("Description visibility", () => {
    it("shows description without expanding", () => {
      renderCard(makeEvidence({ description: "Full build passes" }));
      expect(screen.getByText("Full build passes")).toBeTruthy();
    });
  });

  describe("CLI output evidence", () => {
    it("renders fetched content in a pre block when expanded", () => {
      mockCaptureResult.content = "hello from CLI";

      renderCard(makeEvidence({ type: "cli_output" }));
      expandCard();

      expect(screen.getByTestId("evidence-cli-output")).toBeTruthy();
      expect(screen.getByText("hello from CLI")).toBeTruthy();
    });

    it("shows loading state", () => {
      mockCaptureResult.isLoading = true;

      renderCard(makeEvidence({ type: "cli_output" }));
      expandCard();

      expect(screen.getByText("Loading output...")).toBeTruthy();
    });

    it("shows error state", () => {
      mockCaptureResult.error = "Failed to load capture: 404 Not Found";

      renderCard(makeEvidence({ type: "cli_output" }));
      expandCard();

      expect(screen.getByText(/Failed to load/)).toBeTruthy();
    });

    it("shows truncation link when content is truncated", () => {
      mockCaptureResult.content = "truncated content";
      mockCaptureResult.isTruncated = true;

      renderCard(makeEvidence({ type: "cli_output" }));
      expandCard();

      expect(screen.getByTestId("evidence-truncated-link")).toBeTruthy();
      expect(screen.getByText("Open full output")).toBeTruthy();
    });
  });

  describe("Config diff evidence", () => {
    it("renders diff with colored lines when expanded", () => {
      mockCaptureResult.content = [
        "--- a/config.yml",
        "+++ b/config.yml",
        "@@ -1,3 +1,3 @@",
        " unchanged",
        "-old line",
        "+new line",
      ].join("\n");

      renderCard(makeEvidence({ type: "config_diff", capture_path: "config.diff" }));
      expandCard();

      const diffBlock = screen.getByTestId("evidence-config-diff");
      expect(diffBlock).toBeTruthy();

      const lines = Array.from(diffBlock.querySelectorAll("pre div"));
      expect(lines.length).toBe(6);
      // --- → text-slate-400
      expect((lines[0] as HTMLElement).className).toContain("text-slate-400");
      // +++ → text-slate-400
      expect((lines[1] as HTMLElement).className).toContain("text-slate-400");
      // @@ → text-cyan-400
      expect((lines[2] as HTMLElement).className).toContain("text-cyan-400");
      // unchanged → text-slate-300
      expect((lines[3] as HTMLElement).className).toContain("text-slate-300");
      // -old → bg-red
      expect((lines[4] as HTMLElement).className).toContain("bg-red");
      // +new → bg-emerald
      expect((lines[5] as HTMLElement).className).toContain("bg-emerald");
    });
  });

  describe("Workflow recording evidence", () => {
    it("renders video element with controls when expanded", () => {
      renderCard(makeEvidence({ type: "workflow_recording", capture_path: "recording.webm" }));
      expandCard();

      const container = screen.getByTestId("evidence-workflow-recording");
      const video = container.querySelector("video");
      expect(video).toBeTruthy();
      expect((video as HTMLVideoElement).hasAttribute("controls")).toBe(true);
      expect((video as HTMLVideoElement).getAttribute("src")).toContain("recording.webm");
    });

    it("opens lightbox when video is clicked", () => {
      renderCard(makeEvidence({ type: "workflow_recording", capture_path: "recording.webm" }));
      expandCard();

      const video = screen.getByTestId("evidence-workflow-recording").querySelector("video") as HTMLVideoElement;
      fireEvent.click(video);

      expect(screen.getByTestId("media-lightbox")).toBeTruthy();
    });
  });

  describe("Review toggle", () => {
    it("calls onVerify when checkbox is clicked on unreviewed item", () => {
      const onVerify = vi.fn();
      render(
        <EvidenceItemCard
          item={makeEvidence({ verified: false })}
          backlogKind="fix"
          backlogName="my-item"
          onVerify={onVerify}
        />,
        { wrapper: createWrapper() },
      );

      fireEvent.click(screen.getByTitle("Mark as reviewed"));
      expect(onVerify).toHaveBeenCalledWith("ev-1", true);
    });

    it("calls onVerify(false) when checkbox is clicked on reviewed item", () => {
      const onVerify = vi.fn();
      render(
        <EvidenceItemCard
          item={makeEvidence({ verified: true })}
          backlogKind="fix"
          backlogName="my-item"
          onVerify={onVerify}
        />,
        { wrapper: createWrapper() },
      );

      fireEvent.click(screen.getByTitle("Mark as unreviewed"));
      expect(onVerify).toHaveBeenCalledWith("ev-1", false);
    });
  });
});
