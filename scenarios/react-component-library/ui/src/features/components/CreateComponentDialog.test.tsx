import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  ComponentSchema,
  InitializeComponentResponseSchema,
  type InitializeComponentResponse,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/components", () => ({
  componentsClient: {
    initializeComponent: vi.fn(),
  },
}));

import { componentsClient } from "../../api/components";
import { CreateComponentDialog } from "./CreateComponentDialog";

function makeInitializeResponse(componentId?: string): InitializeComponentResponse {
  return create(InitializeComponentResponseSchema, {
    component: componentId ? create(ComponentSchema, { id: componentId }) : undefined,
  });
}

describe("CreateComponentDialog", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(componentsClient.initializeComponent).mockResolvedValue(makeInitializeResponse("cmp-1"));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("submits the authored component metadata with normalized tags", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<CreateComponentDialog onClose={onClose} />);

    await user.type(screen.getByTestId(selectors.components.create.slug), "notice");
    await user.type(screen.getByTestId(selectors.components.create.libraryId), "ui:Notice");
    await user.type(screen.getByTestId(selectors.components.create.displayName), "Notice");
    await user.type(screen.getByTestId(selectors.components.create.tags), " status, feedback , ");
    await user.type(screen.getByTestId(selectors.components.create.description), "An inline notice.");
    await user.type(screen.getByTestId(selectors.components.create.initialSource), "export const Notice = () => null;");
    await user.click(screen.getByTestId(selectors.components.create.submit));

    await waitFor(() => {
      expect(componentsClient.initializeComponent).toHaveBeenCalledWith({
        slug: "notice",
        libraryId: "ui:Notice",
        displayName: "Notice",
        description: "An inline notice.",
        tags: ["status", "feedback"],
        initialVersion: "0.1.0",
        fileName: "",
        initialSource: "export const Notice = () => null;",
      });
    });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("keeps the dialog open and explains an initialization failure", async () => {
    const onClose = vi.fn();
    vi.mocked(componentsClient.initializeComponent).mockRejectedValueOnce(new Error("catalog unavailable"));
    const user = userEvent.setup();
    renderWithProviders(<CreateComponentDialog onClose={onClose} />);

    await user.type(screen.getByTestId(selectors.components.create.slug), "notice");
    await user.click(screen.getByTestId(selectors.components.create.submit));

    expect(await screen.findByTestId(selectors.components.create.error)).toHaveTextContent("catalog unavailable");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("closes after a successful initialization that does not return a routeable component", async () => {
    const onClose = vi.fn();
    vi.mocked(componentsClient.initializeComponent).mockResolvedValueOnce(makeInitializeResponse());
    const user = userEvent.setup();
    renderWithProviders(<CreateComponentDialog onClose={onClose} />);

    await user.type(screen.getByTestId(selectors.components.create.slug), "notice");
    await user.click(screen.getByTestId(selectors.components.create.submit));

    await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  });

  it("prevents a duplicate submission while initialization is pending", async () => {
    let resolveInitialization: (value: InitializeComponentResponse) => void = () => {};
    vi.mocked(componentsClient.initializeComponent).mockImplementationOnce(
      () => new Promise((resolve) => { resolveInitialization = resolve; }),
    );
    const user = userEvent.setup();
    renderWithProviders(<CreateComponentDialog onClose={vi.fn()} />);

    await user.type(screen.getByTestId(selectors.components.create.slug), "notice");
    const submit = screen.getByTestId(selectors.components.create.submit);
    await user.click(submit);

    await waitFor(() => expect(submit).toBeDisabled());
    resolveInitialization(makeInitializeResponse());
  });

  it("closes without creating when cancelled", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<CreateComponentDialog onClose={onClose} />);

    await user.click(screen.getByTestId(selectors.components.create.cancel));

    expect(onClose).toHaveBeenCalledOnce();
    expect(componentsClient.initializeComponent).not.toHaveBeenCalled();
  });
});
