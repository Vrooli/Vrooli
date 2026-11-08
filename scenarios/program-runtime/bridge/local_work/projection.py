"""Strict local-work wake projection shared by the disabled adapter candidate.

Derived from the independently reviewed offline prototype, 7 November 2026.
No transport, credential lookup, subscriptions or lifecycle side effects.
"""
from __future__ import annotations
import hashlib
import json
import re
import sqlite3
from dataclasses import dataclass
from datetime import datetime, timezone


class Hold(ValueError):
    """A durable or input-level boundary requires attention, never silent success."""


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), allow_nan=False)


def digest(value):
    return hashlib.sha256(canonical(value).encode()).hexdigest()


def timestamp(value):
    if not isinstance(value, str):
        raise Hold('missing_timestamp')
    try:
        dt = datetime.fromisoformat(value.replace('Z', '+00:00'))
        if dt.tzinfo is None:
            raise ValueError('timezone required')
        return dt.timestamp()
    except ValueError as exc:
        raise Hold('invalid_timestamp') from exc


def opaque(value, name, maximum=256):
    if not isinstance(value, str) or not re.fullmatch(r'[A-Za-z0-9_.:/-]{1,%d}' % maximum, value):
        raise Hold('invalid_' + name)
    return value


def hash_value(value):
    if not isinstance(value, str) or not re.fullmatch('[0-9a-f]{64}', value):
        raise Hold('invalid_digest')
    return value


@dataclass(frozen=True)
class Policy:
    source_instance: str
    # Exact verified source/type pairs only. Empty by default, no wildcard.
    allowed_pairs: tuple[tuple[str, str], ...] = ()
    coalesce_seconds: int = 60
    max_delay_seconds: int = 300
    snapshot_max_age: int = 300
    producer_max_age: int = 86400
    event_max_age: int = 300
    max_batch: int = 64
    max_retained: int = 10000

    def __post_init__(self):
        opaque(self.source_instance, 'source_instance', 64)
        if not (0 <= self.coalesce_seconds <= self.max_delay_seconds <= 3600):
            raise Hold('invalid_window')
        if not (1 <= self.max_batch <= 64 and 1 <= self.max_retained <= 10000):
            raise Hold('invalid_bound')
        for value in (self.snapshot_max_age, self.producer_max_age, self.event_max_age):
            if type(value) is not int or not 1 <= value <= 604800:
                raise Hold('invalid_freshness_bound')
        if len(self.allowed_pairs) > 32 or len(set(self.allowed_pairs)) != len(self.allowed_pairs):
            raise Hold('invalid_allowlist')
        for source, kind in self.allowed_pairs:
            opaque(source, 'source', 64)
            opaque(kind, 'event_type', 256)
            if source not in {'agent-manager', 'prompt-manager', 'swarm-manager'} or kind == 'vrooli.events.receipt.v1':
                raise Hold('unsafe_event_family')


SOURCE_FAMILIES = {'runs', 'teams', 'objectives', 'team_execution', 'team_tasks', 'efforts',
                   'swarm_goals', 'swarm_backlog', 'swarm_executions', 'research_catalog',
                   'research_workspace', 'program_usage'}


