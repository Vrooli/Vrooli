import { OwnerOpenRequestSchema, type OwnerOpenRequest } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { readDesktopRecovery, writeDesktopRecovery, type RecoveryActor } from "./desktopRecoveryMarker";
import { createContext, createElement, useContext, useEffect, useState, useSyncExternalStore, type ReactNode } from "react";
import { clone, create, equals } from "@bufbuild/protobuf";
import { SessionRefSchema, SurfaceRefSchema, type SessionRef, type SurfaceRef } from "@vrooli/proto-types/common/v1/surface_pb";
import { desktopClient, operatorClient, operatorHeaders } from "../../api/desktop";
import { useCompanionStop } from "../companion/CompanionPresentation";

export type DesktopAccount = { id: string; realm: string; token: string; refresh: string; expires: number; email: string };
export type DesktopSession = { ref: SessionRef; control: boolean; expires: number };
type Snapshot = { account?: DesktopAccount; active?: DesktopSession; opening: boolean; stopping: boolean; stopRequested: boolean; uncertain: boolean; recoveryRequired: boolean; recovering: boolean; needsAuthentication: boolean; refreshing: boolean; restored: boolean; persistenceFailed: boolean; recovered: Map<string, DesktopSession> };

const sessionIdentity = (ref: SessionRef) => JSON.stringify([ref.surface?.target?.ownerScenario, ref.surface?.target?.resourceId, ref.surface?.target?.hostNodeId, ref.surface?.ownerScenario, ref.surface?.surfaceId, ref.sessionId, ref.desktopSessionId]);

