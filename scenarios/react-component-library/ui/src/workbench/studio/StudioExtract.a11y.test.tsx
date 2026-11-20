import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { expectNoA11yViolations } from "../../test-utils";
import { node } from "../templates/spec";
import { StudioExtract } from "./StudioExtract";

const spec = node("page", "page", {}, [
  node("plan.basic", "surface", { pad: 16 }, [
    node("plan.basic.title", "text", { text: "Starter" }),
    node("plan.basic.body", "text", { text: "For one seat" }),
  ]),
  node("plan.pro", "surface", { pad: 20 }, [
    node("plan.pro.title", "text", { text: "Team" }),
    node("plan.pro.body", "text", { text: "For the whole crew" }),
  ]),
]);

describe("StudioExtract accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("labels the section, the list and each select action", async () => {
    const onSelect = vi.fn();
    const { container } = renderWithProviders(<StudioExtract spec={spec} onSelect={onSelect} />);

    expect(screen.getByRole("heading", { name: /Extraction candidates/i })).toBeInTheDocument();
    expect(screen.getByRole("list")).toBeInTheDocument();

    const select = screen.getByRole("button", { name: /Select example of surface/i });
    fireEvent.click(select);
    expect(onSelect).toHaveBeenCalledWith("plan.basic");

    await expectNoA11yViolations(container);
  });

  it("states the honest empty result", async () => {
    const { container } = renderWithProviders(<StudioExtract spec={node("page", "page")} onSelect={vi.fn()} />);

    expect(screen.queryByRole("list")).not.toBeInTheDocument();
    expect(screen.getByText(/No repeated custom structure/i)).toBeInTheDocument();
    expect(screen.getByText(/RCL primitives, pages and slots are excluded/i)).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
