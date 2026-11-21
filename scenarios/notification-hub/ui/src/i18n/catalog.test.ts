import { describe, expect, it } from "vitest";
// Plain `.mjs` shared with codegen and lint; typed through catalog-source.d.mts.
import { readCatalog } from "../../scripts/catalog-source.mjs";
import { LOCALE_CODES } from "./locales";
import * as catalogs from "./catalog";

describe("catalog assembly", () => {
  it.each(LOCALE_CODES)("the browser and Node readers assemble the same %s catalog", (code) => {
    expect(catalogs[code]).toEqual(readCatalog(code));
  });

  it("nests dotted file names under their key path", () => {
    expect(
      catalogs.assemble({
        "./en/common.json": { retry: "Retry" },
        "./en/pages.dashboard.json": { title: "Updates" },
        "./en/pages.delivery.json": { title: "Delivery" },
      }),
    ).toEqual({
      common: { retry: "Retry" },
      pages: { dashboard: { title: "Updates" }, delivery: { title: "Delivery" } },
    });
  });
});
