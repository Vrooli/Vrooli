import { describe, expect, it } from "vitest";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.2.0";

describe("adopter fixture", () => {
  it("uses the imported library asset", () => {
    expect(EmptyState).toBeDefined();
  });
});
