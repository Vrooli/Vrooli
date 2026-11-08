"""Exact, bounded owner API clients. No request is made until explicitly invoked.

There is no credential discovery or shell invocation. Approved principal headers,
loopback origins, transport and current clock are supplied by the managed host.
"""
from __future__ import annotations
import hashlib
import ipaddress
import json
import urllib.parse
import urllib.request
import urllib.error
from dataclasses import dataclass,field
from contextlib import contextmanager
from datetime import datetime,timezone
from .adapter import Hold,Outbox,NoRedirect,canonical,digest,opaque,strict_json
from .projection import hash_value

NAME='program-runtime.local-work-snapshot'
LIB='/vrooli.program_runtime.v1.library.LibraryService/'
PROGRAMS='/vrooli.program_runtime.v1.programs.ProgramService/'
INPUTS={'effort_page_token','research_root','row_limit','run_offsets','run_statuses',
        'sample_offsets','team_ids','team_task_offsets','usage_window_days'}
TERMINAL={'PROGRAM_STATUS_SUCCEEDED','PROGRAM_STATUS_FAILED','PROGRAM_STATUS_CANCELLED'}


def loopback_origin(value):
    if not isinstance(value,str): raise Hold('invalid_owner_origin')
    try:
        parsed=urllib.parse.urlsplit(value)
        ip=ipaddress.ip_address(parsed.hostname or '')
        port=parsed.port
    except (ValueError,TypeError) as exc: raise Hold('invalid_owner_origin') from exc
    if parsed.scheme!='http' or not ip.is_loopback or not port or not 1<=port<=65535 or parsed.path or parsed.query or parsed.fragment or parsed.username or parsed.password:
        raise Hold('owner_origin_must_be_exact_numeric_loopback')
    return value


@dataclass(frozen=True)
class OwnerConfig:
    events_origin:str
    runtime_origin:str
    expected_digest:str
    inputs_json:str=field(default='{}',repr=False)
    enabled:bool=False
    approval_ref:str=''
    not_before:int=0
    expires_at:int=0

    def __post_init__(self):
        loopback_origin(self.events_origin);loopback_origin(self.runtime_origin);hash_value(self.expected_digest)
        value=strict_json(self.inputs_json.encode(),16384)
        if not isinstance(value,dict) or set(value)-INPUTS: raise Hold('collector_scope_invalid')
        if canonical(value)!=self.inputs_json: raise Hold('collector_inputs_not_canonical')

    def authorize(self,now):
        if self.enabled is not True: raise Hold('owner_access_disabled')
        opaque(self.approval_ref,'owner_approval_ref')
        if type(self.not_before) is not int or type(self.expires_at) is not int or not 0<self.expires_at-self.not_before<=604800 or not self.not_before<=now<self.expires_at:
            raise Hold('owner_access_not_current')

    def fingerprint(self):
        return digest({'events_origin':self.events_origin,'runtime_origin':self.runtime_origin,
                       'name':NAME,'expected_digest':self.expected_digest,'inputs':json.loads(self.inputs_json)})


class LoopbackTransport:
    """Optional concrete transport. Never called in qualification; no proxy/redirect."""
    def __call__(self,method,url,headers,body,timeout,maximum):
        parsed=urllib.parse.urlsplit(url)
        loopback_origin(urllib.parse.urlunsplit((parsed.scheme,parsed.netloc,'','','')))
        opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),NoRedirect())
        request=urllib.request.Request(url,data=body,headers=headers,method=method)
        try: response=opener.open(request,timeout=timeout)
        except urllib.error.HTTPError as error: response=error
        with response:
            raw=response.read(maximum+1)
            if len(raw)>maximum: raise Hold('owner_response_size')
            return response.status,raw


