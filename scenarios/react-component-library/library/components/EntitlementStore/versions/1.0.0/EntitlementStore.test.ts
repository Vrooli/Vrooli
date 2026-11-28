import { describe, expect, it, vi } from "vitest";

import { EntitlementStore } from "./EntitlementStore";

describe("EntitlementStore", () => {
  it("publishes the shared snapshot contract to subscribers", () => {
    const store = new EntitlementStore();
    const listener = vi.fn();
    const unsubscribe = store.subscribe(listener);
    const snapshot = { identity: "customer@example.test", tier: "pro", status: "active", features: ["record"] };

    store.set(snapshot);

    expect(store.get()).toEqual(snapshot);
    expect(listener).toHaveBeenCalledWith(snapshot);
    unsubscribe();
    store.set(null);
    expect(listener).toHaveBeenCalledOnce();
  });
});
