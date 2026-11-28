import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import type { UseQueryResult } from "@tanstack/react-query";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

vi.mock("../data", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../data")>();
  return { ...actual, useCapabilities: vi.fn() };
});

import type { CapabilityDescription, CapabilityState } from "../../api/catalog";
import * as data from "../data";
import { CapabilitiesPage } from "./CapabilitiesPage";

function queryResult(
  value: CapabilityDescription | undefined,
  options: { isLoading?: boolean; isError?: boolean; isFetching?: boolean } = {},
): UseQueryResult<CapabilityDescription> {
  return {
    data: value,
    error: options.isError ? new Error("registry offline") : null,
    isError: options.isError ?? false,
    isLoading: options.isLoading ?? false,
    isFetching: options.isFetching ?? false,
    isSuccess: !options.isError && value !== undefined,
    refetch: vi.fn(),
  } as unknown as UseQueryResult<CapabilityDescription>;
}

function capability(id: string, overrides: Partial<CapabilityState> = {}): CapabilityState {
  return {
    id,
    name: id,
    description: `${id} description`,
    dependencyKind: "scenario",
    dependencySlug: id,
    status: "available",
    message: `${id} is ready`,
    ...overrides,
  };
}

const clipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, "clipboard");

describe("CapabilitiesPage", () => {
  beforeEach(() => {
    vi.mocked(data.useCapabilities).mockReturnValue(queryResult({ definitions: [], states: [] }));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    if (clipboardDescriptor) Object.defineProperty(navigator, "clipboard", clipboardDescriptor);
    else Reflect.deleteProperty(navigator, "clipboard");
  });

  it("shows loading placeholders and then labels an empty registry honestly", () => {
    vi.mocked(data.useCapabilities).mockReturnValue(queryResult(undefined, { isLoading: true }));
    const { rerender } = renderWithProviders(<CapabilitiesPage />);

    expect(document.querySelectorAll(".wb-skeleton")).toHaveLength(4);
    expect(screen.queryByText(strings.workbench.capabilities.emptyTitle)).not.toBeInTheDocument();

    vi.mocked(data.useCapabilities).mockReturnValue(queryResult({ definitions: [], states: [] }));
    rerender(<CapabilitiesPage />);
    expect(screen.getByText(strings.workbench.capabilities.emptyTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.capabilities.emptyBody)).toBeInTheDocument();
  });

  it("summarizes service status and keeps recovery actions operator-controlled", () => {
    vi.mocked(data.useCapabilities).mockReturnValue(
      queryResult({
        definitions: [],
        states: [
          capability("Ready service", { status: "available", features: ["Extraction"] }),
          capability("Degraded service", {
            status: "degraded",
            required: true,
            operatorCommand: "vrooli scenario start degraded-service",
          }),
          capability("Missing service", { status: "unavailable", features: [] }),
        ],
      }),
    );

    renderWithProviders(<CapabilitiesPage embedded />);

    expect(screen.getByTestId("capabilities-page")).toHaveClass("is-embedded");
    expect(screen.getByText((content) => content === "Ready service")).toBeInTheDocument();
    expect(screen.getByText((content) => content === "Extraction")).toBeInTheDocument();
    expect(screen.getByText((content) => content === "Degraded service")).toBeInTheDocument();
    expect(screen.getByText((content) => content === "Missing service")).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.capabilities.required)).toBeInTheDocument();
    expect(
      screen.getAllByText((content) => content === "Missing service description"),
    ).toHaveLength(2);
    expect(screen.getAllByText(strings.workbench.capabilities.startNote)).toHaveLength(2);
    expect(
      screen.getAllByRole("button", { name: strings.workbench.capabilities.start }),
    ).toHaveLength(2);
    for (const button of screen.getAllByRole("button", {
      name: strings.workbench.capabilities.start,
    })) {
      expect(button).toBeDisabled();
    }
  });

  it("re-checks capabilities and copies the recovery command when the clipboard succeeds", async () => {
    const refetch = vi.fn();
    const result = queryResult({
      definitions: [],
      states: [
        capability("Unavailable service", {
          status: "unavailable",
          operatorCommand: "vrooli scenario start service",
        }),
      ],
    });
    result.refetch = refetch;
    vi.mocked(data.useCapabilities).mockReturnValue(result);
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });

    renderWithProviders(<CapabilitiesPage />);

    fireEvent.click(screen.getByRole("button", { name: strings.workbench.capabilities.refresh }));
    expect(refetch).toHaveBeenCalledOnce();
    const copyButton = screen.getByRole("button", {
      name: strings.workbench.capabilities.copyCommand,
    });
    fireEvent.click(copyButton);

    expect(writeText).toHaveBeenCalledWith("vrooli scenario start service");
    await waitFor(() => expect(copyButton.querySelector(".lucide-check")).toBeInTheDocument());
  });

  it("does not claim capabilities are available when the registry fails", () => {
    vi.mocked(data.useCapabilities).mockReturnValue(queryResult(undefined, { isError: true }));

    renderWithProviders(<CapabilitiesPage />);

    expect(screen.getByText(strings.workbench.capabilities.errorTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.capabilities.errorBody)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.capabilities.emptyTitle)).toBeInTheDocument();
  });

  it("leaves the copy icon unchanged when clipboard access is rejected", async () => {
    vi.mocked(data.useCapabilities).mockReturnValue(
      queryResult({
        definitions: [],
        states: [capability("Service", { operatorCommand: "start service" })],
      }),
    );
    const writeText = vi.fn().mockRejectedValue(new Error("permission denied"));
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });

    renderWithProviders(<CapabilitiesPage />);
    const copyButton = screen.getByRole("button", {
      name: strings.workbench.capabilities.copyCommand,
    });
    fireEvent.click(copyButton);

    expect(writeText).toHaveBeenCalledWith("start service");
    await waitFor(() => expect(copyButton.querySelector(".lucide-copy")).toBeInTheDocument());
    expect(copyButton.querySelector(".lucide-check")).not.toBeInTheDocument();
  });
});
