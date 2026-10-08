"""Read the Content Desk marketing board once: current campaign/draft work, the deterministic next action per draft, capability readiness gaps, and a baseline diff."""
import json

inputs = program.inputs()

envelope = {'program': 'content-desk.board-read', 'version': '1', 'status': 'failed',
            'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}

ROW_CAP = 80
SAMPLE_CAP = 8

# The artifact lifecycle is the sole source of the next action. Held here so an
# unrecognised status is reported as drift rather than silently given an action.
KNOWN_DRAFT_STATUS = ('requested', 'drafting', 'drafted', 'checking', 'blocked',
                      'reviewed', 'approved', 'published', 'abandoned')
NEXT_ACTION = {
    'requested': 'Begin drafting',
    'drafting': 'Complete drafting',
    'drafted': 'Start checking and review',
    'checking': 'Complete the review run',
    'blocked': 'Resolve the blocking gate',
    'reviewed': 'Operator approval required',
    'approved': 'Submit for release',
}
KNOWN_CAMPAIGN_STATUS = ('proposed', 'active', 'closed')
READINESS_DIMENSIONS = ('definition_status', 'implementation_status', 'operational_readiness',
                        'output_quality', 'distribution_connectivity')

guarded = program.guarded

fail = program.fail

state = {'baseline': {}}


def _str(value):
    return '' if value is None else str(value)


def _field(row, camel, snake):
    # protojson returns camelCase; keep the snake_case fallback for a manual caller.
    value = row.get(camel)
    if value is None:
        value = row.get(snake)
    return value


def _dedupe(values, cap=SAMPLE_CAP):
    seen = []
    for value in values:
        text = _str(value).strip()
        if text and text not in seen:
            seen.append(text)
        if len(seen) >= cap:
            break
    return seen


def step_validate():
    # phase: validate — inputs only; no binding may be called here.
    envelope['phase'] = 'validate'
    raw = inputs.get('baseline') or {}
    if not isinstance(raw, dict):
        return fail('failed', 'invalid_input', 'baseline must be an object', 'validate')
    campaigns = raw.get('campaigns') or {}
    drafts = raw.get('drafts') or {}
    if not isinstance(campaigns, dict):
        return fail('failed', 'invalid_input', 'baseline.campaigns must be an object of id to status', 'validate')
    if not isinstance(drafts, dict):
        return fail('failed', 'invalid_input', 'baseline.drafts must be an object of id to status', 'validate')
    state['baseline'] = {
        'campaigns': {str(k): str(v) for k, v in campaigns.items()},
        'drafts': {str(k): str(v) for k, v in drafts.items()},
    }
    envelope['inputs'] = {
        'baseline_campaigns': len(state['baseline']['campaigns']),
        'baseline_drafts': len(state['baseline']['drafts']),
        'baseline_captured_at': raw.get('captured_at') or '',
    }
    return 'collect'


def step_collect():
    # phase: collect — three governed reads; the board is the single address for team state.
    envelope['phase'] = 'collect'
    reads = {
        'campaigns': guarded(lambda: content_desk.campaigns.list()),
        'drafts': guarded(lambda: content_desk.artifacts.list()),
        'capabilities': guarded(lambda: content_desk.capabilities.list()),
        'offer_ladder': guarded(lambda: offer_desk.offers.release_ladder(rows='entries')),
    }
    names = list(reads.keys())
    outcomes = gather(*[reads[name] for name in names])
    by_name = dict(zip(names, outcomes))

    # The campaign and draft reads are the board's required spine: if either
    # fails, the board cannot report current work and the program fails.
    for name, binding in (('campaigns', 'content-desk/campaigns/list'),
                          ('drafts', 'content-desk/artifacts/list')):
        result = by_name[name]
        if isinstance(result, Exception):
            status, klass = program.classify(result)
            return fail(status, klass, result, 'collect')
        try:
            state[name] = result.head(ROW_CAP)
            state[name + '_total'] = result.count()
        except Exception as exc:
            status, klass = program.classify(exc)
            return fail(status, klass, exc, 'collect')
        envelope['evidence'].append(binding)

    # The capability catalog is optional: an unavailable catalog degrades the
    # board to partial with an explicit error rather than failing it.
    capabilities = by_name['capabilities']
    if isinstance(capabilities, Exception):
        status, klass = program.classify(capabilities)
        state['capabilities'] = []
        state['capabilities_total'] = 0
        state['capability_read_status'] = 'unavailable'
        envelope['errors'].append({'class': 'capabilities_unavailable',
                                   'detail': 'capability catalog read failed (%s/%s): %s' % (status, klass, capabilities),
                                   'where': 'collect'})
    else:
        try:
            state['capabilities'] = capabilities.head(ROW_CAP)
            state['capabilities_total'] = capabilities.count()
        except Exception as exc:
            status, klass = program.classify(exc)
            return fail(status, klass, exc, 'collect')
        state['capability_read_status'] = 'read'
        envelope['evidence'].append('content-desk/capabilities/list')

    # The offer release ladder is supplementary: an unavailable read degrades
    # the board to partial with an explicit error, never fails it.
    offer_ladder = by_name['offer_ladder']
    if isinstance(offer_ladder, Exception):
        status, klass = program.classify(offer_ladder)
        state['offer_ladder'] = []
        state['offer_ladder_total'] = 0
        state['offer_read_status'] = 'unavailable'
        envelope['errors'].append({'class': 'offer_readiness_unavailable',
                                   'detail': 'offer release ladder read failed (%s/%s): %s' % (status, klass, offer_ladder),
                                   'where': 'collect'})
    else:
        try:
            state['offer_ladder'] = offer_ladder.head(ROW_CAP)
            state['offer_ladder_total'] = offer_ladder.count()
        except Exception as exc:
            status, klass = program.classify(exc)
            return fail(status, klass, exc, 'collect')
        state['offer_read_status'] = 'read'
        envelope['evidence'].append('offer-desk/offers/release-ladder')

    state['truncated'] = any(state.get(name + '_total', 0) > ROW_CAP
                             for name in ('campaigns', 'drafts', 'capabilities', 'offer_ladder'))
    return 'classify'


def step_classify():
    # phase: classify — a deterministic table; no judgment is needed to derive next actions.
    envelope['phase'] = 'classify'

    campaign_rows = state.get('campaigns') or []
    draft_rows = state.get('drafts') or []
    capability_rows = state.get('capabilities') or []

    campaign_counts = {}
    campaign_snapshot = {}
    active_campaigns = []
    unknown_campaign_status = []
    campaign_title = {}
    for row in campaign_rows:
        campaign_id = _str(_field(row, 'id', 'id')).strip()
        if not campaign_id:
            continue
        status = _str(_field(row, 'status', 'status'))
        campaign_snapshot[campaign_id] = status
        campaign_title[campaign_id] = _str(_field(row, 'name', 'name'))
        campaign_counts[status] = campaign_counts.get(status, 0) + 1
        if status and status not in KNOWN_CAMPAIGN_STATUS:
            unknown_campaign_status.append({'id': campaign_id, 'status': status})
        if status == 'active':
            active_campaigns.append({
                'id': campaign_id,
                'name': campaign_title[campaign_id],
                'scenario_names': [_str(v) for v in (_field(row, 'scenarioNames', 'scenario_names') or [])],
            })

    draft_counts = {}
    draft_snapshot = {}
    drafts_by_campaign = {}
    unknown_draft_status = []
    current_work = []
    next_actions = []
    for row in draft_rows:
        draft_id = _str(_field(row, 'id', 'id')).strip()
        if not draft_id:
            continue
        status = _str(_field(row, 'status', 'status'))
        campaign_id = _str(_field(row, 'campaignId', 'campaign_id')).strip()
        draft_snapshot[draft_id] = status
        draft_counts[status] = draft_counts.get(status, 0) + 1
        drafts_by_campaign[campaign_id] = drafts_by_campaign.get(campaign_id, 0) + 1
        if status and status not in KNOWN_DRAFT_STATUS:
            unknown_draft_status.append({'draft_id': draft_id, 'status': status})
            continue
        action = NEXT_ACTION.get(status)
        if action:
            current_work.append({
                'draft_id': draft_id,
                'campaign_id': campaign_id,
                'channel': _str(_field(row, 'channel', 'channel')),
                'status': status,
                'next_action': action,
                'campaign_name': campaign_title.get(campaign_id, ''),
            })
            next_actions.append('%s: %s (%s)' % (action, draft_id, campaign_id or 'no campaign'))

    readiness = {dim: {} for dim in READINESS_DIMENSIONS}
    unknown_qualification = 0
    ownerless = 0
    with_limitations = 0
    limitations = []
    capability_next_actions = []
    for row in capability_rows:
        for dim in READINESS_DIMENSIONS:
            camel = ''.join(part.title() for part in dim.split('_'))
            camel = camel[0].lower() + camel[1:]
            value = _str(_field(row, camel, dim))
            readiness[dim][value] = readiness[dim].get(value, 0) + 1
        qualification = _field(row, 'latestQualification', 'latest_qualification')
        observed_at = _str(_field(qualification or {}, 'observedAt', 'observed_at')).strip().lower()
        try:
            max_age = int(_field(qualification or {}, 'maxAgeSeconds', 'max_age_seconds') or -1)
        except (TypeError, ValueError):
            max_age = -1
        if not qualification or observed_at in ('', 'unknown') or max_age < 0:
            unknown_qualification += 1
        if not _str(_field(row, 'owner', 'owner')).strip():
            ownerless += 1
        row_limits = _field(row, 'readinessLimitations', 'readiness_limitations') or []
        if row_limits:
            with_limitations += 1
            limitations.extend(row_limits)
        action = _str(_field(row, 'nextAction', 'next_action')).strip()
        if action:
            capability_next_actions.append(action)

    # Offer release readiness for the scenarios the active campaigns target.
    # The ladder owns ranking and goal state; this read only matches and reports.
    offer_rows = state.get('offer_ladder') or []
    launch_scenarios = sorted({name for campaign in active_campaigns
                               for name in (campaign.get('scenario_names') or []) if _str(name).strip()})
    shipped = ('TRIGGER_MET', 'RETIRED')
    launch_targets = []
    unknown_goal_state = 0
    for row in offer_rows:
        deliverable = _field(row, 'deliverable', 'deliverable') or {}
        name = _str(deliverable.get('name')).strip()
        if not name or name not in launch_scenarios:
            continue
        status = _str(deliverable.get('status'))
        reported = ('readinessGoalExists' in row) or ('readiness_goal_exists' in row)
        goal_exists = bool(_field(row, 'readinessGoalExists', 'readiness_goal_exists')) if reported else None
        goal_closed = bool(_field(row, 'readinessGoalClosed', 'readiness_goal_closed')) if reported else None
        blocked_by = []
        for enabler in row.get('enablers') or []:
            inner = (enabler.get('node') if isinstance(enabler, dict) else None) or enabler or {}
            enabler_status = _str(inner.get('status'))
            if enabler_status and enabler_status not in shipped:
                blocked_by.append({'name': _str(inner.get('name')), 'status': enabler_status})
        if not reported:
            unknown_goal_state += 1
        if status in shipped:
            next_action = 'Shipped; verify the release record.'
        elif not reported:
            next_action = 'Offer Desk did not report a readiness goal state; verify the release target there.'
        elif blocked_by:
            next_action = 'Clear the enabling prerequisites before release.'
        elif goal_exists and not goal_closed:
            next_action = 'Close the open offer readiness goal.'
        elif not goal_exists:
            next_action = 'Record the offer readiness goal.'
        else:
            next_action = 'Offer readiness goal is closed; ready for release.'
        launch_targets.append({
            'scenario': name,
            'node_id': _str(deliverable.get('id')),
            'status': status,
            'release_rank': int(deliverable.get('releaseRank', 0) or 0),
            'readiness_goal_reported': reported,
            'readiness_goal_exists': goal_exists,
            'readiness_goal_closed': goal_closed,
            'readiness_approved_commit': _str(_field(row, 'readinessApprovedCommit', 'readiness_approved_commit')),
            'blocked_by': blocked_by,
            'next_action': next_action,
        })

    baseline = state['baseline']
    changes = None
    if baseline.get('campaigns') or baseline.get('drafts'):
        changes = {
            'campaigns': _diff(campaign_snapshot, baseline.get('campaigns') or {}, campaign_title),
            'drafts': _diff(draft_snapshot, baseline.get('drafts') or {}, {}),
        }

    envelope['signals'] = {
        'campaign_total': state.get('campaigns_total', len(campaign_snapshot)),
        'campaign_status_counts': campaign_counts,
        'campaign_snapshot': campaign_snapshot,
        'active_campaigns': active_campaigns,
        'unknown_campaign_status': unknown_campaign_status,
        'draft_total': state.get('drafts_total', len(draft_snapshot)),
        'draft_status_counts': draft_counts,
        'draft_snapshot': draft_snapshot,
        'drafts_by_campaign': drafts_by_campaign,
        'unknown_draft_status': unknown_draft_status,
        'current_work': current_work,
        'next_actions': next_actions,
        'capability_total': state.get('capabilities_total', len(capability_rows)),
        'capability_readiness': readiness,
        'unknown_qualification': unknown_qualification,
        'ownerless': ownerless,
        'with_limitations': with_limitations,
        'limitations': _dedupe(limitations, 16),
        'capability_next_actions': _dedupe(capability_next_actions, 16),
        'offer_read_status': state.get('offer_read_status', 'unavailable'),
        'offer_readiness': {
            'ladder_entries': state.get('offer_ladder_total', len(offer_rows)),
            'launch_scenarios': launch_scenarios,
            'launch_targets': launch_targets,
            'unknown_goal_state': unknown_goal_state,
        },
        'changes_status': 'compared' if changes is not None else 'no_baseline',
        'changes': changes,
        'truncated': bool(state.get('truncated')),
        'capability_read_status': state.get('capability_read_status', 'unavailable'),
    }
    if state.get('truncated'):
        envelope['errors'].append({'class': 'row_cap_reached',
                                   'detail': 'an owner returned more rows than the materialize cap; report covers the first %d' % ROW_CAP,
                                   'where': 'classify'})
    if (state.get('truncated') or state.get('capability_read_status') == 'unavailable'
            or state.get('offer_read_status') == 'unavailable'):
        envelope['status'] = 'partial'
    else:
        envelope['status'] = 'ok'
    return 'report'


def _diff(snapshot, baseline, title_of):
    changes = {'added': [], 'removed': [], 'status_changed': []}
    for key, status in snapshot.items():
        was = baseline.get(key)
        if was is None:
            changes['added'].append({'id': key, 'title': title_of.get(key), 'status': status})
        elif was != status:
            changes['status_changed'].append({'id': key, 'title': title_of.get(key),
                                              'from': was, 'to': status})
    for key, status in baseline.items():
        if key not in snapshot:
            changes['removed'].append({'id': key, 'was': status})
    return changes


def step_report():
    # phase: report — one envelope on every path.
    envelope['phase'] = 'report'
    print(json.dumps(envelope, allow_nan=False, separators=(',', ':')))
    return None


STATES = {'validate': step_validate, 'collect': step_collect,
          'classify': step_classify, 'report': step_report}
program.run(STATES, 'validate')
