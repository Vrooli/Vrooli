import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { FilterThumbnailGrid } from "./filter-thumbnail-grid";

const options = [
  { value: "grayscale", label: "Grayscale", css: "grayscale(1)" },
  { value: "sepia", label: "Sepia", css: "sepia(1)" },
];

describe("FilterThumbnailGrid", () => {
  it("falls back to labelled tiles when there is no preview", () => {
    render(
      <FilterThumbnailGrid
        label="Filter"
        value="grayscale"
        options={options}
        previewUrl={null}
        onChange={vi.fn()}
      />,
    );
    expect(screen.getByRole("radio", { name: "Grayscale" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
    // No preview image rendered in the fallback path.
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });

  it("renders preview images with a CSS filter when a previewUrl is given", () => {
    render(
      <FilterThumbnailGrid
        label="Filter"
        value="grayscale"
        options={options}
        previewUrl="blob:preview"
        onChange={vi.fn()}
      />,
    );
    const img = screen.getByRole("img", { name: "Sepia" });
    expect(img).toHaveStyle({ filter: "sepia(1)" });
  });

  it("emits the filter token on click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <FilterThumbnailGrid
        label="Filter"
        value="grayscale"
        options={options}
        previewUrl={null}
        onChange={onChange}
      />,
    );
    await user.click(screen.getByRole("radio", { name: "Sepia" }));
    expect(onChange).toHaveBeenCalledWith("sepia");
  });
});
