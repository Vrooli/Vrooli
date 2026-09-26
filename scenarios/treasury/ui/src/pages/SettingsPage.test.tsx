import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { MandateSchema, MandateStatus } from "@vrooli/proto-types/treasury/v1/mandate/mandate_pb";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { renderWithProviders } from "../test-utils";
import { SettingsPage } from "./SettingsPage";

const first = create(MandateSchema, {
  id: "standing-1",
  bookId: "book-1",
  recurrenceSeconds: 3600n,
  nextChargeAt: create(TimestampSchema, { seconds: 1_800_000_000n }),
  status: MandateStatus.LIVE,
});
const second = create(MandateSchema, {
  ...first,
  id: "standing-2",
  bookId: "book-2",
});

const mocks = vi.hoisted(() => ({
  listOperatorMandates: vi.fn(),
  getScenarioFrozen: vi.fn(),
  setScenarioFrozen: vi.fn(),
  cancelStandingMandate: vi.fn(),
}));

vi.mock("../api/controls", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/controls")>()),
  ...mocks,
}));

beforeEach(async () => {
  await setLocale("en");
  mocks.listOperatorMandates.mockResolvedValue([first, second]);
  mocks.getScenarioFrozen.mockResolvedValue(true);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("Treasury controls", () => {
  // [REQ:TRS-P1-005] [REQ:TRS-P1-006]
  it("loads durable freeze state, switches books, and cancels one standing obligation", async () => {
    const cancelled = create(MandateSchema, { ...second, status: MandateStatus.REVOKED });
    mocks.cancelStandingMandate.mockResolvedValue(cancelled);
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.type(screen.getByTestId(selectors.controls.tokenInput), "operator-secret");
    await user.click(screen.getByTestId(selectors.controls.openButton));

    expect(await screen.findByRole("button", { name: "Unfreeze all spending" })).toBeVisible();
    expect(mocks.getScenarioFrozen).toHaveBeenCalledWith("operator-secret");
    expect(screen.getByTestId(selectors.controls.standingItem({ id: "standing-1" }))).toBeVisible();
    expect(screen.queryByTestId(selectors.controls.standingItem({ id: "standing-2" }))).not.toBeInTheDocument();

    await user.selectOptions(screen.getByTestId(selectors.controls.bookSelect), "book-2");
    await user.click(screen.getByRole("button", { name: "Cancel standing mandate standing-2" }));
    await waitFor(() => expect(mocks.cancelStandingMandate).toHaveBeenCalledWith("operator-secret", "standing-2"));
    expect(await screen.findByRole("status")).toHaveTextContent("Standing mandate standing-2 was cancelled.");
    expect(screen.getByRole("button", { name: "Cancel standing mandate standing-2" })).toBeDisabled();
  });

  it("toggles from the server-owned freeze state", async () => {
    mocks.setScenarioFrozen.mockResolvedValue(false);
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    await user.type(screen.getByTestId(selectors.controls.tokenInput), "operator-secret");
    await user.click(screen.getByTestId(selectors.controls.openButton));
    await user.click(await screen.findByRole("button", { name: "Unfreeze all spending" }));
    expect(mocks.setScenarioFrozen).toHaveBeenCalledWith("operator-secret", false);
    expect(await screen.findByRole("status")).toHaveTextContent("scenario-wide freeze is released");
  });

  it("keeps controls closed when operator authentication fails", async () => {
    mocks.listOperatorMandates.mockRejectedValue(new Error("unauthenticated"));
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    await user.type(screen.getByTestId(selectors.controls.tokenInput), "wrong-token");
    await user.click(screen.getByTestId(selectors.controls.openButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("Authentication failed");
    expect(screen.queryByTestId(selectors.controls.freezeAllButton)).not.toBeInTheDocument();
  });

  it("preserves the visible freeze state when a control write fails", async () => {
    mocks.setScenarioFrozen.mockRejectedValue(new Error("write failed"));
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    await user.type(screen.getByTestId(selectors.controls.tokenInput), "operator-secret");
    await user.click(screen.getByTestId(selectors.controls.openButton));
    await user.click(await screen.findByRole("button", { name: "Unfreeze all spending" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("could not be recorded");
    expect(screen.getByRole("button", { name: "Unfreeze all spending" })).toBeVisible();
  });
});
