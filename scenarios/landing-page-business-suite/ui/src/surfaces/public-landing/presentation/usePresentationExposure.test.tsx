import { StrictMode, type ReactNode } from 'react';
import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { PresentationDiagnosticsSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import { PresentationAssignmentSource } from '@vrooli/proto-types/landing-page-business-suite/v1/config_pb';
import { usePresentationExposure } from './usePresentationExposure';
import { recordPresentationExposure } from '../../../shared/api/landing';
vi.mock('../../../shared/api/landing', () => ({ recordPresentationExposure: vi.fn() }));
const record = vi.mocked(recordPresentationExposure);
const diagnostics = () => create(PresentationDiagnosticsSchema, { resolvedRoute: '/', requestedRoute: '/', resolvedVariant: 'control', resolvedRevision: 'revision-1', locale: 'en', blockDigest: 'digest-1', weightFingerprint: 'weights-1', assignmentSource: 'weighted_visitor' });
beforeEach(() => { record.mockReset().mockResolvedValue({ $typeName: 'landing_page_business_suite.v1.RecordPresentationExposureResponse', recorded: true }); vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible'); });
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
describe('explicit rendered public exposure', () => {
  it('sends the exact proof once after ready, including StrictMode effect replay', () => {
    const d = diagnostics();
    const { rerender } = renderHook(({ ready }) => { usePresentationExposure(d, 'server_visitor', ready, '/'); }, { initialProps: { ready: false }, wrapper: ({ children }: { children: ReactNode }) => <StrictMode>{children}</StrictMode> });
    expect(record).not.toHaveBeenCalled(); rerender({ ready: true }); rerender({ ready: true });
    expect(record).toHaveBeenCalledTimes(1);
    expect(record).toHaveBeenCalledWith({ visitorId: 'server_visitor', variantSlug: 'control', revision: 'revision-1', route: '/', locale: 'en', blockDigest: 'digest-1', weightFingerprint: 'weights-1', source: PresentationAssignmentSource.WEIGHTED_VISITOR });
  });
  it('waits for visibility and removes the listener on unmount', () => {
    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    const d = diagnostics(); const { unmount } = renderHook(() => { usePresentationExposure(d, 'visitor', true, '/'); });
    expect(record).not.toHaveBeenCalled(); visibility.mockReturnValue('visible');
    act(() => { document.dispatchEvent(new Event('visibilitychange')); }); expect(record).toHaveBeenCalledTimes(1);
    unmount(); act(() => { document.dispatchEvent(new Event('visibilitychange')); }); expect(record).toHaveBeenCalledTimes(1);
  });
  it.each(['preview', 'explicit', 'source', 'fingerprint', 'revision', 'visitor', 'download', 'admin', 'route'])('does not expose %s', kind => {
    const d = diagnostics(); let path = '/'; let visitor = 'visitor';
    if (kind === 'preview') d.preview = true;
    if (kind === 'explicit') d.requestedVariant = 'control';
    if (kind === 'source') d.assignmentSource = 'explicit_url';
    if (kind === 'fingerprint') d.weightFingerprint = '';
    if (kind === 'revision') d.resolvedRevision = '';
    if (kind === 'visitor') visitor = 'bad visitor';
    if (kind === 'download') { path = '/apps/example/download'; d.resolvedRoute = '/apps/example'; }
    if (kind === 'admin') { path = '/admin/presentation'; d.resolvedRoute = path; }
    if (kind === 'route') path = '/apps/other';
    renderHook(() => { usePresentationExposure(d, visitor, true, path); }); expect(record).not.toHaveBeenCalled();
  });
  it('isolates owner errors and does not retry on visibility churn', async () => {
    record.mockRejectedValue(new Error('owner offline'));
    await act(async () => { renderHook(() => { usePresentationExposure(diagnostics(), 'visitor', true, '/'); }); await Promise.resolve(); });
    act(() => { document.dispatchEvent(new Event('visibilitychange')); }); expect(record).toHaveBeenCalledTimes(1);
  });
  it('records a new detail proof independently', () => {
    const d = diagnostics(); d.requestedRoute = '/apps/example'; d.resolvedRoute = d.requestedRoute;
    renderHook(() => { usePresentationExposure(d, 'visitor', true, '/apps/example'); });
    expect(record).toHaveBeenCalledWith(expect.objectContaining({ route: '/apps/example' }));
  });
});
