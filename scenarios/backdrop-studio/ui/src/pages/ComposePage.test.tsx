import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { makeStudioMocks, renderWithProviders } from "../test-utils";
import { ComposePage } from "./ComposePage";

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

async function chooseStyle() {
  renderWithProviders(<ComposePage />);
  const select = await screen.findByTestId("compose-style-select");
  // Wait for the catalog before selecting: firing a change for a value the
  // select has no option for leaves it on the empty choice, and the test then
  // asserts against a page that never received a style.
  await waitFor(() => expect(select.querySelectorAll("option").length).toBeGreaterThan(1));
  fireEvent.change(select, { target: { value: "cyanotype-arcade" } });
}

describe("ComposePage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("asks for a style before showing a plan", async () => {
    renderWithProviders(<ComposePage />);
    expect(await screen.findByText(strings.pages.compose.chooseStyle)).toBeInTheDocument();
  });

  it("resolves a plan naming the lane before anything is spent", async () => {
    await chooseStyle();
    const plan = await screen.findByTestId("compose-plan");
    expect(plan).toHaveTextContent("execution_path");
    expect(plan).toHaveTextContent("scene → image-tools treatments");
  });

  it("renders only after the operator asks for it", async () => {
    const { submitRender } = await import("../api/studio");
    await chooseStyle();
    expect(vi.mocked(submitRender)).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /compose\.render/i }));
    expect(await screen.findByTestId("mockup-landing")).toBeInTheDocument();
  });

  it("announces loading before the catalog arrives", () => {
    renderWithProviders(<ComposePage />);
    expect(screen.getByTestId(selectors.pages.compose)).toHaveAttribute(
      "data-experience-state",
      "loading",
    );
  });

  it("reports a catalog it cannot read", async () => {
    const { listStyles } = await import("../api/studio");
    vi.mocked(listStyles).mockRejectedValue(new Error("catalog unreachable"));
    renderWithProviders(<ComposePage />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.pages.compose)).toHaveAttribute(
        "data-experience-state",
        "error",
      ),
    );
  });
});
