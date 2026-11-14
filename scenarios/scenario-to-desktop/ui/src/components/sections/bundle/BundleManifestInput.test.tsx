import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { BundleManifestInput } from "./BundleManifestInput";

describe("BundleManifestInput", () => {
  it("reports edits and explains the staged bundle manifest", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <BundleManifestInput value="" onChange={onChange} />,
    );

    expect(screen.getByLabelText("Bundle manifest path")).toHaveAttribute(
      "placeholder",
      "/home/you/Vrooli/scenarios/my-scenario/platforms/electron/bundle/bundle.json",
    );
    fireEvent.change(screen.getByLabelText("Bundle manifest path"), {
      target: { value: "/tmp/bundle.json" },
    });
    expect(onChange).toHaveBeenCalledWith("/tmp/bundle.json");
  });

  it("shows the staged-file guidance for a populated and disabled path", () => {
    renderWithProviders(
      <BundleManifestInput
        value=" /tmp/bundle.json "
        onChange={vi.fn()}
        disabled
      />,
    );

    expect(screen.getByLabelText("Bundle manifest path")).toBeDisabled();
    expect(
      screen.getByText(/alongside staged binaries\/assets/),
    ).toBeInTheDocument();
  });
});
