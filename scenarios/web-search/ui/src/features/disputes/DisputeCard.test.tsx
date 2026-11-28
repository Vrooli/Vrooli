import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  FindingSchema,
  FindingSource,
  FindingStatus,
} from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

import { interp, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { setLocale } from "../../i18n";
import en from "../../i18n/locales/en.json";

vi.mock("../../api/clients", () => ({
  findingsClient: { resolveDispute: vi.fn() },
  liveSearchClient: { search: vi.fn() },
  researchClient: { runL3: vi.fn() },
}));

import { findingsClient, researchClient } from "../../api/clients";
import { DisputeCard } from "./DisputeCard";

const queueKey = ["disputes"] as const;

const disputed = create(FindingSchema, {
  id: "d1",
  claim: "The capital of Australia is Sydney",
  confidence: 0.4,
  status: FindingStatus.DISPUTED,
  query: "australia capital",
  disputeNote: "Sources disagree on the capital",
  source: FindingSource.MANUAL,
  citations: [
    { id: "c1", url: "https://a.example/sydney", title: "Source A" },
    { id: "c2", url: "https://b.example/canberra", title: "Source B" },
  ],
});

describe("DisputeCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("displays the claim, dispute note, and both conflicting source URLs", () => {
    renderWithProviders(
      <ul>
        <DisputeCard finding={disputed} queueKey={queueKey} />
      </ul>,
    );

    expect(screen.getByTestId(selectors.disputes.item)).toHaveTextContent(disputed.claim);
    expect(screen.getByTestId(selectors.disputes.note)).toBeInTheDocument();

    const links = screen.getAllByRole("link");
    expect(links).toHaveLength(2);
    expect(links[0]).toHaveAttribute("href", "https://a.example/sydney");
    expect(links[0]).toHaveTextContent("Source A");
    expect(links[1]).toHaveAttribute("href", "https://b.example/canberra");
    expect(links[1]).toHaveTextContent("Source B");
  });

  it("resolves with supersede, sending the chosen replacement as the winner", async () => {
    vi.mocked(findingsClient.resolveDispute).mockResolvedValue({ finding: disputed } as never);

    renderWithProviders(
      <ul>
        <DisputeCard finding={disputed} queueKey={queueKey} />
      </ul>,
    );

    fireEvent.click(screen.getByTestId(selectors.disputes.resolveButton));
    fireEvent.click(screen.getByRole("radio", { name: strings.disputes.resolutionSupersede }));
    fireEvent.change(screen.getByTestId(selectors.disputes.replacement), {
      target: { value: "f-winner" },
    });
    fireEvent.change(screen.getByTestId(selectors.disputes.reason), {
      target: { value: "newer evidence" },
    });
    fireEvent.submit(screen.getByTestId(selectors.disputes.resolveForm));

    await waitFor(() => {
      expect(findingsClient.resolveDispute).toHaveBeenCalledWith({
        id: "d1",
        resolution: "supersede",
        replacement: "f-winner",
        reason: "newer evidence",
      });
    });
  });

  it("disables resolve submit for supersede until a replacement id is entered", () => {
    renderWithProviders(
      <ul>
        <DisputeCard finding={disputed} queueKey={queueKey} />
      </ul>,
    );

    fireEvent.click(screen.getByTestId(selectors.disputes.resolveButton));

    // "keep" needs no winner, so submit starts enabled.
    expect(screen.getByTestId(selectors.disputes.resolveSubmit)).toBeEnabled();

    // "supersede" with no replacement id has no winner — submit is disabled.
    fireEvent.click(screen.getByRole("radio", { name: strings.disputes.resolutionSupersede }));
    expect(screen.getByTestId(selectors.disputes.resolveSubmit)).toBeDisabled();

    // Whitespace is not a winner either.
    fireEvent.change(screen.getByTestId(selectors.disputes.replacement), {
      target: { value: "   " },
    });
    expect(screen.getByTestId(selectors.disputes.resolveSubmit)).toBeDisabled();

    fireEvent.change(screen.getByTestId(selectors.disputes.replacement), {
      target: { value: "f-winner" },
    });
    expect(screen.getByTestId(selectors.disputes.resolveSubmit)).toBeEnabled();
  });

  it("dismisses the dispute as a one-click keep resolution with an audit reason", async () => {
    vi.mocked(findingsClient.resolveDispute).mockResolvedValue({ finding: disputed } as never);

    renderWithProviders(
      <ul>
        <DisputeCard finding={disputed} queueKey={queueKey} />
      </ul>,
    );

    fireEvent.click(screen.getByTestId(selectors.disputes.dismissButton));

    await waitFor(() => {
      expect(findingsClient.resolveDispute).toHaveBeenCalledWith({
        id: "d1",
        resolution: "keep",
        replacement: "",
        reason: "dismissed from review queue",
      });
    });
  });

  it("re-researches the disputed claim via RunL3 and surfaces the run id", async () => {
    // Real English so the inline status line's interpolated run id is visible
    // (cimode renders the bare key path without interpolation).
    await setLocale("en");
    vi.mocked(researchClient.runL3).mockResolvedValue({ runId: "run-42" } as never);

    renderWithProviders(
      <ul>
        <DisputeCard finding={disputed} queueKey={queueKey} />
      </ul>,
    );

    fireEvent.click(screen.getByTestId(selectors.disputes.reresearchButton));

    await waitFor(() => {
      expect(researchClient.runL3).toHaveBeenCalledWith({ query: disputed.claim });
    });
    expect(screen.getByTestId(selectors.disputes.reresearchStatus)).toHaveTextContent(
      interp(en.disputes.reresearchStarted, { runId: "run-42" }),
    );
  });

  it("shows an error message when re-research cannot be started", async () => {
    vi.mocked(researchClient.runL3).mockRejectedValue(new Error("agent-manager unavailable"));

    renderWithProviders(
      <ul>
        <DisputeCard finding={disputed} queueKey={queueKey} />
      </ul>,
    );

    fireEvent.click(screen.getByTestId(selectors.disputes.reresearchButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.disputes.reresearchError)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.disputes.reresearchStatus)).not.toBeInTheDocument();
  });
});
