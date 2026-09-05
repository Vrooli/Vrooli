import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { GeneratorPage } from "./GeneratorPage";
import { useSidebarStore } from "../store/sidebarStore";
import { renderWithProviders } from "@vrooli/api-base/testing";

vi.mock("../components/layout", () => ({
  GeneratorLayout: ({ children }: { children: ReactNode }) => (
    <main>{children}</main>
  ),
}));

vi.mock("../components/generator/GeneratorForm", () => ({
  GeneratorForm: ({
    scenarioName,
    onFormStateChange,
    onSubmitHandlerReady,
    onValidationStateChange,
  }: {
    scenarioName: string;
    onFormStateChange: (state: {
      isBundled: boolean;
      bundleManifestPath: string;
    }) => void;
    onSubmitHandlerReady: (submit: () => void) => void;
    onValidationStateChange: (state: {
      errors: string[];
      isPending: boolean;
    }) => void;
  }) => (
    <div>
      <p>Form for {scenarioName}</p>
      <button
        onClick={() => {
          onFormStateChange({
            isBundled: true,
            bundleManifestPath: "/tmp/bundle.json",
          });
        }}
      >
        Share bundled state
      </button>
      <button
        onClick={() => {
          onSubmitHandlerReady(vi.fn());
        }}
      >
        Share submit handler
      </button>
      <button
        onClick={() => {
          onValidationStateChange({
            errors: ["Name is required"],
            isPending: true,
          });
        }}
      >
        Share validation
      </button>
    </div>
  ),
}));

vi.mock("../components/sections", async () => {
  const { forwardRef } = await import("react");
  return {
    SectionCard: forwardRef<
      HTMLDivElement,
      { children: ReactNode; title: string }
    >(({ children, title }, ref) => (
      <section ref={ref}>
        {title}
        {children}
      </section>
    )),
    BundleSection: forwardRef<
      HTMLDivElement,
      { isBundled: boolean; bundleManifestPath: string }
    >(({ isBundled, bundleManifestPath }, ref) => (
      <div ref={ref} data-testid="bundle">
        {String(isBundled)}:{bundleManifestPath}
      </div>
    )),
    PreflightSection: forwardRef<
      HTMLDivElement,
      { isBundled: boolean; bundleManifestPath: string }
    >(({ isBundled, bundleManifestPath }, ref) => (
      <div ref={ref} data-testid="preflight">
        {String(isBundled)}:{bundleManifestPath}
      </div>
    )),
    GenerateSection: forwardRef<
      HTMLDivElement,
      { onSubmit?: () => void; validationErrors: string[]; isPending: boolean }
    >(({ onSubmit, validationErrors, isPending }, ref) => (
      <div ref={ref} data-testid="generate">
        {String(Boolean(onSubmit))}:{validationErrors.join(",")}:
        {String(isPending)}
      </div>
    )),
    BuildSection: forwardRef<HTMLDivElement>((_, ref) => (
      <div ref={ref}>Build</div>
    )),
    SmokeTestSection: forwardRef<HTMLDivElement>((_, ref) => (
      <div ref={ref}>Smoke test</div>
    )),
    DeploySection: forwardRef<HTMLDivElement>((_, ref) => (
      <div ref={ref}>Deploy</div>
    )),
  };
});

describe("GeneratorPage", () => {
  it("routes generator form state to the bundle and preflight stages", () => {
    renderWithProviders(
      <GeneratorPage
        scenarioName="calculator"
        onScenarioNameChange={vi.fn()}
        selectedTemplate="default"
        onTemplateChange={vi.fn()}
        selectionSource="inventory"
        onOpenSigningTab={vi.fn()}
      />,
    );

    expect(screen.getByTestId("bundle")).toHaveTextContent("false:");
    fireEvent.click(
      screen.getByRole("button", { name: "Share bundled state" }),
    );
    expect(screen.getByTestId("bundle")).toHaveTextContent(
      "true:/tmp/bundle.json",
    );
    expect(screen.getByTestId("preflight")).toHaveTextContent(
      "true:/tmp/bundle.json",
    );
  });

  it("makes generator submit and validation state available to the generate stage", () => {
    renderWithProviders(
      <GeneratorPage
        scenarioName="calculator"
        onScenarioNameChange={vi.fn()}
        selectedTemplate="default"
        onTemplateChange={vi.fn()}
        selectionSource="manual"
        onOpenSigningTab={vi.fn()}
      />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Share submit handler" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Share validation" }));
    expect(screen.getByTestId("generate")).toHaveTextContent(
      "true:Name is required:true",
    );
  });

  it("updates sidebar navigation to the last visible pipeline section while scrolling", () => {
    const setActiveSection = vi.spyOn(
      useSidebarStore.getState(),
      "setActiveSection",
    );
    renderWithProviders(
      <GeneratorPage
        scenarioName="calculator"
        onScenarioNameChange={vi.fn()}
        selectedTemplate="default"
        onTemplateChange={vi.fn()}
        selectionSource={null}
        onOpenSigningTab={vi.fn()}
      />,
    );

    fireEvent.scroll(window);
    expect(setActiveSection).toHaveBeenCalledWith("deploy");
  });
});
