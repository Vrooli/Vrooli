import { describe, expect, it } from "vitest";

import { migrationClient } from "./migration";

describe("api/migration.migrationClient", () => {
  it("exposes every MigrationService RPC as a callable method", () => {
    const rpcs = [
      "createMigration",
      "listMigrations",
      "getMigrationStatus",
      "nextMigrationStep",
      "resolveFinding",
      "applyFinding",
      "reauditMigration",
      "closeMigration",
    ] as const;
    for (const rpc of rpcs) {
      expect(typeof migrationClient[rpc]).toBe("function");
    }
  });
});
