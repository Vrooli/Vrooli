import { create } from '@bufbuild/protobuf';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { CapacityFindingSchema } from '@vrooli/proto-types/system-monitor/v1/capacity/capacity_pb';
import { FindingsPanel } from './FindingsPanel';

// provider-free-exception: read-only findings receive all data through props.
describe('FindingsPanel identity', () => {
  it('keeps same-owner aggregate rows distinct across GPUs through reordered updates', () => {
    const errors = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const gpu0 = create(CapacityFindingSchema, {
      ownerId: 'music-worker', ownerKind: 'resource', resourceKind: 'vram',
      class: 'over_claim', gpuIndex: 0, pid: 0, claimId: 'claim-0',
      message: 'GPU zero aggregate', severity: 'warn',
    });
    const gpu1 = create(CapacityFindingSchema, {
      ...gpu0, gpuIndex: 1, claimId: 'claim-1', message: 'GPU one aggregate',
    });
    const view = render(<FindingsPanel available findings={[gpu0, gpu1]} />);
    const zeroRow = screen.getByText('GPU zero aggregate').closest('li');
    const oneRow = screen.getByText('GPU one aggregate').closest('li');

    view.rerender(<FindingsPanel available findings={[gpu1, gpu0]} />);

    expect(screen.getAllByRole('listitem')).toHaveLength(2);
    expect(screen.getByText('GPU zero aggregate').closest('li')).toBe(zeroRow);
    expect(screen.getByText('GPU one aggregate').closest('li')).toBe(oneRow);
    expect(errors.mock.calls.filter((args) => args.some((arg) => String(arg).includes('same key')))).toEqual([]);
  });

  it('keeps independent claims on one GPU distinct when values and order change', () => {
    const errors = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const first = create(CapacityFindingSchema, {
      ownerId: 'worker', ownerKind: 'resource', resourceKind: 'vram',
      class: 'claimed', gpuIndex: 0, pid: 0, claimId: 'first', message: 'First claim',
    });
    const second = create(CapacityFindingSchema, { ...first, claimId: 'second', message: 'Second claim' });
    const view = render(<FindingsPanel available findings={[first, second]} />);
    const firstRow = screen.getByText('First claim').closest('li');
    const secondRow = screen.getByText('Second claim').closest('li');

    view.rerender(<FindingsPanel available findings={[
      { ...second, observedBytes: 900n, message: 'Second updated' },
      { ...first, class: 'over_claim', pids: [231, 587], observedBytes: 500n, message: 'First updated' },
    ]} />);

    expect(screen.getByText('First updated').closest('li')).toBe(firstRow);
    expect(screen.getByText('Second updated').closest('li')).toBe(secondRow);
    expect(errors.mock.calls.filter((args) => args.some((arg) => String(arg).includes('same key')))).toEqual([]);
  });
  it('keeps one unclaimed process on two GPUs distinct', () => {
    const errors = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const gpu0 = create(CapacityFindingSchema, {
      ownerId: 'unknown', resourceKind: 'vram', class: 'unclaimed', gpuIndex: 0,
      pid: 231, message: 'Process on GPU zero',
    });
    const gpu1 = create(CapacityFindingSchema, { ...gpu0, gpuIndex: 1, message: 'Process on GPU one' });
    const view = render(<FindingsPanel available findings={[gpu0, gpu1]} />);
    const zeroRow = screen.getByText('Process on GPU zero').closest('li');
    const oneRow = screen.getByText('Process on GPU one').closest('li');

    view.rerender(<FindingsPanel available findings={[gpu1, gpu0]} />);

    expect(screen.getByText('Process on GPU zero').closest('li')).toBe(zeroRow);
    expect(screen.getByText('Process on GPU one').closest('li')).toBe(oneRow);
    expect(errors.mock.calls.filter((args) => args.some((arg) => String(arg).includes('same key')))).toEqual([]);
  });

  it('keeps unclaimed process rows distinct across reordered attribution updates', () => {
    const errors = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const first = create(CapacityFindingSchema, {
      ownerId: 'unknown', resourceKind: 'vram', class: 'unclaimed', gpuIndex: 0,
      pid: 231, message: 'First process',
    });
    const second = create(CapacityFindingSchema, { ...first, pid: 587, message: 'Second process' });
    const view = render(<FindingsPanel available findings={[first, second]} />);
    const firstRow = screen.getByText('First process').closest('li');
    const secondRow = screen.getByText('Second process').closest('li');

    view.rerender(<FindingsPanel available findings={[
      second, { ...first, ownerId: 'newly-attributed', message: 'First updated' },
    ]} />);

    expect(screen.getByText('First updated').closest('li')).toBe(firstRow);
    expect(screen.getByText('Second process').closest('li')).toBe(secondRow);
    expect(errors.mock.calls.filter((args) => args.some((arg) => String(arg).includes('same key')))).toEqual([]);
  });
});
