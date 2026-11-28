import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { stepRegistry } from "./stepRegistry";
import type { StepRegistryProps } from "./stepRegistry";

it("registers every step declared by the API contract", () => {
  const contract = JSON.parse(
    readFileSync(
      resolve(process.cwd(), "../api/testdata/step-model.json"),
      "utf8",
    ),
  ) as { steps: Array<{ id: string }> };
  expect(Object.keys(stepRegistry).sort()).toEqual(
    contract.steps.map((step) => step.id).sort(),
  );
});

it("keeps every registered renderer callable with the shared wizard contract", () => {
  const props = {
    step: { id: "test", ordinal: 0, title: "Test", route: "/test", deferred: false },
    selectedScenarios: new Set<string>(),
    operatorState: null,
    toggleScenario: () => undefined,
    setCoreSeed: () => undefined,
    setScenarioAutoRestart: () => undefined,
    setHostOptIn: () => undefined,
    setHostConfig: () => undefined,
    setResourceEnabled: () => undefined,
    target: "local",
  } satisfies StepRegistryProps;

  for (const render of Object.values(stepRegistry)) {
    expect(render(props)).toBeTruthy();
  }
});
