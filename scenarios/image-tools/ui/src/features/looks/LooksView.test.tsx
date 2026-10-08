import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { LookKind, LookSchema, type Look } from "@vrooli/proto-types/image-tools/v1/looks/looks_pb";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { LooksView, type LooksClient } from "./LooksView";

const look = (over: MessageInitShape<typeof LookSchema> = {}): Look =>
  create(LookSchema, {
    id: "noir",
    name: "Noir",
    description: "High-contrast B&W",
    kind: LookKind.FILM,
    builtin: true,
    ...over,
  });

const fakeClient = (over: Partial<LooksClient> = {}): LooksClient => ({
  list: vi.fn().mockResolvedValue([
    look({ id: "noir", name: "Noir", kind: LookKind.FILM, builtin: true }),
    look({ id: "anime", name: "Anime", kind: LookKind.STYLE, builtin: true, description: "Anime style" }),
  ]),
  renderPreview: vi.fn().mockResolvedValue({ thumbnailRef: "thumb/x.png", deferredSteps: [] }),
  ...over,
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("LooksView", () => {
  it("renders the look gallery with kind and built-in badges", async () => {
    renderWithProviders(<LooksView client={fakeClient()} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.looks.card({ index: 1 }))).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.looks.card({ index: 2 }))).toBeInTheDocument();
    expect(screen.getByText(/Noir/)).toBeInTheDocument(); // raw look.name (not translated)
    // Kind label + built-in badge render. The test i18n runs in cimode, so t()
    // returns the key — assert on the strings keys, not the English copy.
    expect(screen.getAllByText(strings.looks.kind.film).length).toBeGreaterThan(0);
    expect(screen.getAllByText(strings.looks.builtinBadge).length).toBe(2);
  });

  it("renders a preview when the Preview button is clicked", async () => {
    const user = userEvent.setup();
    const renderPreview = vi.fn().mockResolvedValue({ thumbnailRef: "thumb/x.png", deferredSteps: [] });
    renderWithProviders(<LooksView client={fakeClient({ renderPreview })} />);
    const btn = await screen.findByTestId(selectors.looks.preview({ index: 1 }));
    await user.click(btn);
    await waitFor(() => {
      expect(renderPreview).toHaveBeenCalledWith("noir");
    });
    // After a successful render, the card shows a thumbnail image (alt is the
    // i18n key under cimode).
    await waitFor(() => {
      expect(screen.getByAltText(strings.looks.thumbnailAlt)).toBeInTheDocument();
    });
  });

  it("surfaces the deferred-steps note for a style look", async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      renderPreview: vi.fn().mockResolvedValue({ thumbnailRef: "thumb/y.png", deferredSteps: ["edit_instruct"] }),
    });
    renderWithProviders(<LooksView client={client} />);
    const btn = await screen.findByTestId(selectors.looks.preview({ index: 2 }));
    await user.click(btn);
    await waitFor(() => {
      expect(screen.getByText(strings.looks.deferred)).toBeInTheDocument();
    });
  });

  it("shows an empty state when the library is empty", async () => {
    renderWithProviders(<LooksView client={fakeClient({ list: vi.fn().mockResolvedValue([]) })} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.looks.empty)).toBeInTheDocument();
    });
  });

  it("shows an error state when the library fails to load", async () => {
    renderWithProviders(<LooksView client={fakeClient({ list: vi.fn().mockRejectedValue(new Error("boom")) })} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.looks.error)).toBeInTheDocument();
    });
  });

  it("has no a11y violations", async () => {
    const { container } = renderWithProviders(<LooksView client={fakeClient()} />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.looks.grid)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
