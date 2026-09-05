// Type and accessor for stubbing Connect-Web domain clients in unit tests.
//
// Domain clients (src/api/<domain>.ts) export a `createClient(Service, transport)`
// instance. In tests we mock the domain module to replace that export with a
// plain object of vi.fn() stubs:
//
//   vi.mock("../api/shortcuts", () => ({
//     shortcutsClient: {
//       getEffective: vi.fn(),
//       listProfiles: vi.fn(),
//       upsertProfile: vi.fn(),
//       deleteProfile: vi.fn(),
//     },
//   }));
//
//   import { shortcutsClient as _shortcutsClient } from "../api/shortcuts";
//   const shortcutsClient = asMockedClient(_shortcutsClient);
//
// vi.mock is hoisted, so the factory must be self-contained — it cannot
// reference imported helpers. The asMockedClient cast widens the imported
// (mocked) client back to a Mock-typed surface so tests can call
// .mockResolvedValueOnce / .toHaveBeenCalledWith with type checking.

import type { Mock } from "vitest";

export type MockedConnectClient<Client> = {
  [K in keyof Client]: Mock;
};

export function asMockedClient<Client>(client: Client): MockedConnectClient<Client> {
  return client as unknown as MockedConnectClient<Client>;
}
