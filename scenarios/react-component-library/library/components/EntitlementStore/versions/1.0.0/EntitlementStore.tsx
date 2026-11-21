/**
 * @libraryId react-component-library:EntitlementStore
 * @displayName EntitlementStore
 * @description Shared entitlement state contract for paid surfaces
 * @version 1.0.0
 * @tags ["entitlement","monetization"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
export interface EntitlementSnapshot { identity: string; tier: string; status: string; features: string[]; }
export type EntitlementListener = (snapshot: EntitlementSnapshot | null) => void;
export class EntitlementStore {
  private snapshot: EntitlementSnapshot | null = null;
  private readonly listeners = new Set<EntitlementListener>();
  get(): EntitlementSnapshot | null { return this.snapshot; }
  set(snapshot: EntitlementSnapshot | null): void { this.snapshot = snapshot; this.listeners.forEach((listener) => listener(snapshot)); }
  subscribe(listener: EntitlementListener): () => void { this.listeners.add(listener); return () => this.listeners.delete(listener); }
}
export default EntitlementStore;
