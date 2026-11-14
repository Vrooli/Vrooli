"""Disabled native adapter/exporter candidate; no listener, daemon or CLI entry point.

All live boundaries are injected. No environment, credential file, owner database,
subscription, service, or network is touched on import or by offline preparation.
A managed host integration and separately approved runtime grant are still needed.
"""
from __future__ import annotations

import base64
import hashlib
import hmac
import json
import os
import re
import sqlite3
import stat
import urllib.error
import urllib.parse
import urllib.request
from email.utils import parsedate_to_datetime
from contextlib import contextmanager
from dataclasses import dataclass, asdict
from typing import Callable

from .projection import Hold, Policy, canonical, digest, opaque, timestamp, project_v2, validate_projection

PATH = '/api/bridge/local-work-events'
ORIGIN = 'https://decision-inbox-experiment.mdhalloran.chatgpt.site'
VERIFIED_PAIRS = (
    ('agent-manager', 'agent_manager.run.notification.v1'),
    ('agent-manager', 'agent_manager.run.decision_requested.v1'),
    ('agent-manager', 'agent_manager.run.decision_withdrawn.v1'),
)
GAPS = {'unknown_checkpoint', 'retention_gap', 'replay_limit', 'dropped_events', 'source_reset'}


def strict_json(raw: bytes, maximum: int):
    if not isinstance(raw, bytes) or not 0 < len(raw) <= maximum:
        raise Hold('body_size')
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise Hold('duplicate_json_key')
            result[key] = value
        return result
    try:
        return json.loads(raw.decode('utf-8'), object_pairs_hook=pairs,
                          parse_constant=lambda _: (_ for _ in ()).throw(Hold('nonfinite_json')))
    except (UnicodeError, ValueError, RecursionError) as exc:
        raise Hold('invalid_json') from exc


def headers_lower(headers):
    result = {}
    for key, value in headers.items():
        key = key.lower()
        if key in result or not isinstance(value, str):
            raise Hold('ambiguous_header')
        result[key] = value
    return result


@dataclass(frozen=True)
class AdapterPolicy:
    # This stable identity is configuration, not a hostname inferred from a packet.
    source_instance: str = 'swarminator'
    source_epoch: str = 'uncommissioned'
    allowed_pairs: tuple = ()
    max_retained: int = 10000
    quiet_seconds: int = 60
    max_delay_seconds: int = 300
    max_batch: int = 64
    snapshot_max_age: int = 300
    producer_max_age: int = 86400
    event_max_age: int = 300

    def __post_init__(self):
        opaque(self.source_epoch, 'source_epoch', 64)
        if any(type(getattr(self, field)) is not int for field in ('max_retained','quiet_seconds','max_delay_seconds','max_batch','snapshot_max_age','producer_max_age','event_max_age')):
            raise Hold('invalid_integer_policy')
        if type(self.allowed_pairs) is not tuple or any(type(pair) is not tuple for pair in self.allowed_pairs):
            raise Hold('mutable_allowlist_forbidden')
        if any(pair not in VERIFIED_PAIRS for pair in self.allowed_pairs):
            raise Hold('unverified_event_pair')
        self.projection_policy()

    def projection_policy(self):
        return Policy(self.source_instance, self.allowed_pairs, self.quiet_seconds,
                      self.max_delay_seconds, self.snapshot_max_age, self.producer_max_age,
                      self.event_max_age, self.max_batch, self.max_retained)


@dataclass(frozen=True)
class RuntimeGrant:
    """Host-owned approved configuration, never inferred from source payloads.

    This is a local runtime guard, not a replacement for action-time user approval.
    No grant is shipped/configured by this task. Keys are held only by the caller.
    """
    enabled: bool = False
    approval_ref: str = ''
    source_instance: str = 'swarminator'
    source_epoch: str = 'uncommissioned'
    origin: str = ORIGIN
    key_id: str = ''
    not_before: int = 0
    expires_at: int = 0

    def validate(self, policy, now):
        if self.enabled is not True:
            raise Hold('transport_disabled')
        opaque(self.approval_ref, 'approval_ref')
        opaque(self.key_id, 'key_id', 128)
        if self.origin != ORIGIN or self.source_instance != policy.source_instance or self.source_epoch != policy.source_epoch:
            raise Hold('grant_scope_mismatch')
        if type(self.not_before) is not int or type(self.expires_at) is not int or not 0 < self.expires_at-self.not_before <= 604800:
            raise Hold('invalid_grant_lifetime')
        if not self.not_before <= now < self.expires_at:
            raise Hold('grant_not_current')
        if not policy.allowed_pairs or policy.source_epoch == 'uncommissioned':
            raise Hold('uncommissioned_source')