// The shell owns authority and unresolved identities. Routed views own only
// presentation state; leaving a route must not discard a lease or repeat Stop.
class DesktopSessions {
  private snapshot: Snapshot = { opening: false, stopping: false, stopRequested: false, uncertain: false, recoveryRequired: false, recovering: false, needsAuthentication: false, refreshing: false, restored: false, persistenceFailed: false, recovered: new Map() };
  private expectedActor: RecoveryActor | undefined;
  constructor() {
    try {
      const marker = readDesktopRecovery();
      if (!marker) return;
      this.expectedActor = marker.actor;
      this.openUnknown = marker.unknownOpen;
      this.pendingOpen = marker.pendingOpen;
      const recovered = new Map(marker.sessions.map(ref => [sessionIdentity(ref), { ref, control: false, expires: 0 }]));
      this.snapshot = { ...this.snapshot, recovered, restored: true, recoveryRequired: true, uncertain: true, needsAuthentication: true };
    } catch {
      this.openUnknown = true;
      this.snapshot = { ...this.snapshot, restored: true, recoveryRequired: true, uncertain: true, needsAuthentication: true, persistenceFailed: true };
    }
  }
  private recovery: Promise<boolean> | undefined;
  private openUnknown = false;
  private pendingOpen: OwnerOpenRequest | undefined;
  private cancelOpening = false;
  private refreshing: Promise<boolean> | undefined;
  private listeners = new Set<() => void>();
  getSnapshot = () => this.snapshot;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private publish(patch: Partial<Snapshot>): boolean {
    this.snapshot = { ...this.snapshot, ...patch };
    let persisted = false;
    try {
      const actor = this.snapshot.account ?? this.expectedActor;
      const unresolved = Boolean(this.snapshot.active) || this.snapshot.opening || this.snapshot.recoveryRequired || this.snapshot.recovered.size > 0;
      if (unresolved && !actor) throw new Error("Recovery actor is unavailable");
      const refs = [...this.snapshot.recovered.values()].map(session => session.ref);
      if (this.snapshot.active) refs.push(this.snapshot.active.ref);
      writeDesktopRecovery(unresolved && actor ? { actor, unknownOpen: this.openUnknown, pendingOpen: this.pendingOpen, sessions: refs } : undefined);
      this.snapshot = { ...this.snapshot, persistenceFailed: false };
      persisted = true;
    } catch { this.snapshot = { ...this.snapshot, persistenceFailed: true, recoveryRequired: true, uncertain: true }; }
    this.listeners.forEach(listener => listener());
    return persisted;
  }
  setAccount = (account?: DesktopAccount) => {
    const current = this.snapshot.account ?? this.expectedActor;
    if (this.snapshot.opening || this.snapshot.refreshing) throw new Error("Desktop account operation in progress");
    if (!account && (this.snapshot.active || this.snapshot.recoveryRequired)) throw new Error("Desktop session remains unresolved");
    if (account && (!account.id || !account.realm || account.expires <= Date.now() || (current && (current.id !== account.id || current.realm !== account.realm)))) throw new Error("Original desktop actor authentication required");
    this.expectedActor = account ? { id: account.id, realm: account.realm } : undefined;
    this.publish({ account, needsAuthentication: false, recoveryRequired: Boolean(account) });
    if (account) void this.recover();
  };
  refresh = (): Promise<boolean> => {
    if (this.refreshing) return this.refreshing;
    if (this.snapshot.needsAuthentication) return Promise.resolve(false);
    const account = this.snapshot.account;
    if (!account || !account.refresh) { this.publish({ needsAuthentication: Boolean(account) }); return Promise.resolve(false); }
    this.publish({ refreshing: true });
    this.refreshing = (async () => {
      try {
        const response = await operatorClient.refresh({ refreshToken: account.refresh });
        const tokens = response.tokens;
        const expires = Number(tokens?.accessTokenExpiresAt?.seconds ?? 0n) * 1000;
        if (!tokens?.accessToken || !tokens.refreshToken || expires <= Date.now()) throw new Error("Invalid refreshed credentials");
        if (this.snapshot.account !== account) return false;
        this.publish({ account: { ...account, token: tokens.accessToken, refresh: tokens.refreshToken, expires }, needsAuthentication: false });
        return true;
      } catch { this.publish({ needsAuthentication: true }); return false; }
      finally { this.refreshing = undefined; this.publish({ refreshing: false }); }
    })();
    return this.refreshing;
  };
  setUncertain = (uncertain: boolean, ref?: SessionRef) => {
    if (!ref || !this.snapshot.active || !equals(SessionRefSchema, ref, this.snapshot.active.ref)) return;
    this.publish({ uncertain: this.snapshot.stopRequested || uncertain });
  };
  canObserve = (ref: SessionRef) => Boolean(this.snapshot.active && !this.snapshot.needsAuthentication && !this.snapshot.recoveryRequired && !this.snapshot.stopRequested && equals(SessionRefSchema, ref, this.snapshot.active.ref));
  expire = () => {
    const { active, account } = this.snapshot;
    if (active && active.expires <= Date.now() && !this.snapshot.stopRequested) this.publish({ stopRequested: true, uncertain: true });
    if (account && account.expires <= Date.now() + 15_000 && !this.snapshot.needsAuthentication) void this.refresh();
  };
  open = async (surface: SurfaceRef, control: boolean): Promise<DesktopSession> => {
    const { account, active, opening, recoveryRequired } = this.snapshot;
    if (!account || this.snapshot.needsAuthentication || account.expires <= Date.now() || active || opening || recoveryRequired) throw new Error("Desktop admission unavailable");
    const request = clone(OwnerOpenRequestSchema, create(OwnerOpenRequestSchema, { surface, ttlSeconds: 120, control, requestId: crypto.randomUUID() }));
    this.pendingOpen = request;
    if (!this.publish({ opening: true })) {
      this.pendingOpen = undefined;
      this.publish({ opening: false });
      throw new Error("Desktop recovery identity could not be persisted");
    }
    try {
      const result = await desktopClient.open(request, { headers: operatorHeaders(account.token) });
      if (!result.session?.surface || !result.expiresAt || !equals(SurfaceRefSchema, result.session.surface, surface)) throw new Error("Missing desktop admission identity");
      const session = { ref: result.session, control: result.control, expires: Number(result.expiresAt.seconds) * 1000 };
      this.pendingOpen = undefined;
      this.publish({ active: session, stopRequested: false, uncertain: false });
      return session;
    } catch (error) {
      // A rejected or lost response does not establish that admission never ran.
      this.publish({ recoveryRequired: true, uncertain: true });
      throw error;
    } finally {
      this.publish({ opening: false });
      if (this.cancelOpening) { this.cancelOpening = false; await this.stop(); }
    }
  };
  recover = (): Promise<boolean> => {
    if (this.recovery) return this.recovery;
    if (!this.snapshot.account || this.snapshot.needsAuthentication || this.snapshot.opening) return Promise.resolve(false);
    this.publish({ recovering: true, recoveryRequired: true });
    this.recovery = this.readAdmissions().finally(() => {
      this.recovery = undefined;
      this.publish({ recovering: false });
    });
    return this.recovery;
  };
  private async readAdmissions(): Promise<boolean> {
    const account = this.snapshot.account;
    if (!account) return false;
    const pending = new Map(this.snapshot.recovered);
    try {
      const request = this.pendingOpen;
      if (request) {
        const result = await desktopClient.reconcileOpen(request, { headers: operatorHeaders(account.token) });
        if (result.requestId !== request.requestId) throw new Error("Wrong desktop request disposition");
        if (result.state === "not_admitted" && !result.session) this.pendingOpen = undefined;
        else if (result.state === "forwarding" && result.session?.surface && result.session.sessionId && result.session.desktopSessionId && request.surface && equals(SurfaceRefSchema, result.session.surface, request.surface) && result.expiresAt && result.control === request.control) {
          pending.set(sessionIdentity(result.session), { ref: result.session, expires: Number(result.expiresAt.seconds) * 1000, control: result.control });
          this.pendingOpen = undefined;
        } else throw new Error("Desktop Open remains unresolved");
        this.publish({ recovered: new Map(pending) });
      }
      let token = "";
      const seen = new Set<string>();
      for (let pageNumber = 0; ; pageNumber++) {
        if (pageNumber >= 50 || seen.has(token)) throw new Error("Desktop admission traversal incomplete");
        seen.add(token);
        const page = await desktopClient.listAdmissions({ pageToken: token, pageSize: 50 }, { headers: operatorHeaders(account.token) });
        if (page.admissions.length > 50) throw new Error("Desktop admission page exceeded bound");
        for (const admission of page.admissions) {
          const ref = admission.session;
          if (!ref?.sessionId || !ref.desktopSessionId || !ref.surface?.surfaceId || !ref.surface.target?.resourceId || !admission.expiresAt) throw new Error("Invalid desktop admission reference");
          if (this.snapshot.active && equals(SessionRefSchema, ref, this.snapshot.active.ref)) continue;
          pending.set(sessionIdentity(ref), { ref, expires: Number(admission.expiresAt.seconds) * 1000, control: admission.control });
        }
        this.publish({ recovered: new Map(pending) });
        if (!page.nextPageToken) break;
        token = page.nextPageToken;
      }
      // Only an exact positive receipt removes an identity, including entries
      // retained from earlier incomplete traversals. No Open or Stop is retried.
      const entries = [...pending.entries()];
      let next = 0;
      await Promise.all(Array.from({ length: Math.min(4, entries.length) }, async () => {
        while (next < entries.length) {
          const entry = entries[next++];
          if (!entry) return;
          const [key, session] = entry;
          try {
            const receipt = await desktopClient.readCleanup({ session: session.ref }, { headers: operatorHeaders(account.token) });
            if (receipt.session && equals(SessionRefSchema, receipt.session, session.ref) && receipt.observedAt && receipt.released) pending.delete(key);
          } catch { /* Missing or unavailable evidence remains unresolved. */ }
        }
      }));
      const unresolved = this.openUnknown || Boolean(this.pendingOpen) || pending.size > 0;
      this.publish({ recovered: new Map(pending), recoveryRequired: unresolved, uncertain: this.snapshot.active ? this.snapshot.uncertain : unresolved });
      return !this.snapshot.recoveryRequired;
    } catch {
      this.publish({ recovered: new Map(pending), recoveryRequired: true, uncertain: true });
      return false;
    }
  }
  stop = async (): Promise<boolean> => {
    const { account, active, stopping, stopRequested } = this.snapshot;
    if (this.snapshot.opening) { this.cancelOpening = true; return false; }
    if (!active && this.snapshot.recoveryRequired) return this.recover();
    if (!account || !active || stopping || this.snapshot.needsAuthentication || account.expires <= Date.now()) return false;
    this.publish({ stopping: true, stopRequested: true, uncertain: true });
    try {
      if (stopRequested) {
        const receipt = await desktopClient.readCleanup({ session: active.ref }, { headers: operatorHeaders(account.token) });
        if (!receipt.session || !equals(SessionRefSchema, receipt.session, active.ref) || !receipt.observedAt || !receipt.released) throw new Error("Desktop cleanup remains unconfirmed");
      } else await desktopClient.stop({ session: active.ref }, { headers: operatorHeaders(account.token) });
      this.publish({ active: undefined, uncertain: false, stopRequested: false });
      return true;
    } catch { this.publish({ uncertain: true }); return false; }
    finally { this.publish({ stopping: false }); }
  };
}
const Context = createContext<DesktopSessions | null>(null);
export function DesktopSessionsProvider({ children }: { children: ReactNode }) {
  const [store] = useState(() => new DesktopSessions());
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot, store.getSnapshot);
  useEffect(() => { const timer = setInterval(store.expire, 500); return () => clearInterval(timer); }, [store]);
  useCompanionStop("desktop", "desktop", Boolean(state.active) || state.opening || state.recoveryRequired, async () => { await store.stop(); }, state.stopRequested || state.recoveryRequired);
  return createElement(Context.Provider, { value: store }, children);
}
export function useDesktopSession() {
  const store = useContext(Context);
  if (!store) throw new Error("Desktop session provider is missing");
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot, store.getSnapshot);
  return { ...state, store };
}