class OwnerClient:
    def __init__(self,config,transport,authorization,clock):
        self.config=config;self.transport=transport;self.authorization=authorization;self.clock=clock;self.deadline=None

    @contextmanager
    def bounded_cycle(self,seconds=180):
        if not 1<=seconds<=180:raise Hold('invalid_owner_cycle_budget')
        previous=self.deadline;self.deadline=self.clock()+seconds
        try:yield
        finally:self.deadline=previous

    def request(self,owner,method,path,body=None,timeout=10,maximum=1048576):
        self.config.authorize(self.clock())
        if self.deadline is None:raise Hold('owner_cycle_budget_required')
        remaining=self.deadline-self.clock()
        if remaining<=0:raise Hold('owner_cycle_deadline')
        timeout=min(timeout,remaining)
        if owner not in {'events','runtime'}: raise Hold('unknown_owner')
        if (owner=='events' and (method!='GET' or not path.startswith('/api/v1/events?event_id='))) or (owner=='runtime' and (method!='POST' or path not in {LIB+'GetDeclaredExecution',LIB+'RunDeclaredProgram',PROGRAMS+'WaitForProgram'})):
            raise Hold('operation_not_allowlisted')
        authorization=self.authorization(owner)
        if not isinstance(authorization,str) or not authorization.startswith('Bearer ') or any(c in authorization for c in '\r\n'):
            raise Hold('approved_owner_principal_required')
        headers={'Authorization':authorization,'Content-Type':'application/json','Connect-Protocol-Version':'1'}
        origin=self.config.events_origin if owner=='events' else self.config.runtime_origin
        code,raw=self.transport(method,origin+path,headers,None if body is None else canonical(body).encode(),timeout,maximum)
        if self.clock()>self.deadline:raise Hold('owner_cycle_deadline')
        if code in {401,403}: raise Hold('owner_permission_hold')
        if code!=200: raise Hold('owner_unavailable')
        return strict_json(raw,maximum)

    def resolve(self,event_id):
        opaque(event_id,'event_id')
        query=urllib.parse.urlencode({'event_id':event_id,'limit':2})
        rows=self.request('events','GET','/api/v1/events?'+query)
        if not isinstance(rows,list) or len(rows)>1: raise Hold('canonical_query_not_exact')
        if not rows:return None
        if not isinstance(rows[0],dict) or rows[0].get('eventId')!=event_id: raise Hold('canonical_identity_mismatch')
        return rows[0]


