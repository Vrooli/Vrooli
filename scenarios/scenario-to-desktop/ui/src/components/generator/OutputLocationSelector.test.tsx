import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { OutputLocationSelector, OutputPathField } from "./OutputLocationSelector";

describe("OutputLocationSelector", () => {
  it("presents durable, staging, and custom destinations and reports changes", () => {
    const onChange = vi.fn();
    render(<OutputLocationSelector locationMode="proper" standardPath="/scenarios/canvas-lab/.desktop" stagingPreview="/tmp/canvas-lab" onChange={onChange} />);
    expect(screen.getByRole("radio", { name: /Proper/ })).toBeChecked();
    fireEvent.click(screen.getByRole("radio", { name: /Temporary/ }));
    fireEvent.click(screen.getByRole("radio", { name: /Custom path/ }));
    expect(onChange).toHaveBeenNthCalledWith(1, "temp");
    expect(onChange).toHaveBeenNthCalledWith(2, "custom");
    expect(screen.getByText("/tmp/canvas-lab")).toBeInTheDocument();
  });

  it("edits an explicit custom output path", () => {
    const onOutputPathChange = vi.fn();
    render(<OutputPathField outputPath="/opt/apps" onOutputPathChange={onOutputPathChange} />);
    fireEvent.change(screen.getByLabelText("Output Directory"), { target: { value: "/srv/desktop-apps" } });
    expect(onOutputPathChange).toHaveBeenCalledWith("/srv/desktop-apps");
    expect(screen.getByText(/Used only when choosing a custom location/)).toBeInTheDocument();
  });
});
