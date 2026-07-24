import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { JsonCodePreview } from "./JsonCodePreview";

describe("JsonCodePreview", () => {
  it("formats deployment data with line numbers and typed JSON tokens", () => {
    const { container, rerender } = renderWithProviders(
      <JsonCodePreview
        className="manifest-preview"
        data={{
          name: "vault \"production\"",
          replicas: -1.5e2,
          enabled: true,
          disabled: false,
          optional: null
        }}
      />
    );

    expect(container.querySelector(".manifest-preview")).toBeInTheDocument();
    expect(container.querySelectorAll(".text-sky-300")).not.toHaveLength(0);
    expect(container.querySelectorAll(".text-emerald-300")).not.toHaveLength(0);
    expect(container.querySelectorAll(".text-amber-300")).not.toHaveLength(0);
    expect(container.querySelectorAll(".text-violet-300")).not.toHaveLength(0);
    expect(container.querySelectorAll(".text-rose-300")).not.toHaveLength(0);
    expect(container.querySelectorAll(".text-white\\/50")).not.toHaveLength(0);
    expect(screen.getByText('"vault \\"production\\""')).toBeInTheDocument();
    expect(screen.getByText("-150")).toBeInTheDocument();
    expect(screen.getByText("true")).toBeInTheDocument();
    expect(screen.getByText("false")).toBeInTheDocument();
    expect(screen.getByText("null")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();

    rerender(<JsonCodePreview data={undefined} />);
    expect(screen.getByText("null")).toBeInTheDocument();
  });
});
