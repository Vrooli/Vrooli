import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { ScenarioSelector } from "./ScenarioSelector";

const scenarios = [
  { name: "secrets-manager", description: "Manage deployment secrets", version: "v2" },
  { name: "desktop-console", description: "Desktop operations", version: "" }
];

describe("ScenarioSelector", () => {
  afterEach(cleanup);

  it("searches and selects a lifecycle scenario", () => {
    const onSearchChange = vi.fn();
    const onSelect = vi.fn();
    renderWithProviders(
      <ScenarioSelector scenarios={scenarios} filtered={scenarios} search="" isLoading={false} selectedScenario="secrets-manager" onSearchChange={onSearchChange} onSelect={onSelect} />
    );
    expect(screen.getByText("2 available")).toBeInTheDocument();
    expect(screen.getByText("Selected")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search scenarios (desktop, cloud, pick a name...)"), { target: { value: "desktop" } });
    fireEvent.click(screen.getByRole("button", { name: /desktop-console/ }));
    expect(onSearchChange).toHaveBeenCalledWith("desktop");
    expect(onSelect).toHaveBeenCalledWith("desktop-console");
  });

  it("represents loading and no-match states", () => {
    const { rerender } = renderWithProviders(
      <ScenarioSelector scenarios={scenarios} filtered={[]} search="vault" isLoading selectedScenario="" onSearchChange={() => {}} onSelect={() => {}} />
    );
    expect(screen.getByText("Loading scenarios...")).toBeInTheDocument();
    rerender(<ScenarioSelector scenarios={scenarios} filtered={[]} search="vault" isLoading={false} selectedScenario="" onSearchChange={() => {}} onSelect={() => {}} />);
    expect(screen.getByText("No scenarios match “vault”.")).toBeInTheDocument();
  });
});
