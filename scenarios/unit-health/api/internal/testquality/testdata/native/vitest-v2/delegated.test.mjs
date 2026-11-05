// @vitest-environment jsdom
import { test } from "vitest";
import { expectNoA11yViolations } from "./a11y-helper.mjs";

test("C022 delegated axe assertion", async () => {
  const main = document.createElement("main");
  main.innerHTML = '<button type="button">Save</button>';
  document.body.append(main);
  try { await expectNoA11yViolations(main); } finally { main.remove(); }
});
