import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { OpsFilterBar } from "./OpsFilterBar";
import type { OperationsFilters } from "../../types/operations";

const baseFilters: OperationsFilters = {
  windowSeconds: 3 * 60 * 60,
  statuses: [],
  lanes: [],
  modes: [],
  ownerTypes: [],
  q: "",
};

describe("OpsFilterBar", () => {
  it("dispatches an updated q on search input change", () => {
    const onChange = vi.fn();
    render(
      <OpsFilterBar
        filters={baseFilters}
        onFiltersChange={onChange}
        onReset={() => {}}
      />,
    );
    const input = screen.getByLabelText(/search activities/i);
    fireEvent.change(input, { target: { value: "auth" } });
    expect(onChange).toHaveBeenLastCalledWith({ q: "auth" });
  });

  it("wraps the selected status into a single-entry array", async () => {
    const onChange = vi.fn();
    render(
      <OpsFilterBar
        filters={baseFilters}
        onFiltersChange={onChange}
        onReset={() => {}}
      />,
    );
    await userEvent.selectOptions(
      screen.getByLabelText(/status filter/i),
      "running",
    );
    expect(onChange).toHaveBeenCalledWith({ statuses: ["running"] });
  });

  it("emits an empty array when status reset to all", async () => {
    const onChange = vi.fn();
    render(
      <OpsFilterBar
        filters={{ ...baseFilters, statuses: ["running"] }}
        onFiltersChange={onChange}
        onReset={() => {}}
      />,
    );
    await userEvent.selectOptions(screen.getByLabelText(/status filter/i), "");
    expect(onChange).toHaveBeenCalledWith({ statuses: [] });
  });

  it("dispatches lane and owner-type updates", async () => {
    const onChange = vi.fn();
    render(
      <OpsFilterBar
        filters={baseFilters}
        onFiltersChange={onChange}
        onReset={() => {}}
      />,
    );
    await userEvent.selectOptions(screen.getByLabelText(/lane filter/i), "execute");
    await userEvent.selectOptions(
      screen.getByLabelText(/owner type filter/i),
      "initiative",
    );
    expect(onChange).toHaveBeenCalledWith({ lanes: ["execute"] });
    expect(onChange).toHaveBeenCalledWith({ ownerTypes: ["initiative"] });
  });

  it("dispatches numeric window updates", async () => {
    const onChange = vi.fn();
    render(
      <OpsFilterBar
        filters={baseFilters}
        onFiltersChange={onChange}
        onReset={() => {}}
      />,
    );
    await userEvent.selectOptions(
      screen.getByLabelText(/time window/i),
      String(60 * 60),
    );
    expect(onChange).toHaveBeenCalledWith({ windowSeconds: 60 * 60 });
  });

  it("hides the reset button when no filters are active", () => {
    render(
      <OpsFilterBar
        filters={baseFilters}
        onFiltersChange={() => {}}
        onReset={() => {}}
      />,
    );
    expect(screen.queryByRole("button", { name: /reset/i })).toBeNull();
  });

  it("shows the reset button when filters are active and triggers onReset", async () => {
    const onReset = vi.fn();
    render(
      <OpsFilterBar
        filters={{ ...baseFilters, statuses: ["running"] }}
        onFiltersChange={() => {}}
        onReset={onReset}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: /reset/i }));
    expect(onReset).toHaveBeenCalled();
  });
});
