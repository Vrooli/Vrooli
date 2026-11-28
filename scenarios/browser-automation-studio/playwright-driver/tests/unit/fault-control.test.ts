import { FaultController } from '../../src/fault-control';

describe('FaultController', () => {
  let now = 1_000;
  let controller: FaultController;

  beforeEach(() => { now = 1_000; controller = new FaultController(() => now); });

  it('consumes a one-shot fault before exposing its controlled outcome', () => {
    controller.arm({ token: 'a-valid-drill-token', fault: 'driver_unavailable', ttl_ms: 1000 });
    expect(controller.consume('a-valid-drill-token', 'driver_unavailable')).toBe(true);
    expect(controller.snapshot()).toEqual([]);
    expect(controller.consume('a-valid-drill-token', 'driver_unavailable')).toBe(false);
    expect(controller.auditEvents().map((event) => event.event)).toEqual(['armed', 'consumed']);
  });

  it('isolates faults by token and expires residue', () => {
    controller.arm({ token: 'first-drill-token-1', fault: 'capacity_lease', ttl_ms: 100, lease_count: 2 });
    expect(controller.capacityReserved('other-drill-token')).toBe(0);
    expect(controller.capacityReserved('first-drill-token-1')).toBe(2);
    now += 100;
    expect(controller.snapshot()).toEqual([]);
    expect(controller.auditEvents().at(-1)?.event).toBe('expired');
  });

  it('rejects duplicate ownership and malformed requests', () => {
    controller.arm({ token: 'duplicate-drill-token', fault: 'driver_unavailable', ttl_ms: 1000 });
    expect(() => controller.arm({ token: 'duplicate-drill-token', fault: 'driver_unavailable', ttl_ms: 1000 })).toThrow('already armed');
    expect(() => controller.arm({ token: 'short', fault: 'driver_unavailable', ttl_ms: 1000 })).toThrow('opaque');
  });
});
