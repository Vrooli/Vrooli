import { afterEach, beforeEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { CounterexampleDiff } from "./CounterexampleDiff";

describe("CounterexampleDiff accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const ce = JSON.stringify({
      states: [
        { state: "draft" },
        { state: "uploading", event: "begin" },
      ],
    });
    const { container } = renderWithProviders(
      <CounterexampleDiff
        counterexampleJson={ce}
        expectedTransitions={[
          { from: "draft", event: "begin", to: "uploading", wantError: false },
        ]}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