def signed_headers(body: bytes, grant: RuntimeGrant, key: bytes, now: int):
    """Pure signing function. Call only with an already-approved injected key."""
    if not isinstance(key, bytes) or len(key) != 32:
        raise Hold('invalid_signing_key')
    ts = str(int(now))
    prefix = f'flywheel.local-work-http.v1\n{ts}\nPOST\n{grant.origin}{PATH}\n'.encode()
    signature = base64.b64encode(hmac.new(key, prefix+body, hashlib.sha256).digest()).decode()
    return {'Content-Type':'application/json', 'X-Flywheel-Bridge-Key-Id':grant.key_id,
            'X-Flywheel-Bridge-Timestamp':ts, 'X-Flywheel-Bridge-Signature':'v1,'+signature}


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class HTTPSender:
    """Optional single-call HTTP boundary; not instantiated or invoked automatically.

    No proxies, cookies, redirects, client sessions, environment credentials, or
    fallback hosts. Standard library TLS verifies the exact hostname certificate.
    Site private access must independently admit this endpoint after approval.
    """
    def __call__(self, url, headers, body):
        if url != ORIGIN + PATH:
            raise Hold('destination_mismatch')
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
        request = urllib.request.Request(url, data=body, headers=headers, method='POST')
        try:
            response = opener.open(request, timeout=10)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            data = response.read(8193)
            if len(data) > 8192:
                raise Hold('response_size')
            return response.status, dict(response.headers.items()), data


def authenticate_native(raw,headers,local_key):
    """Authenticate and validate native input before opening durable storage."""
    headers = headers_lower(headers)
    if not isinstance(raw, bytes) or not 0 < len(raw) <= 1048576:
        raise Hold('native_body_size')
    if not isinstance(local_key, bytes) or not local_key:
        raise Hold('native_signature_required')
    expected = 'sha256='+hmac.new(local_key,raw,hashlib.sha256).hexdigest()
    supplied = headers.get('x-vrooli-events-signature','')
    if not re.fullmatch('sha256=[0-9a-f]{64}', supplied) or not hmac.compare_digest(expected,supplied):
        raise Hold('native_signature_invalid')
    body = strict_json(raw,1048576)
    required={'event_id','event_type','source_scenario','delivered_at'}
    if not isinstance(body,dict) or not required <= set(body) or set(body)-required-{'payload'}:
        raise Hold('native_webhook_contract')
    event_id=opaque(body['event_id'],'event_id')
    if headers.get('x-vrooli-events-event-id') != event_id or headers.get('x-vrooli-events-idempotency-key') != event_id:
        raise Hold('native_identity_header_mismatch')
    timestamp(body['delivered_at'])
    return body


def validated_snapshot(receipt,now,policy):
    """One strict snapshot input boundary shared by preparation and recovery."""
    parent=receipt.get('program',{}) if isinstance(receipt,dict) else {}
    raw=parent.get('stdout') if isinstance(parent,dict) else None
    if not isinstance(raw,str):raise Hold('invalid_collector_receipt')
    strict_json(raw.encode(),65536)
    try:
        snapshot=project_v2(receipt,now,policy)
        validate_projection(snapshot,policy.producer_max_age)
        return snapshot
    except (TypeError,KeyError,AttributeError) as exc:
        raise Hold('invalid_collector_receipt') from exc


