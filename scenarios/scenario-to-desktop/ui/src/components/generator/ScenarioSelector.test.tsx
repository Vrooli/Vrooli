/**
 * Tests for ScenarioSelector component.
 * Tests scenario selection, locked state, and user interactions.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { ScenarioSelector } from "./ScenarioSelector";
import type { ScenarioDesktopStatus } from "../scenario-inventory/types";

describe("ScenarioSelector", () => {
  const defaultProps = {
    scenarioName: "",
    loadingScenarios: false,
    onOpenScenarioModal: vi.fn(),
  };

  it("renders empty state when no scenario selected", () => {
    render(<ScenarioSelector {...defaultProps} />);

    expect(screen.getByText("Scenario Name")).toBeInTheDocument();
    expect(screen.getByText("Select a scenario")).toBeInTheDocument();
    expect(
      screen.getByText("Browse scenarios to choose a desktop target."),
    ).toBeInTheDocument();
  });

  it("shows loading state when loading scenarios", () => {
    render(<ScenarioSelector {...defaultProps} loadingScenarios={true} />);

    expect(screen.getByText("Loading scenarios...")).toBeInTheDocument();
  });

  it("displays selected scenario name", () => {
    render(<ScenarioSelector {...defaultProps} scenarioName="my-app" />);

    expect(screen.getByText("my-app")).toBeInTheDocument();
    expect(screen.getByText("Slug: my-app")).toBeInTheDocument();
  });

  it("displays scenario display name when available", () => {
    const selectedScenario: ScenarioDesktopStatus = {
      name: "my-app",
      display_name: "My Application",
      has_desktop: true,
    };

    render(
      <ScenarioSelector
        {...defaultProps}
        scenarioName="my-app"
        selectedScenario={selectedScenario}
      />,
    );

    expect(screen.getByText("My Application")).toBeInTheDocument();
  });

  it("calls onOpenScenarioModal when browse button clicked", () => {
    const onOpenScenarioModal = vi.fn();
    render(
      <ScenarioSelector
        {...defaultProps}
        onOpenScenarioModal={onOpenScenarioModal}
      />,
    );

    // There are two "Browse scenarios" elements - a link and a button
    // Get all buttons - the actual Button component
    const buttons = screen.getAllByRole("button");
    // The Browse scenarios button is the one with that text
    const browseButton = buttons.find(
      (btn) => btn.textContent === "Browse scenarios",
    );
    if (!browseButton) {
      throw new Error("Browse scenarios button not found");
    }
    fireEvent.click(browseButton);

    expect(onOpenScenarioModal).toHaveBeenCalled();
  });

  it("calls onOpenScenarioModal when browse link clicked", () => {
    const onOpenScenarioModal = vi.fn();
    render(
      <ScenarioSelector
        {...defaultProps}
        onOpenScenarioModal={onOpenScenarioModal}
      />,
    );

    // Click the text link (styled as button but is actually a button element)
    const browseLinks = screen.getAllByText("Browse scenarios");
    const browseLink = browseLinks[0];
    if (!browseLink) {
      throw new Error("Browse scenarios link not found");
    }
    fireEvent.click(browseLink);

    expect(onOpenScenarioModal).toHaveBeenCalled();
  });

  describe("locked state", () => {
    it("renders locked badge when locked with scenario", () => {
      render(
        <ScenarioSelector
          {...defaultProps}
          scenarioName="locked-app"
          locked={true}
        />,
      );

      expect(screen.getByText("Scenario")).toBeInTheDocument();
      expect(screen.getByText("locked-app")).toBeInTheDocument();
      // Should not show the normal selector UI
      expect(screen.queryByText("Scenario Name")).not.toBeInTheDocument();
    });

    it("shows change scenario button in locked state", () => {
      const onUnlock = vi.fn();
      render(
        <ScenarioSelector
          {...defaultProps}
          scenarioName="locked-app"
          locked={true}
          onUnlock={onUnlock}
        />,
      );

      const changeButton = screen.getByRole("button", {
        name: /change scenario/i,
      });
      expect(changeButton).toBeInTheDocument();

      fireEvent.click(changeButton);
      expect(onUnlock).toHaveBeenCalled();
    });

    it("hides change button when onUnlock not provided", () => {
      render(
        <ScenarioSelector
          {...defaultProps}
          scenarioName="locked-app"
          locked={true}
        />,
      );

      expect(
        screen.queryByRole("button", { name: /change scenario/i }),
      ).not.toBeInTheDocument();
    });

    it("shows normal selector when locked but no scenario", () => {
      render(
        <ScenarioSelector {...defaultProps} scenarioName="" locked={true} />,
      );

      // Should show normal selector UI
      expect(screen.getByText("Scenario Name")).toBeInTheDocument();
    });
  });

  describe("load saved URLs", () => {
    it("shows load saved button when scenario has connection config", () => {
      const onLoadSaved = vi.fn();
      const selectedScenario: ScenarioDesktopStatus = {
        name: "my-app",
        display_name: "My Application",
        has_desktop: true,
        connection_config: {
          proxy_url: "https://api.example.com",
        },
      };

      render(
        <ScenarioSelector
          {...defaultProps}
          scenarioName="my-app"
          selectedScenario={selectedScenario}
          onLoadSaved={onLoadSaved}
        />,
      );

      const loadSavedButton = screen.getByRole("button", {
        name: /load saved urls/i,
      });
      fireEvent.click(loadSavedButton);

      expect(onLoadSaved).toHaveBeenCalled();
    });

    it("hides load saved button when no connection config", () => {
      const onLoadSaved = vi.fn();
      const selectedScenario: ScenarioDesktopStatus = {
        name: "my-app",
        display_name: "My Application",
        has_desktop: true,
      };

      render(
        <ScenarioSelector
          {...defaultProps}
          scenarioName="my-app"
          selectedScenario={selectedScenario}
          onLoadSaved={onLoadSaved}
        />,
      );

      expect(
        screen.queryByRole("button", { name: /load saved urls/i }),
      ).not.toBeInTheDocument();
    });

    it("hides load saved button when onLoadSaved not provided", () => {
      const selectedScenario: ScenarioDesktopStatus = {
        name: "my-app",
        display_name: "My Application",
        has_desktop: true,
        connection_config: {
          proxy_url: "https://api.example.com",
        },
      };

      render(
        <ScenarioSelector
          {...defaultProps}
          scenarioName="my-app"
          selectedScenario={selectedScenario}
        />,
      );

      expect(
        screen.queryByRole("button", { name: /load saved urls/i }),
      ).not.toBeInTheDocument();
    });
  });

  it("shows helper text for scenario selection", () => {
    render(<ScenarioSelector {...defaultProps} />);

    expect(
      screen.getByText(
        /Select from available scenarios or enter a slug from the modal/,
      ),
    ).toBeInTheDocument();
  });
});
