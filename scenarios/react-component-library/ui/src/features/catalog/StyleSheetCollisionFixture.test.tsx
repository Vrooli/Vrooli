import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { StyleSheetCollisionFixture } from "./StyleSheetCollisionFixture";

describe("StyleSheetCollisionFixture", () => {
  afterEach(() => cleanup());

  it("mounts one stylesheet per exact MessageList version", () => {
    renderWithProviders(<StyleSheetCollisionFixture />);

    expect(screen.getByTestId("stylesheet-collision-v100")).toBeInTheDocument();
    expect(screen.getByTestId("stylesheet-collision-v111")).toBeInTheDocument();
    const sheets = Array.from(
      document.head.querySelectorAll<HTMLStyleElement>("style[data-rcl-sheet]"),
    ).filter((sheet) => sheet.dataset.rclSheet?.includes("messagelist"));

    expect(sheets).toHaveLength(2);
    expect(sheets.map((sheet) => sheet.dataset.rclSheet)).toEqual([
      "react-component-library-messagelist-1.1.1",
      "react-component-library-messagelist-1.0.0",
    ]);
    expect(sheets[0]?.textContent).not.toBe(sheets[1]?.textContent);
  });
});
