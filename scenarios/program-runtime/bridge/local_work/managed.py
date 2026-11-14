"""Disabled managed-host runner ABI. No executable entry point or listener.

The Go host owns lifecycle/deadline/serialization. A reviewed private IPC adapter
must inject this runner's config and providers; none is configured by this task.
"""
from dataclasses import dataclass
from .adapter import Hold,Outbox,AdapterPolicy,RuntimeGrant,authenticate_native
from .owners import OwnerClient,DurableCollector,OwnerConfig


@dataclass(frozen=True)
class ManagedConfig:
    policy:AdapterPolicy
    owners:OwnerConfig
    export:RuntimeGrant=RuntimeGrant()
    enabled:bool=False


class ManagedRunner:
    def __init__(self,config,clock,outbox_factory,credential_provider,owner_transport,export_transport):
        self.config=config;self.clock=clock;self.outbox_factory=outbox_factory
        self.credential_provider=credential_provider;self.owner_transport=owner_transport
        self.export_transport=export_transport

    def dispatch(self,operation,body=b'',signature='',event_id='',idempotency_key=''):
        # Disabled means zero storage/key/transport calls, not a partially active loop.
        if self.config.enabled is not True:return {'http_status':404,'state':'disabled'}
        if operation not in {'accept_native','tick'}:return {'http_status':400,'state':'invalid_operation'}
        try:
            self.config.owners.authorize(self.clock())
            if not self.config.policy.allowed_pairs or self.config.policy.source_epoch=='uncommissioned':
                raise Hold('uncommissioned_source')
        except Hold:return {'http_status':503,'state':'local_access_hold'}
        box=None
        try:
            if operation=='accept_native':
                if not isinstance(body,bytes) or len(body)>65536:return {'http_status':413,'state':'body_limit'}
                local_key=self.credential_provider('native_webhook_verify')
                headers={'X-Vrooli-Events-Signature':signature,
                    'X-Vrooli-Events-Event-ID':event_id,'X-Vrooli-Events-Idempotency-Key':idempotency_key}
                native=authenticate_native(body,headers,local_key)
                if (native['source_scenario'],native['event_type']) not in self.config.policy.allowed_pairs:
                    return {'http_status':204,'state':'outside_exact_allowlist'}
                box=self.outbox_factory(self.config.policy)
                box.accept_native(body,headers,local_key,self.clock())
                return {'http_status':204,'state':'durable_or_duplicate'}
            if body or signature or event_id or idempotency_key:return {'http_status':400,'state':'invalid_tick'}
            box=self.outbox_factory(self.config.policy)
            client=OwnerClient(self.config.owners,self.owner_transport,
                lambda owner:self.credential_provider('events_reader' if owner=='events' else 'runtime_collector'),self.clock)
            collector=DurableCollector(box,client)
            with client.bounded_cycle(150):
                collection=box.collect_once(client.resolve,collector,self.clock(),clock=self.clock)
            if collection['state']=='collection_held':
                return {'http_status':503,'state':'collection_held'}
            if self.config.export.enabled is not True:
                return {'http_status':204,'state':'export_disabled','collection':collection['state']}
            self.config.export.validate(self.config.policy,self.clock())
            # Do not retrieve a cloud key merely because an idle host ticks.
            pending=box.db.execute("SELECT 1 FROM batches WHERE state IN ('prepared','retry','sending') LIMIT 1").fetchone()
            if pending:
                key=self.credential_provider('flywheel_submit')
                delivery=box.export_once(self.config.export,key,self.clock(),self.export_transport)
                return {'http_status':503 if delivery['state']=='held' else 204,'state':delivery['state'],'collection':collection['state']}
            return {'http_status':204,'state':'idle','collection':collection['state']}
        except Hold as error:
            # Fixed categories only. Never echo raw owner/credential/transport errors.
            kind=str(error)
            if kind in {'native_signature_required','native_signature_invalid'}:
                return {'http_status':401,'state':'native_auth_refused'}
            if kind in {'native_webhook_contract','native_identity_header_mismatch','invalid_json'}:
                return {'http_status':400,'state':'native_shape_refused'}
            return {'http_status':503,'state':'held'}
        except Exception:return {'http_status':503,'state':'unavailable'}
        finally:
            if box is not None:box.close()
