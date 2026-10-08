import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { FindingStatus } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the active label and active data-status", () => {
    renderWithProviders(<StatusBadge status={FindingStatus.ACTIVE} />);
    const badge = screen.getByTestId(selectors.findings.statusBadge);
    expect(badge).toHaveTextContent(strings.findings.statusActive);
    expect(badge).toHaveAttribute("data-status", "ACTIVE");
  });

  it("renders disputed in the warning tone", () => {
    renderWithProviders(<StatusBadge status={FindingStatus.DISPUTED} />);
    const badge = screen.getByTestId(selectors.findings.statusBadge);
    expect(badge).toHaveTextContent(strings.findings.statusDisputed);
    expect(badge.className).toContain("app-warning");
  });

  it("renders superseded muted", () => {
    renderWithProviders(<StatusBadge status={FindingStatus.SUPERSEDED} />);
    const badge = screen.getByTestId(selectors.findings.statusBadge);
    expect(badge).toHaveTextContent(strings.findings.statusSuperseded);
    expect(badge.className).toContain("app-muted-foreground");
  });
});
