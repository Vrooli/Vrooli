/**
 * Development-only, token-scoped fault injection for recovery qualification.
 * This controller is intentionally owned by a single driver instance: it is
 * never a process-global test fixture and never exposes the drill token.
 */
export type FaultName = 'driver_unavailable' | 'fail_after_session_registration' | 'capacity_lease';
export type FaultState = 'armed' | 'consumed' | 'expired' | 'disarmed';

export interface ArmFaultRequest {
  token: string;
  fault: FaultName;
  ttl_ms: number;
  remaining_uses?: number;
  lease_count?: number;
}

export interface FaultSnapshot {
  fault: FaultName;
  state: FaultState;
  remaining_uses: number;
  expires_at: string;
  lease_count?: number;
}

export interface FaultAuditEvent {
  fault: FaultName;
  event: 'armed' | 'consumed' | 'expired' | 'disarmed';
  at: string;
}

interface ArmedFault extends ArmFaultRequest {
  remainingUses: number;
  expiresAt: number;
  state: FaultState;
}

export class FaultController {
  private readonly faults = new Map<string, ArmedFault>();
  private readonly audit: FaultAuditEvent[] = [];

  constructor(private readonly now: () => number = Date.now) {}

  arm(request: ArmFaultRequest): FaultSnapshot {
    this.expire();
    if (!/^[A-Za-z0-9_-]{16,128}$/.test(request.token)) throw new Error('drill token must be an opaque 16-128 character token');
    if (!['driver_unavailable', 'fail_after_session_registration', 'capacity_lease'].includes(request.fault)) throw new Error('unsupported fault');
    if (!Number.isInteger(request.ttl_ms) || request.ttl_ms < 100 || request.ttl_ms > 300_000) throw new Error('ttl_ms must be between 100 and 300000');
    if (this.faults.has(request.token)) throw new Error('a fault is already armed for this drill token');
    const remainingUses = request.remaining_uses ?? (request.fault === 'capacity_lease' ? 1 : 1);
    if (!Number.isInteger(remainingUses) || remainingUses < 1 || remainingUses > 100) throw new Error('remaining_uses must be between 1 and 100');
    const leaseCount = request.lease_count ?? 1;
    if (request.fault === 'capacity_lease' && (!Number.isInteger(leaseCount) || leaseCount < 1 || leaseCount > 100)) throw new Error('lease_count must be between 1 and 100');
    const fault: ArmedFault = { ...request, remainingUses, expiresAt: this.now() + request.ttl_ms, state: 'armed', lease_count: request.fault === 'capacity_lease' ? leaseCount : undefined };
    this.faults.set(request.token, fault);
    this.record(fault.fault, 'armed');
    return this.snapshotFault(fault);
  }

  consume(token: string | undefined, faultName: FaultName): boolean {
    this.expire();
    if (!token) return false;
    const fault = this.faults.get(token);
    if (!fault || fault.fault !== faultName || fault.state !== 'armed') return false;
    // Consumption is committed before the caller observes the injected outcome.
    fault.remainingUses -= 1;
    if (fault.remainingUses === 0) {
      fault.state = 'consumed';
      this.faults.delete(token);
    }
    this.record(fault.fault, 'consumed');
    return true;
  }

  capacityReserved(token: string | undefined): number {
    this.expire();
    if (!token) return 0;
    const fault = this.faults.get(token);
    return fault?.state === 'armed' && fault.fault === 'capacity_lease' ? fault.lease_count ?? 0 : 0;
  }

  disarm(token: string): boolean {
    this.expire();
    const fault = this.faults.get(token);
    const removed = this.faults.delete(token);
    if (removed && fault) this.record(fault.fault, 'disarmed');
    return removed;
  }

  snapshot(): FaultSnapshot[] {
    this.expire();
    return [...this.faults.values()].map((fault) => this.snapshotFault(fault));
  }

  auditEvents(): readonly FaultAuditEvent[] { return this.audit; }

  private expire(): void {
    const now = this.now();
    for (const [token, fault] of this.faults) {
      if (fault.expiresAt <= now) {
        this.faults.delete(token);
        this.record(fault.fault, 'expired');
      }
    }
  }

  private snapshotFault(fault: ArmedFault): FaultSnapshot {
    return { fault: fault.fault, state: fault.state, remaining_uses: fault.remainingUses, expires_at: new Date(fault.expiresAt).toISOString(), ...(fault.lease_count ? { lease_count: fault.lease_count } : {}) };
  }

  private record(fault: FaultName, event: FaultAuditEvent['event']): void {
    this.audit.push({ fault, event, at: new Date(this.now()).toISOString() });
    if (this.audit.length > 100) this.audit.shift();
  }
}
