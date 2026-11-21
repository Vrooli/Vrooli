"""Read Offer Desk's own condition rows; retain absent sensors as declared gaps, never as zero."""
import json

inputs = program.inputs()

envelope = {'program': 'offer-desk.setpoint-read', 'version': '1', 'status': 'ok',
            'phase': 'collect', 'inputs': {}, 'signals': {'rows': []}, 'errors': [], 'evidence': []}


def row(name, reading, target, in_band=None, reason=None):
    envelope['signals']['rows'].append({'row': name, 'reading': reading, 'target': target,
                                        'in_band': in_band, 'unavailable': reason is not None,
                                        'reason': reason})


guarded = program.guarded


def read_failure(where, exc):
    # Preserve the runtime's typed transport class. Only an unreachable source
    # lowers board status; permanent read gaps remain visible without implying
    # that the board failed to report its evidence.
    status, klass = program.classify(exc)
    if status == 'unavailable':
        envelope['status'] = 'partial'
    envelope['errors'].append({'class': klass, 'where': where, 'detail': str(exc)[:180]})
    return klass


def failure_reason(klass):
    return 'scenario_unreachable' if klass == 'scenario_unreachable' else 'unreliable:' + klass


results = gather(
    guarded(lambda: offer_desk.offers.board_show(rows='entries')),
    guarded(lambda: offer_desk.offers.meters(rows='meters')),
    guarded(lambda: offer_desk.offers.space(rows='cells')),
    guarded(lambda: vrooli_memory.learning.measure(scope='offer-desk-usage', operation='board-read', rows='cohorts')),
)
envelope['phase'] = 'classify'
board, meters, space, learning = results

# --- ledger account coverage and financial posture, both from the board -----------------
if isinstance(board, Exception):
    klass = read_failure('board-show', board)
    reason = failure_reason(klass)
    row('ledger-account-coverage', None, 'every ACTIVE or TRIGGER_MET node maps to a ledger account', None, reason)
    row('default-alive-posture', None, 'a readable default-alive gap', None, reason)
else:
    entries = board.head(80)
    total = board.count()
    # protojson omits false, so an absent actualsAvailable means "not available".
    mapped = sum(1 for e in entries if e.get('actualsAvailable', False))
    earning = [e for e in entries if str(e.get('status') or '') in ('ACTIVE', 'TRIGGER_MET')]
    earning_mapped = sum(1 for e in earning if e.get('actualsAvailable', False))
    row('ledger-account-coverage',
        {'nodes': total, 'mapped': mapped, 'earning_nodes': len(earning), 'earning_mapped': earning_mapped},
        'every ACTIVE or TRIGGER_MET node maps to a ledger account',
        (earning_mapped == len(earning)) if earning else None,
        None if entries else 'unreliable:empty_board')
    gap_text = str(board.meta().get('defaultAliveGap') or '')
    readable = bool(gap_text) and not gap_text.lower().startswith('unavailable')
    # An unreliable row keeps in_band null: the posture text is evidence, not a verdict.
    row('default-alive-posture', gap_text or None, 'a readable default-alive gap',
        True if readable else None,
        None if readable else 'unreliable:' + (gap_text[:60] or 'no_posture_text'))
    envelope['evidence'].append('offer-desk/offers/board-show')

# --- meter declaration coverage ---------------------------------------------------------
if isinstance(meters, Exception):
    klass = read_failure('meters', meters)
    reason = failure_reason(klass)
    row('meter-coverage', None, 'no deliverable meter gaps and no undeclared streams', None, reason)
else:
    meta = meters.meta()
    gaps = len(meta.get('deliverableMeterGaps') or [])
    undeclared = len(meta.get('undeclaredStreams') or [])
    row('meter-coverage', {'meters': meters.count(), 'deliverable_meter_gaps': gaps,
                           'undeclared_streams': undeclared},
        'no deliverable meter gaps and no undeclared streams',
        gaps == 0 and undeclared == 0, None)
    envelope['evidence'].append('offer-desk/offers/meters')

# --- obligation denominator confidence ---------------------------------------------------
if isinstance(space, Exception):
    klass = read_failure('space', space)
    reason = failure_reason(klass)
    row('obligation-confidence', None, 'denominator confidence above sketch', None, reason)
else:
    confidence = str(space.meta().get('denominatorConfidence') or '')
    row('obligation-confidence', {'cells': space.count(), 'denominator_confidence': confidence or None},
        'denominator confidence above sketch',
        (confidence != '' and confidence != 'sketch') if confidence else None,
        None if confidence else 'unreliable:no_confidence_reported')
    envelope['evidence'].append('offer-desk/offers/space')

# --- outcome-linked learning -------------------------------------------------------------
if isinstance(learning, Exception):
    klass = read_failure('learning', learning)
    reason = failure_reason(klass)
    row('learning', None, None, None, reason)
else:
    meta = learning.meta()
    reliable = bool(meta.get('reliable', False))
    truncated = bool(meta.get('truncated', False))
    row('learning', {'eligible_attempts': int(meta.get('eligibleAttempts', 0) or 0),
                     'reliable': reliable, 'reason': meta.get('reason'),
                     'cohorts': learning.head(8), 'truncated': truncated},
        None, None,
        None if reliable and not truncated
        else 'unreliable:' + str(meta.get('reason') or 'capped_or_empty').removeprefix('unreliable:'))
    envelope['evidence'].append('vrooli-memory/learning/measure')

# --- rows this board does not own --------------------------------------------------------
row('external-friction', None, None, None, 'read_elsewhere:agent-manager.friction-digest')
# catalog-verify is a governed read, but it did not return inside a 90 s probe on 2026-09-08,
# so this board declines to call it rather than spend its whole budget on one row.
row('catalog-conformance', None, 'declared catalog sources reconcile against live records', None,
    'kernel_invoke_budget')
row('promotion-latency', None, 'no promotion proposal waits on the operator beyond one review cycle',
    None, 'pending_telemetry')

envelope['phase'] = 'report'
print(json.dumps(envelope, allow_nan=False, separators=(',', ':')))
