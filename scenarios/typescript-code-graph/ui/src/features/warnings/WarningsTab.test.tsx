import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
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
    file: "src/a.ts",
    message: "cannot resolve module",
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
    renderWithProviders(<WarningsTab warnings={warnings} />);
    expect(screen.getByTestId(selectors.features.warnings.item({ index: 0 }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.warnings.item({ index: 1 }))).toBeInTheDocument();
  });

  it("shows the empty state when there are no warnings", () => {
    renderWithProviders(<WarningsTab warnings={[]} />);
    expect(screen.getByTestId(selectors.features.warnings.empty)).toBeInTheDocument();
  });

  it("renders the project-level label for a warning with no file", () => {
    renderWithProviders(<WarningsTab warnings={warnings} />);
    // The second warning (file: "") renders the project-level fallback.
    expect(screen.getByTestId(selectors.features.warnings.item({ index: 1 }))).toBeInTheDocument();
  });
});
