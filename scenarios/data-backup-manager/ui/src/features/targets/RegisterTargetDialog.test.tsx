import { expect, test, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { SourceKind } from "../../api/targets";
import { strings } from "../../consts/strings";
import { RegisterTargetDialog } from "./RegisterTargetDialog";

const mutation = vi.hoisted(() => vi.fn());
vi.mock("../../hooks/useTargets", () => ({ useRegisterTarget: () => ({mutate:mutation,reset:vi.fn(),isPending:false,error:null}) }));

test("workspace selection explains consistency limits without starting a backup", () => {
  renderWithProviders(<RegisterTargetDialog open onClose={vi.fn()} />);
  fireEvent.change(screen.getByRole("combobox"), {target:{value:String(SourceKind.WORKSPACE_CHECKPOINT)}});
  expect(screen.getByRole("note")).toHaveTextContent(strings.targets.checkpointHint);
  expect(mutation).not.toHaveBeenCalled();
});
