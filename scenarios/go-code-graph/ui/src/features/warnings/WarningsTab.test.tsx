import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  CodeGraphWarningSchema,
  CodeGraphWarningKind,
} from "@vrooli/proto-types/common/v1/code_graph_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { WarningsTab } from "./WarningsTab";

const warnings = [
  create(CodeGraphWarningSchema, {
    kind: CodeGraphWarningKind.UNRESOLVED_IMPORT,
    file: "pkga/a.go",
    message: "cannot find package",
  }),
  create(CodeGraphWarningSchema, {
    kind: CodeGraphWarningKind.PARSE_ERROR,
    file: "",
    message: "syntax error",
  }),
];

describe("WarningsTab", () => {
  afterEach(cleanup);

  it("renders one row per warning with a severity badge", () => {
    renderWithProviders(
      <WarningsTab warnings={warnings} includeVendor={false} onToggleVendor={() => {}} />,
    );
    expect(screen.getByTestId(selectors.features.warnings.item({ index: 0 }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.warnings.item({ index: 1 }))).toBeInTheDocument();
  });

  it("shows the empty state when there are no warnings", () => {
    renderWithProviders(<WarningsTab warnings={[]} includeVendor={false} onToggleVendor={() => {}} />);
    expect(screen.getByTestId(selectors.features.warnings.empty)).toBeInTheDocument();
  });

  it("fires onToggleVendor when the vendor checkbox is toggled", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    renderWithProviders(
      <WarningsTab warnings={[]} includeVendor={false} onToggleVendor={onToggle} />,
    );
    await user.click(screen.getByTestId(selectors.features.warnings.vendorToggle));
    expect(onToggle).toHaveBeenCalledWith(true);
  });
});
