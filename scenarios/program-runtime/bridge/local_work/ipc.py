"""Strict one-shot private IPC. Never exposed as an HTTP endpoint.

Initial stdin contains bounded config/request only. Credentials are requested
lazily through separate inherited pipes; stdout contains only a fixed status.
No environment credential lookup, path discovery or provisioning is performed.
"""
from __future__ import annotations
import base64
import dataclasses
import json
import os
import pathlib
import time
from .adapter import AdapterPolicy,RuntimeGrant,Outbox,HTTPSender,Hold,strict_json,canonical
from .owners import OwnerConfig,LoopbackTransport
from .managed import ManagedConfig,ManagedRunner

SCHEMA='flywheel.local-work-native-ipc.v1'
MAX_INPUT=192*1024
MAX_SECRET_RESPONSE=16*1024
STATUS_CODES={204,400,401,403,404,409,413,429,503}
PURPOSES={'native_webhook_verify','events_reader','runtime_collector','flywheel_submit'}


def require_object(value,keys):
    if not isinstance(value,dict) or set(value)!=set(keys):raise Hold('ipc_shape')
    return value


def construct(cls,value):
    if not isinstance(value,dict) or set(value)-{f.name for f in dataclasses.fields(cls)}:raise Hold('ipc_config_shape')
    try:return cls(**value)
    except (ValueError,TypeError,KeyError,AttributeError) as exc:raise Hold('ipc_config_invalid') from exc


def decode_base64(value,maximum):
    if not isinstance(value,str) or len(value)>((maximum+2)//3)*4:raise Hold('ipc_base64_size')
    try:result=base64.b64decode(value,validate=True)
    except (ValueError,TypeError) as exc:raise Hold('ipc_base64_invalid') from exc
    if len(result)>maximum:raise Hold('ipc_base64_size')
    return result


class PipeCredentials:
    """Only finite enumerated secret requests. Values are cached per one-shot call."""
    def __init__(self,request_writer,response_reader,operation,export_enabled):
        self.writer=request_writer;self.reader=response_reader;self.operation=operation
        self.export_enabled=export_enabled;self.cache={}

    def __call__(self,purpose):
        allowed={'native_webhook_verify'} if self.operation=='accept_native' else {'events_reader','runtime_collector'}
        if self.operation=='tick' and self.export_enabled:allowed.add('flywheel_submit')
        if purpose not in allowed or purpose not in PURPOSES:raise Hold('credential_purpose_forbidden')
        if purpose in self.cache:return self.cache[purpose]
        if len(self.cache)>=4:raise Hold('credential_request_limit')
        self.writer.write(canonical({'purpose':purpose}).encode()+b'\n');self.writer.flush()
        line=self.reader.readline(MAX_SECRET_RESPONSE+1)
        if not line or len(line)>MAX_SECRET_RESPONSE or not line.endswith(b'\n'):raise Hold('credential_response_invalid')
        try:response=strict_json(line,MAX_SECRET_RESPONSE)
        except Hold as exc:raise Hold('credential_response_invalid') from exc
        if isinstance(response,dict) and set(response)=={'error'} and response['error']=='unavailable':raise Hold('credential_unavailable')
        require_object(response,{'value_base64'})
        value=decode_base64(response['value_base64'],8192)
        if not value:raise Hold('credential_empty')
        if purpose in {'events_reader','runtime_collector'}:
            try:value=value.decode('utf-8')
            except UnicodeError as exc:raise Hold('credential_encoding') from exc
            if not value.startswith('Bearer ') or any(c in value for c in '\r\n'):raise Hold('credential_format')
        elif purpose=='flywheel_submit' and len(value)!=32:raise Hold('credential_key_size')
        self.cache[purpose]=value
        return value


def parse_packet(raw):
    packet=require_object(strict_json(raw,MAX_INPUT),{'schema','operation','request','config'})
    if packet['schema']!=SCHEMA or packet['operation'] not in {'accept_native','tick'}:raise Hold('ipc_operation')
    request=require_object(packet['request'],{'body_base64','signature','event_id','idempotency_key'})
    for key in ('signature','event_id','idempotency_key'):
        if not isinstance(request[key],str) or len(request[key])>512:raise Hold('ipc_request_header')
    body=decode_base64(request['body_base64'],65536)
    if packet['operation']=='tick' and (body or any(request[k] for k in ('signature','event_id','idempotency_key'))):raise Hold('ipc_tick_payload')
    config=packet['config']
    if not isinstance(config,dict) or type(config.get('enabled')) is not bool:raise Hold('ipc_config_shape')
    if config['enabled'] is False:
        return packet,None,None,body
    require_object(config,{'enabled','storage_root','policy','owners','export'})
    root=config['storage_root']
    if not isinstance(root,str) or len(root)>4096 or any(c in root for c in '\0\r\n') or not pathlib.Path(root).is_absolute() or os.path.normpath(root)!=root:
        raise Hold('ipc_storage_root')
    policy=config['policy']
    if not isinstance(policy,dict):raise Hold('ipc_policy_shape')
    policy=dict(policy)
    pairs=policy.get('allowed_pairs',[])
    if not isinstance(pairs,list) or any(not isinstance(p,list) or len(p)!=2 for p in pairs):raise Hold('ipc_allowlist_shape')
    policy['allowed_pairs']=tuple(tuple(p) for p in pairs)
    managed=ManagedConfig(policy=construct(AdapterPolicy,policy),owners=construct(OwnerConfig,config['owners']),
                          export=construct(RuntimeGrant,config['export']),enabled=True)
    path=os.path.join(root,'local-work-outbox.sqlite3')
    return packet,managed,path,body


def serve_packet(raw,request_writer,response_reader,*,clock=time.time,owner_transport=None,export_transport=None,outbox_factory=None):
    """Execute one private packet. Fakes may be injected only by Python tests."""
    try:
        packet,config,path,body=parse_packet(raw)
        if config is None:return {'http_status':404}
        provider=PipeCredentials(request_writer,response_reader,packet['operation'],config.export.enabled is True)
        factory=outbox_factory or (lambda policy:Outbox(path,policy))
        runner=ManagedRunner(config,clock,factory,provider,owner_transport or LoopbackTransport(),export_transport or HTTPSender())
        request=packet['request']
        result=runner.dispatch(packet['operation'],body,request['signature'],request['event_id'],request['idempotency_key'])
        code=result.get('http_status')
        if type(code) is not int or code not in STATUS_CODES:return {'http_status':503}
        return {'http_status':code}
    except Hold:return {'http_status':400}
    except Exception:return {'http_status':503}
