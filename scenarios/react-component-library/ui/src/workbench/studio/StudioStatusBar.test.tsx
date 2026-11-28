import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import type { DesignDocument } from "../design-types";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { StudioStatusBar } from "./StudioStatusBar";

const doc: DesignDocument = {
  id: "design-1",
  name: "Sample",
  updatedAt: "2026-11-19T10:00:00.000Z",
  revision: 3,
  theme: { ...DEFAULT_TEMPLATE.palette },
  frames: [],
  scenario: null,
  page: null,
};

const coverage = { rcl: 1, custom: 2, total: 3, ratio: 1 / 3 };

function bar() {
  return (
    <StudioStatusBar
      doc={doc}
      coverage={coverage}
      audits={{}}
      saved={{ revision: 3, at: Date.now(), signature: "sig" }}
      dirty={false}
      saving={false}
      saveFailed={false}
      onRetrySave={vi.fn()}
      openCount={1}
      issueCount={0}
      zoom={1}
    />
  );
}

describe("studio status honesty", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("shows a durable server save as saved", () => {
    renderWithProviders(bar());
    expect(screen.getByText(/Saved · revision 3/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
  });
});
