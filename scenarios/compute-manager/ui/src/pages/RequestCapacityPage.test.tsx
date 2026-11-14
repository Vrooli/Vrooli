import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { requestInstance } from "../api/compute";
import { renderWithProviders } from "../test-utils";
import { RequestCapacityPage } from "./RequestCapacityPage";

vi.mock("../api/compute", () => ({
  requestInstance: vi.fn(),
}));

describe("RequestCapacityPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("refuses an invalid lifetime before calling the API", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RequestCapacityPage />);
    const lifetime = screen.getByLabelText(/lifetime/i);
    await user.clear(lifetime);
    fireEvent.submit(screen.getByTestId("page-request-form"));

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(requestInstance).not.toHaveBeenCalled();
  });

  it("renders success after the request resolves", async () => {
    vi.mocked(requestInstance).mockResolvedValueOnce({} as never);
    const user = userEvent.setup();
    renderWithProviders(<RequestCapacityPage />);
    await user.click(screen.getByRole("button"));

    await waitFor(() => expect(screen.getByText("pages.request.success")).toBeInTheDocument());
    expect(requestInstance).toHaveBeenCalledTimes(1);
  });

  it("renders an error when the request is refused", async () => {
    vi.mocked(requestInstance).mockRejectedValueOnce(new Error("refused"));
    const user = userEvent.setup();
    renderWithProviders(<RequestCapacityPage />);
    await user.click(screen.getByRole("button"));

    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
  });
});
