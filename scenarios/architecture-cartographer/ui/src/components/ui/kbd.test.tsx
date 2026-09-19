import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { Kbd } from "./kbd";

import { selectors } from "../../consts/selectors";

describe("Kbd", () => {
  it("renders a <kbd> element with the base classes", () => {
    render(<Kbd>Ctrl</Kbd>);
    const el = screen.getByTestId(selectors.ui.kbd.root);
    expect(el.tagName).toBe("KBD");
    expect(el.className).toMatch(/font-mono/);
  });
});