class DurableCollector:
    """One pinned admission for the exact event set, resumed through owner RPCs.

    The local intent is committed BEFORE admission. Lost responses, process restarts
    and bounded wait deadlines reuse the same key and fixed admission deadline.
    Failed/interrupted original executions are never silently re-executed.
    """
    def __init__(self,outbox:Outbox,client:OwnerClient):
        self.outbox=outbox;self.client=client
        with outbox.transaction():
            outbox.db.execute('''CREATE TABLE IF NOT EXISTS collector_intents(
              selection TEXT PRIMARY KEY,scope_digest TEXT NOT NULL,request_key TEXT NOT NULL,
              deadline TEXT NOT NULL,program_id TEXT,created REAL NOT NULL)''')

    def collect_for(self,event_ids,now,*,collection_generation=None):
        def call(*args,**kwargs):
            self._fence(collection_generation)
            return self.client.request(*args,**kwargs)
        self._fence(collection_generation)
        ids=tuple(sorted(event_ids))
        if not 1<=len(ids)<=64 or len(set(ids))!=len(ids):raise Hold('invalid_collector_event_scope')
        for eid in ids:opaque(eid,'event_id')
        config=self.client.config;config.authorize(self.client.clock())
        selection=digest({'source':self.outbox.policy.source_instance,'epoch':self.outbox.policy.source_epoch,'event_ids':ids})
        scope=config.fingerprint()
        with self.outbox.transaction():
            self._fence(collection_generation)
            intent=self.outbox.db.execute('SELECT * FROM collector_intents WHERE selection=?',(selection,)).fetchone()
            if intent and intent['scope_digest']!=scope: raise Hold('collector_scope_drift')
            if not intent:
                key='local-work-'+digest({'selection':selection,'scope':scope})
                deadline=datetime.fromtimestamp(now+600,timezone.utc).isoformat().replace('+00:00','Z')
                self.outbox.db.execute('INSERT INTO collector_intents VALUES (?,?,?,?,NULL,?)',(selection,scope,key,deadline,now))
                intent=self.outbox.db.execute('SELECT * FROM collector_intents WHERE selection=?',(selection,)).fetchone()
        observation=call('runtime','POST',LIB+'GetDeclaredExecution',{'name':NAME,'idempotencyKey':intent['request_key']})
        if not isinstance(observation,dict) or observation.get('admissionContractVersion')!=2:
            raise Hold('collector_admission_contract_missing')
        # Protobuf nonoptional bool omission is the typed false value only in a
        # successful supported response. Explicit null/malformed values are not.
        found=observation.get('found',False)
        if type(found) is not bool:raise Hold('collector_admission_response_invalid')
        if found:
            program=observation.get('program')
        else:
            if intent['program_id'] is not None:raise Hold('collector_retained_identity_lost')
            from .projection import timestamp
            if self.client.clock()>timestamp(intent['deadline']):raise Hold('collector_admission_expired')
            result=call('runtime','POST',LIB+'RunDeclaredProgram',{
                'name':NAME,'inputs':json.loads(config.inputs_json),'expectedDigest':config.expected_digest,
                'provenance':'PROVENANCE_AGENT','async':True,'idempotencyKey':intent['request_key'],
                'admissionDeadline':intent['deadline']})
            if not isinstance(result,dict):raise Hold('collector_admission_response_invalid')
            program=result.get('program')
        self._validate_program(program,config)
        with self.outbox.transaction():
            self._fence(collection_generation)
            current=self.outbox.db.execute('SELECT program_id FROM collector_intents WHERE selection=?',(selection,)).fetchone()
            if current['program_id'] is not None and current['program_id']!=program['id']:raise Hold('collector_program_identity_conflict')
            self.outbox.db.execute('UPDATE collector_intents SET program_id=? WHERE selection=?',(program['id'],selection))
        if program['status'] not in TERMINAL:
            waited=call('runtime','POST',PROGRAMS+'WaitForProgram',{'id':program['id'],'timeoutMillis':90000},timeout=100)
            if not isinstance(waited,dict) or type(waited.get('terminal',False)) is not bool:raise Hold('collector_wait_contract_invalid')
            next_program=waited.get('program');self._validate_program(next_program,config)
            if next_program['id']!=program['id']:raise Hold('collector_wait_identity_conflict')
            program=next_program
            if not waited.get('terminal',False):raise Hold('collector_still_running')
        if program['status']!='PROGRAM_STATUS_SUCCEEDED':raise Hold('collector_terminal_failure_requires_review')
        stdout=program.get('stdout')
        if not isinstance(stdout,str) or len(stdout.encode())>65536:raise Hold('collector_output_limit')
        return {'program':{'id':program['id'],'status':program['status'],'program_digest':program['programDigest'],'stdout':stdout}}

    def _fence(self,generation):
        if generation is None:return
        row=self.outbox.db.execute("SELECT value FROM meta WHERE key='collection_lease'").fetchone()
        lease=json.loads(row['value']) if row else None
        if not lease or lease['generation']!=generation or lease['until']<=self.client.clock():
            raise Hold('collection_lease_lost')

    @staticmethod
    def _validate_program(program,config):
        if not isinstance(program,dict):raise Hold('collector_program_missing')
        opaque(program.get('id'),'program_id')
        if program.get('programName')!=NAME or program.get('programDigest')!=config.expected_digest:
            raise Hold('collector_program_identity_mismatch')
        if program.get('status') not in TERMINAL|{'PROGRAM_STATUS_ACCEPTED','PROGRAM_STATUS_RUNNING'}:
            raise Hold('collector_program_status_invalid')