class Outbox:
    """Durable metadata-only local owner. BEGIN IMMEDIATE serializes writers.

    A native acknowledgment is safe only after accept_native returns. Source data
    stays in its owner: pending resolution survives collection failures/restarts.
    No event ID or receipt is pruned automatically; capacity is explicit pressure.
    """
    def __init__(self, path, policy=AdapterPolicy()):
        self.policy = policy
        if path != ':memory:':
            parent = os.path.dirname(os.path.abspath(path))
            info = os.stat(parent, follow_symlinks=False)
            if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
                raise Hold('outbox_requires_private_owner_directory')
            flags = os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW
            fd = os.open(path, flags, 0o600)
            try:
                info = os.fstat(fd)
                if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077 or info.st_nlink != 1:
                    raise Hold('outbox_requires_private_regular_file')
            finally:
                os.close(fd)
        self.db = sqlite3.connect(path, timeout=5, isolation_level=None)
        self.db.row_factory = sqlite3.Row
        existing=self.db.execute("SELECT name FROM sqlite_master WHERE type='table'").fetchall()
        application_id=self.db.execute('PRAGMA application_id').fetchone()[0]
        if existing and application_id!=0x56524C42:
            self.db.close()
            raise Hold('outbox_database_not_owned')
        self.db.execute('PRAGMA application_id=1448234050')
        self.db.execute('PRAGMA journal_mode=DELETE')
        self.db.execute('PRAGMA synchronous=FULL')
        self.db.execute('PRAGMA foreign_keys=ON')
        self.db.executescript('''
          CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
          CREATE TABLE IF NOT EXISTS native_events(
            event_id TEXT PRIMARY KEY, event_type TEXT NOT NULL, source TEXT NOT NULL,
            webhook_digest TEXT NOT NULL, received REAL NOT NULL,
            state TEXT NOT NULL, event_projection TEXT, snapshot TEXT, batch_id TEXT);
          CREATE TABLE IF NOT EXISTS holds(kind TEXT PRIMARY KEY, observed REAL NOT NULL);
          CREATE TABLE IF NOT EXISTS batches(
            batch_id TEXT PRIMARY KEY, body BLOB NOT NULL, state TEXT NOT NULL,
            attempts INTEGER NOT NULL DEFAULT 0, due REAL NOT NULL,
            lease_until REAL, receipt_id TEXT, last_category TEXT);
        ''')
        try:
            with self.transaction():
                identity = canonical({'version':1, 'policy':asdict(policy)})
                old = self.db.execute("SELECT value FROM meta WHERE key='identity'").fetchone()
                if old and old['value'] != identity:
                    raise Hold('policy_or_source_reset_requires_migration')
                self.db.execute("INSERT OR IGNORE INTO meta VALUES ('identity',?)", (identity,))
                # No numeric cursor can prove earlier events were observed. No auto-clear.
                self.db.execute("INSERT OR IGNORE INTO holds VALUES ('unknown_checkpoint',0)")
        except Exception:
            self.db.close()
            raise

    @contextmanager
    def transaction(self):
        self.db.execute('BEGIN IMMEDIATE')
        try:
            yield
            self.db.commit()
        except Exception:
            self.db.rollback()
            raise

    def close(self):
        self.db.close()

    def mark_gap(self, kind, now, *, _collection_generation=None):
        if kind not in GAPS:
            raise Hold('unknown_gap')
        with self.transaction():
            if _collection_generation is not None and not self._owns_collection(_collection_generation):
                raise Hold('collection_lease_lost')
            self.db.execute('INSERT OR IGNORE INTO holds VALUES (?,?)',(kind,now))

    def accept_native(self, raw, headers, local_key, now):
        """Authenticate actual Vrooli Events WebhookPayload, then commit before ACK.

        The existing fleet key must be injected only inside its native boundary.
        It is never copied to cloud, persisted here, read from env, or logged.
        delivered_at is delivery time, never event occurrence or complete cursor.
        """
        body=authenticate_native(raw,headers,local_key)
        event_id=body['event_id']
        if (body['source_scenario'],body['event_type']) not in self.policy.allowed_pairs:
            return {'state':'ignored','reason':'outside_exact_allowlist'}
        # DeliveredAt can change for a one-shot retry; event payload identity cannot.
        identity=digest({k:v for k,v in body.items() if k!='delivered_at'})
        conflict=False
        with self.transaction():
            old=self.db.execute('SELECT * FROM native_events WHERE event_id=?',(event_id,)).fetchone()
            if old:
                if old['webhook_digest'] != identity:
                    self.db.execute("INSERT OR IGNORE INTO holds VALUES ('event_identity_conflict',?)",(now,))
                    conflict=True
                else:
                    return {'state':'duplicate','original_state':old['state']}
            else:
                if self.db.execute('SELECT count(*) FROM native_events').fetchone()[0] >= self.policy.max_retained:
                    raise Hold('native_capacity_backpressure')
                self.db.execute('INSERT INTO native_events VALUES (?,?,?,?,?,?,NULL,NULL,NULL)',
                    (event_id,body['event_type'],body['source_scenario'],identity,now,'awaiting_envelope'))
        if conflict:
            raise Hold('event_identity_conflict')
        return {'state':'durable_pending_resolution','event_id':event_id}

    def qualify_event(self, event_id, envelope, now, *, _collection_generation=None):
        """Resolve only exact IDs through the existing Events owner, never payload text.

        The host resolver must use GET /api/v1/events?event_id=<ID>&limit=2 and
        require exactly one canonical envelope. That connection is not installed.
        A missing retained record leaves a gap, never substitutes delivered_at.
        """
        if envelope is None:
            self.mark_gap('retention_gap',now,_collection_generation=_collection_generation)
            raise Hold('canonical_event_unavailable')
        if not isinstance(envelope,dict) or len(canonical(envelope).encode())>1048576:
            raise Hold('canonical_envelope_invalid')
        row=self.db.execute('SELECT * FROM native_events WHERE event_id=?',(event_id,)).fetchone()
        if not row:
            raise Hold('unknown_native_event')
        if not isinstance(envelope.get('source'),dict) or ('target' in envelope and not isinstance(envelope['target'],dict)):
            raise Hold('canonical_envelope_invalid')
        if (envelope.get('eventId'),envelope.get('eventType'),envelope.get('source',{}).get('scenario')) != (event_id,row['event_type'],row['source']):
            raise Hold('canonical_identity_mismatch')
        occurrence=timestamp(envelope.get('occurredAt'))
        if occurrence>now+30:
            raise Hold('canonical_future_clock')
        projection={'event_id':event_id,'event_type':row['event_type'],'source':row['source'],
                    'occurred_at':envelope['occurredAt'],'source_content_digest':digest(envelope)}
        conflict=False
        with self.transaction():
            if _collection_generation is not None and not self._owns_collection(_collection_generation):
                raise Hold('collection_lease_lost')
            row=self.db.execute('SELECT * FROM native_events WHERE event_id=?',(event_id,)).fetchone()
            if row['event_projection']:
                if row['event_projection'] != canonical(projection):
                    self.db.execute("INSERT OR IGNORE INTO holds VALUES ('event_identity_conflict',?)",(now,))
                    conflict=True
                else:
                    return {'state':row['state']}
            else:
                state='resolved'
                if envelope.get('target',{}).get('scenario') in {'program-runtime','flywheel','review-inbox'}:
                    state='ignored_feedback'
                if now-occurrence>self.policy.event_max_age:
                    self.db.execute("INSERT OR IGNORE INTO holds VALUES ('retention_gap',?)",(now,))
                self.db.execute('UPDATE native_events SET event_projection=?,state=? WHERE event_id=?',
                                (canonical(projection),state,event_id))
        if conflict:
            raise Hold('event_identity_conflict')
        return {'state':state}

    def due_ids(self, now):
        rows=self.db.execute("SELECT event_id,received FROM native_events WHERE state='resolved' ORDER BY received,event_id LIMIT ?",(self.policy.max_batch,)).fetchall()
        if not rows:
            return []
        newest=self.db.execute("SELECT max(received) FROM native_events WHERE state='resolved'").fetchone()[0]
        if len(rows)<self.policy.max_batch and now<min(rows[0]['received']+self.policy.max_delay_seconds,newest+self.policy.quiet_seconds):
            return []
        return [row['event_id'] for row in rows]

    def prepare(self, event_ids, receipt, now, *, _collection_generation=None):
        """Freeze one fresh snapshot and immutable packet after coalescing.

        Only known selected native events can trigger collection; a fresh read
        without such an event is not new work. Never suppress a distinct event
        merely because a bounded snapshot sample looks unchanged.
        """
        if not isinstance(event_ids,list) or not 1<=len(event_ids)<=self.policy.max_batch or len(set(event_ids))!=len(event_ids):
            raise Hold('invalid_preparation_scope')
        snapshot=validated_snapshot(receipt,now,self.policy.projection_policy())
        with self.transaction():
            if _collection_generation is not None and not self._owns_collection(_collection_generation):
                raise Hold('collection_lease_lost')
            rows=[]
            for eid in sorted(event_ids):
                row=self.db.execute('SELECT * FROM native_events WHERE event_id=?',(eid,)).fetchone()
                if not row or row['state']!='resolved':
                    raise Hold('preparation_requires_resolved')
                rows.append(row)
            if snapshot['permission_hold']:
                self.db.execute("INSERT OR IGNORE INTO holds VALUES ('permission_hold',?)",(now,))
            holds={r[0] for r in self.db.execute('SELECT kind FROM holds')}
            snapshot['permission_hold'] |= 'permission_hold' in holds
            gap=next((g for g in ('source_reset','dropped_events','replay_limit','retention_gap','unknown_checkpoint') if g in holds),None)
            if 'event_identity_conflict' in holds:
                raise Hold('unresolved_identity_conflict')
            body={'schema':'flywheel.local-work-wake.v1','source_instance':self.policy.source_instance,
                  'events':[json.loads(row['event_projection']) for row in rows], 'snapshot':snapshot,
                  'coverage_gap':gap, 'purpose':'request_read_only_reconciliation',
                  'activation':'not_approved','effect_authority':'none'}
            batch_id=digest(body); body['batch_id']=batch_id
            frozen=canonical(body).encode()
            if len(frozen)>65536:
                raise Hold('cloud_body_size')
            self.db.execute("INSERT INTO batches(batch_id,body,state,due) VALUES (?,?,'prepared',?)",(batch_id,frozen,now))
            self.db.executemany("UPDATE native_events SET state='batched',snapshot=?,batch_id=? WHERE event_id=?",
                                [(canonical(snapshot),batch_id,row['event_id']) for row in rows])
            if _collection_generation is not None:
                # A crash after durable batch commit must not retain prior failures.
                self.db.execute("DELETE FROM meta WHERE key='collection_retry'")
            active=self.db.execute("SELECT value FROM meta WHERE key='active_collection_ids'").fetchone()
            if active and set(json.loads(active['value']))==set(event_ids):
                self.db.execute("DELETE FROM meta WHERE key='active_collection_ids'")
        return body

    def _owns_collection(self, generation):
        row=self.db.execute("SELECT value FROM meta WHERE key='collection_lease'").fetchone()
        return bool(row and json.loads(row['value'])['generation']==generation)

    def collect_once(self, resolver: Callable, collector: Callable, now, *, clock=None):
        """One bounded, fenced host-scheduled cycle; never starts a scheduler.

        A 300-second durable lease allows only one collector. The managed host
        must bound the complete resolver/collector cycle below that lease. A
        superseded worker cannot prepare a batch or overwrite the winner's state.
        Callbacks are fixed owner operations, never source-provided commands.
        """
        current_time=clock or (lambda:now)
        with self.transaction():
            state=self.db.execute("SELECT value FROM meta WHERE key='collection_retry'").fetchone()
            retry=json.loads(state['value']) if state else {'attempts':0,'due':0,'held':False}
            if retry['held']:
                return {'state':'collection_held','attempts':retry['attempts']}
            if now<retry['due']:
                return {'state':'collection_not_due','next_attempt_at':retry['due']}
            current=self.db.execute("SELECT value FROM meta WHERE key='collection_lease'").fetchone()
            if current and json.loads(current['value'])['until']>now:
                return {'state':'collection_busy'}
            previous=self.db.execute("SELECT value FROM meta WHERE key='collection_generation'").fetchone()
            generation=int(previous['value'])+1 if previous else 1
            self.db.execute("INSERT OR REPLACE INTO meta VALUES ('collection_generation',?)",(str(generation),))
            self.db.execute("INSERT OR REPLACE INTO meta VALUES ('collection_lease',?)",(canonical({'generation':generation,'until':now+300}),))
        try:
            unresolved=self.db.execute("SELECT event_id FROM native_events WHERE state='awaiting_envelope' ORDER BY received,event_id LIMIT ?",(self.policy.max_batch,)).fetchall()
            for row in unresolved:
                self.qualify_event(row['event_id'],resolver(row['event_id']),current_time(),_collection_generation=generation)
            with self.transaction():
                if not self._owns_collection(generation):
                    raise Hold('collection_lease_lost')
                active=self.db.execute("SELECT value FROM meta WHERE key='active_collection_ids'").fetchone()
                due=json.loads(active['value']) if active else self.due_ids(current_time())
                if due and not active:
                    self.db.execute("INSERT INTO meta VALUES ('active_collection_ids',?)",(canonical(due),))
            if due:
                receipt=collector.collect_for(due,current_time(),collection_generation=generation) if hasattr(collector,'collect_for') else collector()
                body=self.prepare(due,receipt,current_time(),_collection_generation=generation)
            with self.transaction():
                if not self._owns_collection(generation):
                    return {'state':'collection_superseded'}
                self.db.execute("DELETE FROM meta WHERE key='collection_lease'")
                if due:
                    self.db.execute("DELETE FROM meta WHERE key='collection_retry'")
            return {'state':'prepared','batch_id':body['batch_id'],'events':len(body['events'])} if due else {'state':'no_collection_due'}
        except Exception as failure:
            with self.transaction():
                if not self._owns_collection(generation):
                    return {'state':'collection_superseded'}
                if isinstance(failure,Hold) and str(failure)=='owner_permission_hold':
                    self.db.execute("INSERT OR IGNORE INTO holds VALUES ('permission_hold',?)",(current_time(),))
                attempt=retry['attempts']+1
                state={'attempts':attempt,'due':current_time()+min(300,2**attempt),'held':attempt>=5}
                self.db.execute("INSERT OR REPLACE INTO meta VALUES ('collection_retry',?)",(canonical(state),))
                self.db.execute("DELETE FROM meta WHERE key='collection_lease'")
            return {'state':'collection_held' if state['held'] else 'collection_retry',
                    'attempts':attempt,'next_attempt_at':None if state['held'] else state['due']}

    def pending(self):
        return [dict(r) for r in self.db.execute('SELECT event_id,state FROM native_events ORDER BY received,event_id')]

    def preview(self, batch_id):
        row=self.db.execute('SELECT body FROM batches WHERE batch_id=?',(batch_id,)).fetchone()
        if not row: raise Hold('unknown_batch')
        return bytes(row['body'])

    def export_once(self, grant, key, now, transport: Callable):
        """One bounded delivery attempt. Never starts a loop or auto-enables anything.

        The caller supplies current time on every wake. due survives restart;
        interrupted delivery remains uncertain and retries the exact frozen body.
        No completion/accepted-work claim follows from a transport receipt.
        """
        grant.validate(self.policy,now)
        if not isinstance(key,bytes) or len(key)!=32:
            raise Hold('invalid_signing_key')
        with self.transaction():
            if self.db.execute("SELECT 1 FROM holds WHERE kind='event_identity_conflict'").fetchone():
                raise Hold('unresolved_identity_conflict')
            # Five dispatched attempts, including a crash, is a hard reviewed hold.
            self.db.execute("UPDATE batches SET state='held',last_category='attempts_exhausted' WHERE state='sending' AND lease_until<=? AND attempts>=5",(now,))
            row=self.db.execute("SELECT * FROM batches WHERE (state IN ('prepared','retry') AND due<=?) OR (state='sending' AND lease_until<=? AND attempts<5) ORDER BY rowid LIMIT 1",(now,now)).fetchone()
            if not row:
                return {'state':'not_due'}
            batch_id=row['batch_id']; body=bytes(row['body']); attempt=row['attempts']+1
            self.db.execute("UPDATE batches SET state='sending',attempts=?,lease_until=?,last_category='outcome_unknown' WHERE batch_id=?",(attempt,now+30,batch_id))
        headers=signed_headers(body,grant,key,int(now))
        try:
            status,response_headers,response=transport(grant.origin+PATH,headers,body)
        except Exception:
            return self._finish_failure(batch_id,attempt,'uncertain_transport',now,retry=True)
        if status in (200,201):
            try:
                result=strict_json(response,8192)
                required={'schema','batch_id','receipt_id','state','new_events','duplicate'}
                if not isinstance(result,dict) or set(result)!=required or result['schema']!='flywheel.local-work-receipt.v1' or result['batch_id']!=batch_id:
                    raise Hold('invalid_cloud_receipt')
                opaque(result['receipt_id'],'receipt_id')
                packet=json.loads(body)
                if result['duplicate'] is True and result['new_events'] != 0:
                    raise Hold('invalid_duplicate_receipt')
                if packet['coverage_gap'] is not None and result['state']!='gap_hold':
                    raise Hold('cloud_receipt_lost_gap')
                if packet['coverage_gap'] is None and packet['snapshot']['permission_hold'] and result['state'] not in {'permission_hold','gap_hold'}:
                    raise Hold('cloud_receipt_lost_permission')
                if result['state'] not in {'needs_source_read','permission_hold','gap_hold'} or type(result['duplicate']) is not bool or type(result['new_events']) is not int or not 0<=result['new_events']<=len(json.loads(body)['events']):
                    raise Hold('invalid_cloud_receipt')
            except (Hold,KeyError,TypeError):
                return self._finish_failure(batch_id,attempt,'uncertain_receipt',now,retry=True)
            with self.transaction():
                current=self.db.execute('SELECT state,attempts,receipt_id FROM batches WHERE batch_id=?',(batch_id,)).fetchone()
                if current['attempts']!=attempt or current['state']!='sending':
                    raise Hold('attempt_lease_lost')
                self.db.execute("UPDATE batches SET state='transport_accepted',receipt_id=?,lease_until=NULL,last_category=NULL WHERE batch_id=?",(result['receipt_id'],batch_id))
                self.db.execute("UPDATE native_events SET state='transport_accepted' WHERE batch_id=?",(batch_id,))
            return {'state':'transport_accepted','receipt':result,'handler_pickup':'unverified','reconciliation':'unverified'}
        retry=status in {408,425,429,500,502,503,504}
        category='rate_limited' if status==429 else 'transient_http' if retry else 'permission_denied' if status in {401,403,404} else 'receiver_hold'
        retry_at=None
        if retry:
            try:
                value=headers_lower(response_headers).get('retry-after')
                if value is not None:
                    delay=int(value) if re.fullmatch(r'[0-9]+',value) else parsedate_to_datetime(value).timestamp()-now
                    # Do not retry earlier than requested, even across process restart.
                    if delay>86400:
                        retry=False; category='retry_after_requires_review'
                    else:
                        retry_at=now+max(0,delay)
            except (ValueError,TypeError,OverflowError,Hold):
                retry=False; category='invalid_retry_after'
        return self._finish_failure(batch_id,attempt,category,now,retry=retry,retry_at=retry_at)

    def _finish_failure(self,batch_id,attempt,category,now,retry,retry_at=None):
        state='retry' if retry and attempt<5 else 'held'
        # Small deterministic jitter, stable through restart; no busy looping.
        jitter=int(batch_id[:4],16)%4
        due=max(now+min(300,2**attempt)+jitter,retry_at or now)
        with self.transaction():
            row=self.db.execute('SELECT state,attempts FROM batches WHERE batch_id=?',(batch_id,)).fetchone()
            if row['state']!='sending' or row['attempts']!=attempt:
                raise Hold('attempt_lease_lost')
            self.db.execute('UPDATE batches SET state=?,due=?,lease_until=NULL,last_category=? WHERE batch_id=?',(state,due,category,batch_id))
        return {'state':state,'category':category,'attempts':attempt,'next_attempt_at':due if state=='retry' else None}
