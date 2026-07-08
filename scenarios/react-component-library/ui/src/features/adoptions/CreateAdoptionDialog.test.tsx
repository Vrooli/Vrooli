import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { renderWithProviders } from "../../test-utils";
import { ApplyAdoptionResponseSchema } from "@vrooli/proto-types/react-component-library/v1/adoptions/adoptions_pb";
import {
  DepIssueSchema,
  IssueKind,
  ValidateAdoptionResponseSchema,
  VerdictKind,
} from "@vrooli/proto-types/react-component-library/v1/deps/deps_pb";
import { makeAdoptionsMocks } from "./mocks/adoptions";

vi.mock("../../api/adoptions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/adoptions")>();
  return { ...actual, ...makeAdoptionsMocks() };
});

vi.mock("../../api/deps", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/deps")>();
  return {
    ...actual,
    depsClient: {
      validateAdoption: vi.fn().mockResolvedValue(
        create(ValidateAdoptionResponseSchema, {
          kind: VerdictKind.OK,
          issues: [],
        }),
      ),
      listDeclarations: vi.fn().mockResolvedValue({ declarations: [] }),
    },
  };
});

import { CreateAdoptionDialog } from "./CreateAdoptionDialog";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

async function fillRequired(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByTestId(selectors.adoptions.createComponentId), "cmp-button");
  await user.type(screen.getByTestId(selectors.adoptions.createScenario), "swarm-manager");
  const adoptedPath = screen.getByTestId(selectors.adoptions.createAdoptedPath);
  await waitFor(() => expect(adoptedPath).toHaveValue("ui/src/components/Button.tsx"));
  await user.clear(adoptedPath);
  await user.type(adoptedPath, "ui/src/components/Button.tsx");
  await user.type(screen.getByTestId(selectors.adoptions.createAdoptedVersion), "1.0.0");
}

describe("CreateAdoptionDialog", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("OK verdict enables confirm and creates the adoption", async () => {
    const { depsClient } = await import("../../api/deps");
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(depsClient.validateAdoption).mockResolvedValue(
      create(ValidateAdoptionResponseSchema, { kind: VerdictKind.OK, issues: [] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<CreateAdoptionDialog open onClose={() => {}} />);

    await fillRequired(user);

    await waitFor(() => {
      const verdict = screen.getByTestId(selectors.adoptions.createVerdict);
      expect(verdict.getAttribute("data-verdict-kind")).toBe("ok");
    });

    const confirm = screen.getByTestId(selectors.adoptions.createConfirm);
    expect(confirm).not.toBeDisabled();
    await user.click(confirm);

    await waitFor(() => {
      expect(adoptionsClient.applyAdoption).toHaveBeenCalledWith({
        componentId: "cmp-button",
        scenario: "swarm-manager",
        adoptedPath: "ui/src/components/Button.tsx",
        version: "1.0.0",
        confirmOverwrite: false,
      });
    });
  });

  it("requires explicit overwrite confirmation after target conflict", async () => {
    const { depsClient } = await import("../../api/deps");
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(depsClient.validateAdoption).mockResolvedValue(
      create(ValidateAdoptionResponseSchema, { kind: VerdictKind.OK, issues: [] }),
    );
    vi.mocked(adoptionsClient.applyAdoption)
      .mockRejectedValueOnce(new Error("target file already exists"))
      .mockResolvedValueOnce(create(ApplyAdoptionResponseSchema, { writtenPath: "" }));

    const user = userEvent.setup();
    renderWithProviders(<CreateAdoptionDialog open onClose={() => {}} />);
    await fillRequired(user);

    const confirm = await screen.findByTestId(selectors.adoptions.createConfirm);
    await waitFor(() => expect(confirm).not.toBeDisabled());
    await user.click(confirm);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.adoptions.createError).textContent).toContain(
        "target file already exists",
      );
    });
    expect(confirm.textContent).toContain("Confirm overwrite");

    await user.click(confirm);
    await waitFor(() => {
      expect(adoptionsClient.applyAdoption).toHaveBeenLastCalledWith({
        componentId: "cmp-button",
        scenario: "swarm-manager",
        adoptedPath: "ui/src/components/Button.tsx",
        version: "1.0.0",
        confirmOverwrite: true,
      });
    });
  });

  it("BLOCK verdict disables confirm and shows the failing dep", async () => {
    const { depsClient } = await import("../../api/deps");
    vi.mocked(depsClient.validateAdoption).mockResolvedValue(
      create(ValidateAdoptionResponseSchema, {
        kind: VerdictKind.BLOCK,
        issues: [
          create(DepIssueSchema, {
            depName: "react",
            declaredRange: "^18.0.0",
            scenarioVersion: "17.0.2",
            kind: IssueKind.INCOMPATIBLE_MAJOR,
          }),
        ],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<CreateAdoptionDialog open onClose={() => {}} />);
    await fillRequired(user);

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.adoptions.createVerdict).getAttribute("data-verdict-kind"),
      ).toBe("block");
    });
    expect(screen.getByTestId(selectors.adoptions.createConfirm)).toBeDisabled();
    expect(screen.getByTestId(selectors.adoptions.createVerdictIssue).textContent).toContain("react");
  });

  it("WARN verdict requires ack before enabling confirm", async () => {
    const { depsClient } = await import("../../api/deps");
    vi.mocked(depsClient.validateAdoption).mockResolvedValue(
      create(ValidateAdoptionResponseSchema, {
        kind: VerdictKind.WARN,
        issues: [
          create(DepIssueSchema, {
            depName: "lodash",
            declaredRange: "^4.0.0",
            scenarioVersion: "",
            kind: IssueKind.MISSING_DEP,
          }),
        ],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<CreateAdoptionDialog open onClose={() => {}} />);
    await fillRequired(user);

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.adoptions.createVerdict).getAttribute("data-verdict-kind"),
      ).toBe("warn");
    });
    const confirm = screen.getByTestId(selectors.adoptions.createConfirm);
    expect(confirm).toBeDisabled();

    await user.click(screen.getByTestId(selectors.adoptions.createVerdictAck));
    expect(confirm).not.toBeDisabled();
  });
});
