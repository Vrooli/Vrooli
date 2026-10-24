import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ResourceTable } from "./ResourceTable";
import { renderWithProviders } from "../test-utils/renderWithProviders";

const resources = [
  { resource_name: "postgres", secrets_total: 3, secrets_found: 1, secrets_missing: 2, secrets_optional: 0, health_status: "degraded", last_checked: "2026-09-06T10:00:00Z" },
  { resource_name: "redis", secrets_total: 1, secrets_found: 1, secrets_missing: 0, secrets_optional: 0, health_status: "healthy", last_checked: "2026-09-06T10:00:00Z" }
];

describe("ResourceTable", () => {
  afterEach(cleanup);

  it("searches, filters, sorts, and opens resource details", () => {
    const onOpenResource = vi.fn();
    renderWithProviders(<ResourceTable resourceStatuses={resources} isLoading={false} onOpenResource={onOpenResource} />);

    expect(screen.getByText("postgres")).toBeInTheDocument();
    expect(screen.getByText("Action needed")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search resources"), { target: { value: "redis" } });
    expect(screen.queryByText("postgres")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(onOpenResource).toHaveBeenCalledWith("redis");

    fireEvent.change(screen.getByPlaceholderText("Search resources"), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: "Only action needed" }));
    expect(screen.queryByText("redis")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Resource/ }));
    fireEvent.click(screen.getByRole("button", { name: /Missing/ }));
    fireEvent.click(screen.getByRole("button", { name: /Total secrets/ }));
  });

  it("renders loading and empty states", () => {
    const { rerender } = renderWithProviders(<ResourceTable resourceStatuses={[]} isLoading onOpenResource={() => undefined} />);
    expect(screen.getAllByRole("row")).toHaveLength(2);
    rerender(<ResourceTable resourceStatuses={[]} isLoading={false} onOpenResource={() => undefined} />);
    expect(screen.getByText(/No resources found/)).toBeInTheDocument();
  });
});
