import { renderWithProviders } from "../../test-utils";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { selectors } from "../../consts/selectors";
import { Code } from "./code";

describe("Code", () => {
  it("renders inline code by default", () => {
    renderWithProviders(<Code>foo</Code>);
    const el = screen.getByTestId(selectors.ui.code.root);
    expect(el.tagName).toBe("CODE");
    expect(el.className).toMatch(/font-mono/);
  });

  it("renders a pre block for variant=block", () => {
    renderWithProviders(<Code variant="block">foo</Code>);
    const el = screen.getByTestId(selectors.ui.code.root);
    expect(el.tagName).toBe("PRE");
    expect(el.className).toMatch(/rounded-panel/);
  });
});