def project_v2(receipt, now, policy):
    """Allowlist-only wake summary. Raw stdout, entities, paths and text never escape."""
    parent = receipt.get('program', {})
    if parent.get('status') != 'PROGRAM_STATUS_SUCCEEDED':
        raise Hold('collector_execution_not_succeeded')
    raw = parent.get('stdout')
    if not isinstance(raw, str) or len(raw.encode()) > 65536:
        raise Hold('collector_output_limit')
    try:
        body = json.loads(raw, parse_constant=lambda _: (_ for _ in ()).throw(Hold('invalid_json_number')))
    except (ValueError, TypeError) as exc:
        raise Hold('invalid_collector_json') from exc
    if not isinstance(body, dict) or body.get('program') != 'program-runtime.local-work-snapshot' or body.get('version') != '2':
        raise Hold('collector_contract_mismatch')
    if body.get('status') != 'partial' or body.get('phase') != 'report':
        raise Hold('collector_domain_not_report')
    signals = body.get('signals', {})
    started, finished = timestamp(signals.get('collected_at')), timestamp(signals.get('finished_at'))
    if finished < started or started > now + 30 or finished > now + 30:
        raise Hold('collector_clock_invalid')
    if now - started > policy.snapshot_max_age:
        raise Hold('collector_stale')
    if signals.get('coverage', {}).get('kind') != 'bounded_observation':
        raise Hold('invalid_coverage')
    sources = signals.get('sources')
    if not isinstance(sources, dict) or not 1 <= len(sources) <= 40:
        raise Hold('invalid_sources')
    summaries = []
    permission_hold = False
    for name, source in sorted(sources.items()):
        if not isinstance(source, dict) or name.split('/')[0] not in SOURCE_FAMILIES:
            raise Hold('invalid_source')
        state = source.get('status')
        if state not in {'ok', 'partial', 'unknown', 'unavailable', 'not_requested', 'not_collected', 'failed'}:
            raise Hold('invalid_source_status')
        collected = timestamp(source.get('collected_at'))
        if collected < started - 1 or collected > finished + 1:
            raise Hold('source_clock_invalid')
        producer = source.get('producer_observed_at')
        freshness = 'unknown'
        if producer is not None:
            producer_time = timestamp(producer)
            # Classification is as of the completed observation, not later staging.
            if producer_time > finished + 30:
                freshness = 'future_clock'
            elif finished - producer_time > policy.producer_max_age:
                freshness = 'stale'
            else:
                freshness = 'within_age_bound'
        errors = source.get('errors', [])
        if not isinstance(errors, list):
            raise Hold('invalid_source_errors')
        permission_hold |= any(isinstance(e, dict) and e.get('class') == 'no_grant' for e in errors)
        flags = {}
        for key in ('query_complete', 'output_complete'):
            value = source.get(key)
            if value is not None and type(value) is not bool:
                raise Hold('invalid_coverage_flag')
            flags[key] = value
        summaries.append({
            'family': name.split('/')[0],
            # Hashes are correlation labels, not a promise of anonymization.
            'scope_digest': digest({'name': name, 'scope': source.get('queried_scope')}),
            'status': state, 'collected_at': source['collected_at'],
            'producer_observed_at': producer, 'producer_freshness': freshness,
            **flags,
        })
    top_errors = body.get('errors', [])
    if not isinstance(top_errors, list):
        raise Hold('invalid_collector_errors')
    permission_hold |= any(isinstance(e, dict) and e.get('class') == 'no_grant' for e in top_errors)
    truncated = signals.get('output_truncated')
    if type(truncated) is not bool:
        raise Hold('missing_truncation_flag')
    return {
        'collector': 'program-runtime.local-work-snapshot', 'version': '2',
        'receipt_id': opaque(parent.get('id'), 'receipt_id'),
        'program_digest': hash_value(parent.get('program_digest')),
        'provenance': 'local_runtime_observation',
        'collected_at': signals['collected_at'], 'finished_at': signals['finished_at'],
        'domain_status': 'partial', 'coverage': 'bounded_observation',
        'output_truncated': truncated, 'permission_hold': permission_hold,
        'sources': summaries,
        'acceptance': 'unverified', 'native_liveness': 'unverified',
        'relations': 'not_inferred', 'effect_authority': 'none',
    }


def validate_projection(value, producer_max_age=86400):
    """Strict nested wire projection. Unknown fields never become a payload tunnel."""
    keys = {'collector','version','receipt_id','program_digest','provenance','collected_at','finished_at',
            'domain_status','coverage','output_truncated','permission_hold','sources','acceptance',
            'native_liveness','relations','effect_authority'}
    if not isinstance(value,dict) or set(value) != keys:
        raise Hold('invalid_snapshot_projection')
    expected = {'collector':'program-runtime.local-work-snapshot','version':'2',
                'provenance':'local_runtime_observation','domain_status':'partial','coverage':'bounded_observation',
                'acceptance':'unverified','native_liveness':'unverified','relations':'not_inferred','effect_authority':'none'}
    if any(value[k] != v for k,v in expected.items()):
        raise Hold('invalid_snapshot_projection')
    opaque(value['receipt_id'],'receipt_id'); hash_value(value['program_digest'])
    if any(type(value[k]) is not bool for k in ('output_truncated','permission_hold')):
        raise Hold('invalid_snapshot_flag')
    start,finish = timestamp(value['collected_at']),timestamp(value['finished_at'])
    if finish < start:
        raise Hold('invalid_snapshot_clock_order')
    sources=value['sources']
    if not isinstance(sources,list) or not 1 <= len(sources) <= 40:
        raise Hold('invalid_snapshot_sources')
    seen=set()
    source_keys={'family','scope_digest','status','collected_at','producer_observed_at','producer_freshness','query_complete','output_complete'}
    for source in sources:
        if not isinstance(source,dict) or set(source) != source_keys:
            raise Hold('invalid_source_projection')
        if source['family'] not in SOURCE_FAMILIES or source['status'] not in {'ok','partial','unknown','unavailable','not_requested','not_collected','failed'}:
            raise Hold('invalid_source_projection')
        hash_value(source['scope_digest'])
        scope=(source['family'],source['scope_digest'])
        if scope in seen:
            raise Hold('duplicate_source_scope')
        seen.add(scope)
        if any(source[k] is not None and type(source[k]) is not bool for k in ('query_complete','output_complete')):
            raise Hold('invalid_source_coverage_flag')
        if not start - 1 <= timestamp(source['collected_at']) <= finish + 1:
            raise Hold('invalid_source_clock')
        producer=source['producer_observed_at']
        freshness=source['producer_freshness']
        if producer is None:
            if freshness != 'unknown':
                raise Hold('invalid_unknown_freshness')
        else:
            pt=timestamp(producer)
            if freshness not in {'future_clock','stale','within_age_bound'}:
                raise Hold('invalid_producer_freshness')
            if freshness=='within_age_bound' and (pt > finish+30 or finish-pt > producer_max_age):
                raise Hold('false_freshness')
