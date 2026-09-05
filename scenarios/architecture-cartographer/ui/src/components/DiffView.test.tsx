import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { DiffView } from "./DiffView";

describe("DiffView", () => {
  it("renders added, removed, and context lines", () => {
    render(
      <DiffView
        addedLabel="Added"
        removedLabel="Removed"
        lines={[
          { kind: "context", text: "ctx", lineNumber: 1 },
          { kind: "added", text: "new", lineNumber: 2 },
          { kind: "removed", text: "old", lineNumber: 3 },
        ]}
      />,
    );
    const root = screen.getByTestId(selectors.shared.diffView.root);
    expect(root).toHaveTextContent("ctx");
    expect(root).toHaveTextContent("new");
    expect(root).toHaveTextContent("old");
  });

  it("labels added rows for screen readers", () => {
    render(
      <DiffView
        addedLabel="Added"
        removedLabel="Removed"
        lines={[{ kind: "added", text: "new" }]}
      />,
    );
    const addedRow = screen
      .getByTestId(selectors.shared.diffView.root)
      .querySelector('[aria-label="Added"]');
    expect(addedRow).not.toBeNull();
  });
});
