import { createRef } from "react";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { Text } from "@vrooli/react-component-library/Text/1.1.2";

describe("Text restyle contract", () => {
  it("composes the consumer class and forwards the root ref", () => {
    const ref = createRef<HTMLElement>();
    renderWithProviders(
      <Text ref={ref} className="consumer-text" textStyle="heading">
        Consumer override
      </Text>,
    );

    const root = screen.getByText("Consumer override");
    expect(root).toHaveClass("rcl-text", "consumer-text");
    expect(ref.current).toBe(root);
  });
});
