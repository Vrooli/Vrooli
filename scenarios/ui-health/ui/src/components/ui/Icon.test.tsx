import { render, screen } from "@testing-library/react";
import { Search } from "lucide-react";
import { describe, expect, it } from "vitest";

import { Icon } from "./Icon";

describe("Icon", () => {
  it("is decorative (aria-hidden) without a label", () => {
    const { container } = render(<Icon icon={Search} />);
    const svg = container.querySelector("svg");
    expect(svg).toHaveAttribute("aria-hidden");
  });
  it("exposes an accessible name when label is provided", () => {
    const lbl = "search-x";
    render(<Icon icon={Search} label={lbl} />);
    expect(screen.getByRole("img", { name: lbl })).toBeInTheDocument();
  });
});
