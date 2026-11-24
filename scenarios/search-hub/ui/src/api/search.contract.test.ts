import { expect, it } from "vitest";

import type { FederationStatus } from "./search";

type ClassifierAvailabilityIsExcluded = "classifierAvailable" extends keyof FederationStatus

  ? false
  : true;
const classifierAvailabilityIsExcluded: ClassifierAvailabilityIsExcluded = true;

const federationStatusKeys: Record<keyof FederationStatus, true> = {
  activeStrategy: true,
  rerankerAvailable: true,
  providers: true,
};

it("keeps the Search UI status contract limited to live fields", () => {
  expect(classifierAvailabilityIsExcluded).toBe(true);
  expect(Object.keys(federationStatusKeys).sort()).toEqual([
    "activeStrategy",
    "providers",
    "rerankerAvailable",
  ]);
});
