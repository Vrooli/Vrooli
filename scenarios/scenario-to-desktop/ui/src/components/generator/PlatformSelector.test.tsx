/**
 * Tests for PlatformSelector component.
 * Tests platform checkbox interactions and state display.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { PlatformSelector } from "./PlatformSelector";

describe("PlatformSelector", () => {
  const defaultPlatforms = { win: false, mac: false, linux: false };

  it("renders all platform checkboxes", () => {
    const onPlatformChange = vi.fn();
    render(
      <PlatformSelector
        platforms={defaultPlatforms}
        onPlatformChange={onPlatformChange}
      />
    );

    expect(screen.getByText("Target Platforms")).toBeInTheDocument();
    expect(screen.getByText("Windows")).toBeInTheDocument();
    expect(screen.getByText("macOS")).toBeInTheDocument();
    expect(screen.getByText("Linux")).toBeInTheDocument();
  });

  it("shows checked state for selected platforms", () => {
    const onPlatformChange = vi.fn();
    render(
      <PlatformSelector
        platforms={{ win: true, mac: false, linux: true }}
        onPlatformChange={onPlatformChange}
      />
    );

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes[0]).toBeChecked(); // Windows
    expect(checkboxes[1]).not.toBeChecked(); // macOS
    expect(checkboxes[2]).toBeChecked(); // Linux
  });

  it("calls onPlatformChange when Windows checkbox clicked", () => {
    const onPlatformChange = vi.fn();
    render(
      <PlatformSelector
        platforms={defaultPlatforms}
        onPlatformChange={onPlatformChange}
      />
    );

    const checkboxes = screen.getAllByRole("checkbox");
    const winCheckbox = checkboxes[0];
    expect(winCheckbox).toBeDefined();
    if (winCheckbox) fireEvent.click(winCheckbox); // Windows

    expect(onPlatformChange).toHaveBeenCalledWith("win", true);
  });

  it("calls onPlatformChange when macOS checkbox clicked", () => {
    const onPlatformChange = vi.fn();
    render(
      <PlatformSelector
        platforms={defaultPlatforms}
        onPlatformChange={onPlatformChange}
      />
    );

    const checkboxes = screen.getAllByRole("checkbox");
    const macCheckbox = checkboxes[1];
    expect(macCheckbox).toBeDefined();
    if (macCheckbox) fireEvent.click(macCheckbox); // macOS

    expect(onPlatformChange).toHaveBeenCalledWith("mac", true);
  });

  it("calls onPlatformChange when Linux checkbox clicked", () => {
    const onPlatformChange = vi.fn();
    render(
      <PlatformSelector
        platforms={defaultPlatforms}
        onPlatformChange={onPlatformChange}
      />
    );

    const checkboxes = screen.getAllByRole("checkbox");
    const linuxCheckbox = checkboxes[2];
    expect(linuxCheckbox).toBeDefined();
    if (linuxCheckbox) fireEvent.click(linuxCheckbox); // Linux

    expect(onPlatformChange).toHaveBeenCalledWith("linux", true);
  });

  it("calls onPlatformChange with false when unchecking", () => {
    const onPlatformChange = vi.fn();
    render(
      <PlatformSelector
        platforms={{ win: true, mac: true, linux: true }}
        onPlatformChange={onPlatformChange}
      />
    );

    const checkboxes = screen.getAllByRole("checkbox");
    const winCheckbox = checkboxes[0];
    expect(winCheckbox).toBeDefined();
    if (winCheckbox) fireEvent.click(winCheckbox); // Windows

    expect(onPlatformChange).toHaveBeenCalledWith("win", false);
  });

  it("renders all platforms as checked when all selected", () => {
    const onPlatformChange = vi.fn();
    render(
      <PlatformSelector
        platforms={{ win: true, mac: true, linux: true }}
        onPlatformChange={onPlatformChange}
      />
    );

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes[0]).toBeChecked();
    expect(checkboxes[1]).toBeChecked();
    expect(checkboxes[2]).toBeChecked();
  });
});
