import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { stepRegistry } from "./stepRegistry";

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
