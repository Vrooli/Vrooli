import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { expectNoA11yViolations } from "../../test-utils";
import type { DesignDocument } from "../designs";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { StudioAssist } from "./StudioAssist";

const selected = {
  id: "page.title",
  type: "text" as const,
  props: { text: "The signal is holding." },
};

const design: DesignDocument = {
  id: "design-1",
  name: "Assist sample",
  updatedAt: "2026-11-19T10:00:00.000Z",
  revision: 2,
  theme: { ...DEFAULT_TEMPLATE.palette },
  scenario: null,
  page: null,
  frames: [
    {
      id: "desktop",
      name: "Desktop",
      device: "desktop",
      spec: { id: "page", type: "page", props: {}, children: [selected] },
    },
  ],
};

function assist() {
  return (
    <StudioAssist
      design={design}
      frame={design.frames[0]}
      selected={selected}
      scope="layer"
      onScope={vi.fn()}
      onApply={vi.fn(() => ({ ok: true as const }))}
      onApplied={vi.fn()}
    />
  );
}

describe("StudioAssist accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("has no axe violations and labels the request field", async () => {
    const promptLabel = "What should change?";
    const { container } = render(assist());
    expect(screen.getByLabelText(promptLabel)).toBeInstanceOf(HTMLTextAreaElement);
    expect(screen.getByRole("button", { name: /Propose a patch/i })).toBeDisabled();
    await expectNoA11yViolations(container);
  });
});
