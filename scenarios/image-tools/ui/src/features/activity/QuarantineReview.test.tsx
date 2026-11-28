import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { QuarantinedOutputSchema, type QuarantinedOutput } from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { QuarantineReview } from "./QuarantineReview";
import type { QuarantineClient } from "../safety/quarantineClient";

const entry = (overrides: MessageInitShape<typeof QuarantinedOutputSchema> = {}): QuarantinedOutput =>
  create(QuarantinedOutputSchema, {
    id: 7n,
    jobId: "job-1",
    originalRef: "out/a.png",
    reasonCode: "sexual_unassessable",
    safetyReview: "refused",
    score: 0.95,
    mime: "image/png",
    ...overrides,
  });

const fakeClient = (overrides: Partial<QuarantineClient> = {}): QuarantineClient => ({
  list: vi.fn().mockResolvedValue([entry()]),
  label: vi.fn().mockResolvedValue(undefined),
  imageUrl: vi.fn((id, download) => `/api/v1/safety/quarantine/${id}${download ? "?download=1" : ""}`),
  ...overrides,
});

const id = "7";

describe("QuarantineReview", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("does not load the image until the operator reveals it", async () => {
    const client = fakeClient();
    const user = userEvent.setup();
    renderWithProviders(<QuarantineReview client={client} />);

    const card = await screen.findByTestId(selectors.quarantine.card({ id }));
    expect(card).toHaveTextContent("sexual_unassessable");
    expect(screen.queryByTestId(selectors.quarantine.image({ id }))).not.toBeInTheDocument();
    expect(client.imageUrl).not.toHaveBeenCalled();

    await user.click(screen.getByTestId(selectors.quarantine.reveal({ id })));
    const image = await screen.findByTestId(selectors.quarantine.image({ id }));
    expect(image).toHaveAttribute("src", "/api/v1/safety/quarantine/7");
    expect(screen.getByTestId(selectors.quarantine.download({ id }))).toHaveAttribute("href", "/api/v1/safety/quarantine/7?download=1");
  });

  it("labels the refusal as legitimate with the original output ref", async () => {
    const client = fakeClient();
    const user = userEvent.setup();
    renderWithProviders(<QuarantineReview client={client} />);
    await screen.findByTestId(selectors.quarantine.card({ id }));

    await user.click(screen.getByTestId(selectors.quarantine.legitimate({ id })));
    await waitFor(() => expect(screen.getByTestId(selectors.quarantine.saved({ id }))).toHaveTextContent("legitimate"));
    expect(client.label).toHaveBeenCalledWith("job-1", "out/a.png", "legitimate");
  });

  it("reports a label failure and keeps the card usable", async () => {
    const client = fakeClient({ label: vi.fn().mockRejectedValue(new Error("boom")) });
    const user = userEvent.setup();
    renderWithProviders(<QuarantineReview client={client} />);
    await screen.findByTestId(selectors.quarantine.card({ id }));

    await user.click(screen.getByTestId(selectors.quarantine.confirmed({ id })));
    expect(await screen.findByTestId(selectors.quarantine.labelError({ id }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.quarantine.confirmed({ id }))).not.toBeDisabled();
  });

  it("shows an empty state when nothing is quarantined", async () => {
    renderWithProviders(<QuarantineReview client={fakeClient({ list: vi.fn().mockResolvedValue([]) })} />);
    expect(await screen.findByTestId(selectors.activity.quarantine.empty)).toBeInTheDocument();
  });

  it("reports a list failure", async () => {
    renderWithProviders(<QuarantineReview client={fakeClient({ list: vi.fn().mockRejectedValue(new Error("down")) })} />);
    expect(await screen.findByTestId(selectors.activity.quarantine.error)).toBeInTheDocument();
  });
});
