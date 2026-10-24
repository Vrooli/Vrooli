import { afterEach, describe, it } from "vitest";
import { cleanup, render } from "@testing-library/react";

import { expectNoA11yViolations } from "../../test-utils";
import { TabList, TabPanel } from "./Tabs";

type Key = "a" | "b";
const items: { value: Key; label: string }[] = [
  { value: "a", label: "Alpha" },
  { value: "b", label: "Beta" },
];

describe("Tabs accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = render(
      <>
        <TabList
          idPrefix="t"
          value="a"
          onChange={() => {}}
          items={items}
          aria-label="demo tabs"
        />
        <TabPanel idPrefix="t" value="a" active="a">
          alpha panel
        </TabPanel>
      </>,
    );
    await expectNoA11yViolations(container);
  });
});
