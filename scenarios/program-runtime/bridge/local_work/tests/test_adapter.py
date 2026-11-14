"""Hermetic component qualification. All signing keys are fixed PUBLIC fixtures."""
import copy
import hashlib
import hmac
import json
import pathlib
import socket
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[2]))
from local_work.adapter import (AdapterPolicy, Outbox, RuntimeGrant, VERIFIED_PAIRS, Hold,
    canonical, signed_headers, strict_json, HTTPSender, ORIGIN, PATH)
from local_work.projection import digest

NOW=1794013200
AT='2026-11-07T01:00:00Z'
LOCAL_KEY=b'PUBLIC-NATIVE-TEST-FIXTURE-ONLY'
CLOUD_KEY=b'0123456789abcdef0123456789abcdef'  # PUBLIC fixture, never live
assert len(CLOUD_KEY)==32


def policy(**kw):
    return AdapterPolicy(source_epoch='fixture-epoch',allowed_pairs=VERIFIED_PAIRS,**kw)


def grant(**kw):
    args=dict(enabled=True,approval_ref='PUBLIC-SYNTHETIC-NO-LIVE-AUTHORITY',source_epoch='fixture-epoch',
              key_id='public-fixture',not_before=NOW-10,expires_at=NOW+600)
    args.update(kw)
    return RuntimeGrant(**args)


def native(eid='event-1',**kw):
    result={'event_id':eid,'event_type':VERIFIED_PAIRS[0][1],'source_scenario':'agent-manager',
            'payload':{'text':'PRIVATE_TEXT','credential':'PRIVATE_SECRET'},'delivered_at':AT}
    result.update(kw)
    raw=canonical(result).encode()
    headers={'X-Vrooli-Events-Signature':'sha256='+hmac.new(LOCAL_KEY,raw,hashlib.sha256).hexdigest(),
             'X-Vrooli-Events-Event-ID':eid,'X-Vrooli-Events-Idempotency-Key':eid}
    return raw,headers


def envelope(eid='event-1',**kw):
    result={'eventId':eid,'eventType':VERIFIED_PAIRS[0][1],'source':{'scenario':'agent-manager'},
            'occurredAt':AT,'data':{'text':'PRIVATE_TEXT','secret':'PRIVATE_SECRET'}}
    result.update(kw)
    return result


def receipt(**kw):
    source={'status':'ok','queried_scope':{'team_id':'PRIVATE_TEAM'},'collected_at':AT,
            'producer_observed_at':None,'query_complete':False,'output_complete':False,
            'errors':[],'entities':[{'title':'PRIVATE_TEXT','secret':'PRIVATE_SECRET'}]}
    source.update(kw)
    value={'program':'program-runtime.local-work-snapshot','version':'2','phase':'report','status':'partial',
           'signals':{'collected_at':AT,'finished_at':AT,'coverage':{'kind':'bounded_observation'},
                      'output_truncated':False,'sources':{'team_execution/PRIVATE_TEAM':source}},'errors':[]}
    return {'program':{'id':'prog_fixture','program_digest':'a'*64,'status':'PROGRAM_STATUS_SUCCEEDED','stdout':canonical(value)}}


def response(body,**kw):
    packet=json.loads(body)
    value={'schema':'flywheel.local-work-receipt.v1','batch_id':packet['batch_id'],'receipt_id':'receipt-fixture',
           'state':'gap_hold','new_events':len(packet['events']),'duplicate':False}
    value.update(kw)
    return 200,{'Content-Type':'application/json'},canonical(value).encode()


class AdapterTests(unittest.TestCase):
    def setUp(self):
        self.no_network=patch.object(socket,'create_connection',side_effect=AssertionError('network forbidden'))
        self.no_network.start()
        self.temp=tempfile.TemporaryDirectory(); self.path=str(pathlib.Path(self.temp.name)/'outbox.db')
        self.box=Outbox(self.path,policy())
    def tearDown(self):
        self.box.close(); self.temp.cleanup(); self.no_network.stop()
    def accept(self,eid='event-1',**kw):
        raw,headers=native(eid,**kw)
        return self.box.accept_native(raw,headers,LOCAL_KEY,NOW)
    def prepare(self,eid='event-1',r=None):
        self.accept(eid); self.box.qualify_event(eid,envelope(eid),NOW)
        return self.box.prepare([eid],r or receipt(),NOW+60)
    def test_empty_allowlist_ignores_all(self):
        box=Outbox(':memory:')
        self.assertEqual(box.accept_native(*native(),LOCAL_KEY,NOW)['state'],'ignored'); box.close()
    def test_allowlist_cannot_admit_prototype_synthetic_or_feedback(self):
        for pair in [('agent-manager','agent-manager.run.completed.v1'),('program-runtime','program-runtime.completed.v1')]:
            with self.assertRaises(Hold): AdapterPolicy(allowed_pairs=(pair,))
    def test_import_and_disabled_send_never_calls_transport(self):
        self.prepare(); calls=[]
        with self.assertRaisesRegex(Hold,'disabled'): self.box.export_once(RuntimeGrant(),CLOUD_KEY,NOW+60,lambda *a:calls.append(a))
        self.assertFalse(calls)
    def test_signed_native_receipt_commits_before_collector(self):
        self.assertEqual(self.accept()['state'],'durable_pending_resolution')
        self.box.close(); self.box=Outbox(self.path,policy())
        self.assertEqual(self.box.pending(),[{'event_id':'event-1','state':'awaiting_envelope'}])
    def test_native_key_is_never_optional(self):
        raw,headers=native()
        for key in [b'',None]:
            with self.assertRaises(Hold): self.box.accept_native(raw,headers,key,NOW)
        headers['X-Vrooli-Events-Signature']='sha256='+'0'*64
        with self.assertRaisesRegex(Hold,'signature_invalid'): self.box.accept_native(raw,headers,LOCAL_KEY,NOW)
    def test_native_signature_binds_raw_bytes(self):
        raw,headers=native()
        with self.assertRaisesRegex(Hold,'signature_invalid'): self.box.accept_native(raw+b' ',headers,LOCAL_KEY,NOW)
    def test_wrong_and_ambiguous_identity_headers_fail(self):
        raw,headers=native(); headers['X-Vrooli-Events-Event-ID']='different'
        with self.assertRaises(Hold): self.box.accept_native(raw,headers,LOCAL_KEY,NOW)
        raw,headers=native(); headers['x-vrooli-events-event-id']='event-1'
        with self.assertRaisesRegex(Hold,'ambiguous'): self.box.accept_native(raw,headers,LOCAL_KEY,NOW)
    def test_duplicate_delivery_time_does_not_conflict(self):
        self.accept()
        self.assertEqual(self.accept(delivered_at='2026-11-07T01:01:00Z')['state'],'duplicate')
    def test_changed_id_payload_sets_durable_global_hold(self):
        self.accept()
        with self.assertRaisesRegex(Hold,'identity_conflict'): self.accept(payload={'changed':True})
        self.box.close(); self.box=Outbox(self.path,policy())
        self.assertIsNotNone(self.box.db.execute("SELECT 1 FROM holds WHERE kind='event_identity_conflict'").fetchone())
    def test_raw_payload_and_local_key_never_persist(self):
        self.prepare()
        contents=pathlib.Path(self.path).read_bytes()
        for private in [b'PRIVATE_TEXT',b'PRIVATE_SECRET',b'PRIVATE_TEAM',LOCAL_KEY,CLOUD_KEY]: self.assertNotIn(private,contents)
    def test_native_capacity_is_backpressure_without_loss(self):
        box=Outbox(':memory:',policy(max_retained=1))
        box.accept_native(*native('one'),LOCAL_KEY,NOW)
        with self.assertRaisesRegex(Hold,'capacity'): box.accept_native(*native('two'),LOCAL_KEY,NOW)
        self.assertEqual(len(box.pending()),1); box.close()
    def test_local_database_rejects_shared_directory_or_symlink(self):
        with tempfile.TemporaryDirectory() as root:
            shared=pathlib.Path(root)/'shared'; shared.mkdir(mode=0o755)
            with self.assertRaisesRegex(Hold,'private_owner'): Outbox(str(shared/'x.db'),policy())
            link=pathlib.Path(root)/'link'; link.symlink_to(self.path)
            with self.assertRaises(OSError): Outbox(str(link),policy())
    def test_existing_unrelated_database_is_not_modified(self):
        import os, sqlite3
        path=str(pathlib.Path(self.temp.name)/'foreign.db')
        fd=os.open(path,os.O_CREAT|os.O_RDWR,0o600); os.close(fd)
        db=sqlite3.connect(path); db.execute('CREATE TABLE important_work(id TEXT)'); db.commit(); db.close()
        before=pathlib.Path(path).read_bytes()
        with self.assertRaisesRegex(Hold,'not_owned'): Outbox(path,policy())
        self.assertEqual(before,pathlib.Path(path).read_bytes())
    def test_identity_epoch_or_policy_changes_require_migration(self):
        with self.assertRaisesRegex(Hold,'migration'): Outbox(self.path,AdapterPolicy(source_epoch='new',allowed_pairs=VERIFIED_PAIRS))
    def test_delivered_at_is_never_occurrence_fallback(self):
        self.accept(); e=envelope(); del e['occurredAt']
        with self.assertRaisesRegex(Hold,'timestamp'): self.box.qualify_event('event-1',e,NOW)
        self.assertEqual(self.box.pending()[0]['state'],'awaiting_envelope')
    def test_resolver_rejects_other_id_source_or_type(self):
        self.accept()
        for e in [envelope('other'),envelope(source={'scenario':'swarm-manager'}),envelope(eventType=VERIFIED_PAIRS[1][1])]:
            with self.assertRaisesRegex(Hold,'identity_mismatch'): self.box.qualify_event('event-1',e,NOW)
    def test_missing_canonical_record_is_durable_gap(self):
        self.accept()
        with self.assertRaisesRegex(Hold,'unavailable'): self.box.qualify_event('event-1',None,NOW)
        self.box.close(); self.box=Outbox(self.path,policy())
        self.assertIsNotNone(self.box.db.execute("SELECT 1 FROM holds WHERE kind='retention_gap'").fetchone())
    def test_future_event_rejected_old_event_preserved_with_gap(self):
        self.accept()
        with self.assertRaisesRegex(Hold,'future_clock'): self.box.qualify_event('event-1',envelope(occurredAt='2026-11-08T01:00:00Z'),NOW)
        self.box.qualify_event('event-1',envelope(occurredAt='2026-11-06T01:00:00Z'),NOW)
        body=self.box.prepare(['event-1'],receipt(),NOW)
        self.assertEqual(body['coverage_gap'],'retention_gap')
        self.assertEqual(body['events'][0]['occurred_at'],'2026-11-06T01:00:00Z')
    def test_program_target_is_never_a_feedback_trigger(self):
        self.accept(); result=self.box.qualify_event('event-1',envelope(target={'scenario':'program-runtime'}),NOW)
        self.assertEqual(result['state'],'ignored_feedback'); self.assertFalse(self.box.due_ids(NOW+1000))
    def test_canonical_id_cannot_change_after_resolution(self):
        self.prepare()
        with self.assertRaisesRegex(Hold,'identity_conflict'): self.box.qualify_event('event-1',envelope(data={'changed':True}),NOW)
        with self.assertRaisesRegex(Hold,'identity_conflict'): self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda *a:None)
    def test_coalesce_window_and_batch_count(self):
        self.accept(); self.box.qualify_event('event-1',envelope(),NOW)
        self.assertEqual(self.box.due_ids(NOW+59),[]); self.assertEqual(self.box.due_ids(NOW+60),['event-1'])
    def test_collector_failure_preserves_pending_event(self):
        self.accept(); self.box.qualify_event('event-1',envelope(),NOW)
        r=receipt(); r['program']['status']='PROGRAM_STATUS_FAILED'
        with self.assertRaises(Hold): self.box.prepare(['event-1'],r,NOW)
        self.assertEqual(self.box.pending()[0]['state'],'resolved')
    def test_no_event_means_collection_cannot_manufacture_progress(self):
        self.assertFalse(self.box.due_ids(NOW))
        with self.assertRaises(Hold): self.box.prepare([],receipt(),NOW)
    def test_distinct_event_not_suppressed_by_identical_bounded_sample(self):
        first=self.prepare('one'); second=self.prepare('two')
        self.assertNotEqual(first['batch_id'],second['batch_id'])
    def test_producer_freshness_anchored_to_snapshot_finish_not_staging(self):
        cases=[('2026-11-06T01:00:00Z','within_age_bound'),
               ('2026-11-06T00:59:59Z','stale'),
               ('2026-11-07T01:00:30Z','within_age_bound'),
               ('2026-11-07T01:00:31Z','future_clock')]
        for index,(producer,expected) in enumerate(cases):
            body=self.prepare(str(index),receipt(producer_observed_at=producer))
            self.assertEqual(body['snapshot']['sources'][0]['producer_freshness'],expected)
    def test_partial_unknown_stale_and_no_authority_preserved(self):
        b=self.prepare(r=receipt(status='unavailable',query_complete=None,producer_observed_at='2026-10-07T01:00:00Z'))
        self.assertEqual(b['snapshot']['sources'][0]['producer_freshness'],'stale')
        self.assertIsNone(b['snapshot']['sources'][0]['query_complete'])
        self.assertEqual(b['snapshot']['acceptance'],'unverified'); self.assertEqual(b['effect_authority'],'none')
    def test_permission_hold_survives_later_clean_collection(self):
        self.prepare('one',receipt(errors=[{'class':'no_grant','detail':'PRIVATE_SECRET'}]))
        b=self.prepare('two'); self.assertTrue(b['snapshot']['permission_hold'])
    def test_unknown_checkpoint_does_not_claim_history_complete(self):
        self.assertEqual(self.prepare()['coverage_gap'],'unknown_checkpoint')
    def test_source_reset_remains_hold_after_clean_snapshot(self):
        self.box.mark_gap('source_reset',NOW)
        self.assertEqual(self.prepare()['coverage_gap'],'source_reset')
    def test_stale_snapshot_fails_before_freezing(self):
        self.accept(); self.box.qualify_event('event-1',envelope(),NOW)
        with self.assertRaisesRegex(Hold,'stale'): self.box.prepare(['event-1'],receipt(),NOW+400)
    def test_prepared_packet_is_immutable_after_restart(self):
        b=self.prepare(); raw=self.box.preview(b['batch_id'])
        self.box.close(); self.box=Outbox(self.path,policy()); self.assertEqual(raw,self.box.preview(b['batch_id']))
        with self.assertRaisesRegex(Hold,'resolved'): self.box.prepare(['event-1'],receipt(),NOW+60)
    def test_matching_cloud_receipt_is_transport_only(self):
        self.prepare(); result=self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda u,h,b:response(b))
        self.assertEqual(result['state'],'transport_accepted'); self.assertEqual(result['handler_pickup'],'unverified')
        self.assertEqual(self.box.export_once(grant(),CLOUD_KEY,NOW+61,lambda *a:self.fail())['state'],'not_due')
    def test_independent_cloud_handler_receipt_fixture_is_accepted(self):
        body=self.prepare()
        expected='aa9fc1dbef85a018b76569efbacef84284d5d72a593359af3f56c261057d3bb5'
        self.assertEqual(body['batch_id'],expected)
        # Returned by the actual TypeScript handler using isolated D1/R2 and this
        # native PUBLIC golden request, under owner native_fixture_owner.
        result=self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda u,h,b:response(b,
            receipt_id='bridge_45eab1e5cdf55088600dc5448ee35359a43ab8bcab59e333573d5abb80a10640'))
        self.assertEqual(result['state'],'transport_accepted')
        self.assertEqual(result['receipt']['batch_id'],expected)
    def test_runtime_grant_checks_expiry_source_scope_and_destination(self):
        self.prepare()
        for g in [grant(expires_at=NOW+1),grant(source_instance='other'),grant(origin='https://other.example'),grant(source_epoch='wrong'),grant(expires_at=NOW+700000),grant(approval_ref='')]:
            with self.assertRaises(Hold): self.box.export_once(g,CLOUD_KEY,NOW+60,lambda *a:self.fail())
    def test_signing_binds_origin_path_time_and_raw_bytes(self):
        self.prepare(); body=self.box.db.execute('SELECT body FROM batches').fetchone()[0]
        hdr=signed_headers(body,grant(),CLOUD_KEY,NOW+60)
        expected=hmac.new(CLOUD_KEY,f'flywheel.local-work-http.v1\n{NOW+60}\nPOST\n{ORIGIN}{PATH}\n'.encode()+body,hashlib.sha256).digest()
        import base64
        self.assertEqual(hdr['X-Flywheel-Bridge-Signature'],'v1,'+base64.b64encode(expected).decode())
        self.assertNotEqual(hdr,signed_headers(body,grant(),CLOUD_KEY,NOW+61))
    def test_uncertain_retry_enforces_persisted_due_and_identical_body(self):
        b=self.prepare(); calls=[]
        def timeout(u,h,body): calls.append((h,body)); raise TimeoutError()
        first=self.box.export_once(grant(),CLOUD_KEY,NOW+60,timeout)
        self.assertEqual(first['state'],'retry')
        self.box.close(); self.box=Outbox(self.path,policy())
        self.assertEqual(self.box.export_once(grant(),CLOUD_KEY,first['next_attempt_at']-1,timeout)['state'],'not_due')
        self.box.export_once(grant(),CLOUD_KEY,first['next_attempt_at'],timeout)
        self.assertEqual(calls[0][1],calls[1][1]); self.assertNotEqual(calls[0][0],calls[1][0])
    def test_crash_after_dispatch_reclaims_only_expired_lease(self):
        self.prepare()
        def crash(*args): raise KeyboardInterrupt()
        with self.assertRaises(KeyboardInterrupt): self.box.export_once(grant(),CLOUD_KEY,NOW+60,crash)
        self.box.close(); self.box=Outbox(self.path,policy())
        self.assertEqual(self.box.export_once(grant(),CLOUD_KEY,NOW+89,lambda *a:self.fail())['state'],'not_due')
        self.assertEqual(self.box.export_once(grant(),CLOUD_KEY,NOW+90,lambda u,h,b:response(b,duplicate=True,new_events=0))['state'],'transport_accepted')
    def test_five_failed_attempts_hold(self):
        self.prepare(); at=NOW+60
        for i in range(5):
            result=self.box.export_once(grant(),CLOUD_KEY,at,lambda *a:(503,{},b''))
            self.assertEqual(result['attempts'],i+1)
            at=result['next_attempt_at'] or at+60
        self.assertEqual(result['state'],'held')
        self.assertEqual(self.box.export_once(grant(),CLOUD_KEY,at,lambda *a:self.fail())['state'],'not_due')
    def test_redirect_permission_stale_and_capacity_hold(self):
        for status in [301,302,307,308,401,403,404,409,413]:
            with self.subTest(status=status):
                self.prepare(str(status))
                result=self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda *a:(status,{},b''))
                self.assertEqual(result['state'],'held')
    def test_invalid_cloud_ack_is_uncertain_not_success(self):
        self.prepare()
        result=self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda u,h,b:response(b,batch_id='wrong'))
        self.assertEqual(result['category'],'uncertain_receipt')
    def test_bounded_driver_resolves_exact_id_and_collects_only_after_signal(self):
        calls=[]
        def resolver(eid): calls.append(('resolve',eid)); return envelope(eid)
        def collector(): calls.append(('collect',)); return receipt()
        self.assertEqual(self.box.collect_once(resolver,collector,NOW)['state'],'no_collection_due')
        self.assertFalse(calls)
        self.accept()
        self.assertEqual(self.box.collect_once(resolver,collector,NOW)['state'],'no_collection_due')
        self.assertEqual(calls,[('resolve','event-1')])
        self.assertEqual(self.box.collect_once(resolver,collector,NOW+60)['state'],'prepared')
        self.assertEqual(calls,[('resolve','event-1'),('collect',)])
    def test_concurrent_collectors_do_not_manufacture_retry_failures(self):
        other=Outbox(self.path,policy())
        try:
            for index in range(6):
                eid=str(index); self.accept(eid); self.box.qualify_event(eid,envelope(eid),NOW)
                def collect():
                    self.assertEqual(other.collect_once(envelope,receipt,NOW+60)['state'],'collection_busy')
                    return receipt()
                self.assertEqual(self.box.collect_once(envelope,collect,NOW+60)['state'],'prepared')
            self.assertIsNone(self.box.db.execute("SELECT value FROM meta WHERE key='collection_retry'").fetchone())
        finally: other.close()
    def test_expired_collector_cannot_overwrite_new_owner(self):
        other=Outbox(self.path,policy())
        self.accept(); self.box.qualify_event('event-1',envelope(),NOW)
        def delayed():
            # The new owner runs after A's lease expires, using a fresh receipt.
            fresh=receipt(); body=json.loads(fresh['program']['stdout'])
            at='2026-11-07T01:06:00Z'
            body['signals']['collected_at']=at; body['signals']['finished_at']=at
            for src in body['signals']['sources'].values(): src['collected_at']=at
            fresh['program']['stdout']=canonical(body)
            self.assertEqual(other.collect_once(envelope,lambda:fresh,NOW+360)['state'],'prepared')
            return receipt()
        try:
            self.assertEqual(self.box.collect_once(envelope,delayed,NOW+60)['state'],'collection_superseded')
            self.assertIsNone(self.box.db.execute("SELECT value FROM meta WHERE key='collection_retry'").fetchone())
        finally: other.close()
    def test_superseded_resolver_cannot_add_gap_or_identity_conflict(self):
        self.box.close()
        self.path=str(pathlib.Path(self.temp.name)/'fenced.db')
        self.box=Outbox(self.path,policy(event_max_age=1000))
        other=Outbox(self.path,policy(event_max_age=1000))
        fresh=receipt(); body=json.loads(fresh['program']['stdout']); at='2026-11-07T01:06:00Z'
        body['signals']['collected_at']=at; body['signals']['finished_at']=at
        for src in body['signals']['sources'].values(): src['collected_at']=at
        fresh['program']['stdout']=canonical(body)
        try:
            for index,answer in enumerate([None,envelope('1',data={'changed':True})]):
                eid=str(index); self.accept(eid)
                def stale_resolver(selected):
                    self.assertEqual(other.collect_once(envelope,lambda:fresh,NOW+360)['state'],'prepared')
                    return answer
                self.assertEqual(self.box.collect_once(stale_resolver,receipt,NOW+60)['state'],'collection_superseded')
            holds={r[0] for r in self.box.db.execute('SELECT kind FROM holds')}
            self.assertEqual(holds,{'unknown_checkpoint'})
        finally: other.close()
    def test_crashed_collection_lease_is_recovered_after_expiry(self):
        self.accept(); self.box.qualify_event('event-1',envelope(),NOW)
        def crash(): raise KeyboardInterrupt()
        self.box.db.execute("UPDATE native_events SET received=?",(NOW-60,))
        with self.assertRaises(KeyboardInterrupt): self.box.collect_once(envelope,crash,NOW)
        self.box.close(); self.box=Outbox(self.path,policy())
        self.assertEqual(self.box.collect_once(envelope,receipt,NOW+299)['state'],'collection_busy')
        self.assertEqual(self.box.collect_once(envelope,receipt,NOW+300)['state'],'prepared')
    def test_new_signals_do_not_change_uncertain_collector_selection(self):
        self.accept('one');self.box.qualify_event('one',envelope('one'),NOW)
        calls=[]
        class Collector:
            def collect_for(inner,ids,now,**kw):
                calls.append(list(ids))
                if len(calls)==1:raise TimeoutError()
                return receipt()
        collector=Collector()
        first=self.box.collect_once(envelope,collector,NOW+60)
        self.accept('two');self.box.qualify_event('two',envelope('two'),NOW)
        self.box.collect_once(envelope,collector,first['next_attempt_at'])
        self.assertEqual(calls,[['one'],['one']])
        self.assertEqual({r['event_id']:r['state'] for r in self.box.pending()}['two'],'resolved')
    def test_collection_uses_completion_clock_for_fresh_receipt(self):
        self.accept();self.box.qualify_event('event-1',envelope(),NOW)
        current=[NOW+60]
        def collect():
            current[0]=NOW+150
            value=receipt();body=json.loads(value['program']['stdout']);at='2026-11-07T01:02:30Z'
            body['signals']['collected_at']=at;body['signals']['finished_at']=at
            for src in body['signals']['sources'].values():src['collected_at']=at
            value['program']['stdout']=canonical(body);return value
        self.assertEqual(self.box.collect_once(envelope,collect,current[0],clock=lambda:current[0])['state'],'prepared')
    def test_owner_auth_denial_persists_permission_hold(self):
        self.accept()
        def denied(eid):raise Hold('owner_permission_hold')
        self.box.collect_once(denied,receipt,NOW)
        self.assertIsNotNone(self.box.db.execute("SELECT 1 FROM holds WHERE kind='permission_hold'").fetchone())
    def test_collection_retry_is_durable_and_no_private_error_persisted(self):
        self.accept()
        def unavailable(eid): raise OSError('PRIVATE_SECRET')
        result=self.box.collect_once(unavailable,lambda:self.fail(),NOW)
        self.assertEqual(result['state'],'collection_retry')
        self.box.close(); self.box=Outbox(self.path,policy())
        self.assertEqual(self.box.collect_once(lambda e:self.fail(),lambda:self.fail(),NOW+1)['state'],'collection_not_due')
        self.assertNotIn(b'PRIVATE_SECRET',pathlib.Path(self.path).read_bytes())
    def test_collection_five_failures_require_explicit_repair(self):
        self.accept(); at=NOW
        def unavailable(eid): raise OSError()
        for attempt in range(5):
            result=self.box.collect_once(unavailable,lambda:self.fail(),at)
            at=result['next_attempt_at'] or at+10
        self.assertEqual(result['state'],'collection_held')
        self.assertEqual(self.box.collect_once(lambda e:self.fail(),lambda:self.fail(),at)['state'],'collection_held')
    def test_rate_limit_honors_retry_after_across_restart(self):
        self.prepare()
        result=self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda *a:(429,{'Retry-After':'200'},b''))
        self.assertEqual(result['next_attempt_at'],NOW+260)
        self.box.close(); self.box=Outbox(self.path,policy())
        self.assertEqual(self.box.export_once(grant(),CLOUD_KEY,NOW+259,lambda *a:self.fail())['state'],'not_due')
    def test_invalid_or_excessive_retry_after_holds(self):
        for n,value in enumerate(['nonsense','9999999999999999']):
            self.prepare(str(n))
            result=self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda *a:(429,{'Retry-After':value},b''))
            self.assertEqual(result['state'],'held')
    def test_cloud_receipt_cannot_erase_gap(self):
        self.prepare()
        result=self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda u,h,b:response(b,state='needs_source_read'))
        self.assertEqual(result['category'],'uncertain_receipt')
    def test_duplicate_receipt_must_not_claim_new_events(self):
        self.prepare()
        result=self.box.export_once(grant(),CLOUD_KEY,NOW+60,lambda u,h,b:response(b,duplicate=True,new_events=1))
        self.assertEqual(result['category'],'uncertain_receipt')
    def test_malformed_canonical_source_is_explicit_hold(self):
        self.accept()
        with self.assertRaisesRegex(Hold,'canonical_envelope'): self.box.qualify_event('event-1',envelope(source=None),NOW)
    def test_allowlist_cannot_be_mutated_after_policy_fingerprint(self):
        with self.assertRaisesRegex(Hold,'mutable'): AdapterPolicy(allowed_pairs=list(VERIFIED_PAIRS))
    def test_boolean_numeric_policy_is_rejected(self):
        with self.assertRaises(Hold): policy(quiet_seconds=True)
    def test_strict_json_rejects_duplicates_nonfinite_and_oversize(self):
        for raw in [b'{"x":1,"x":2}',b'{"x":NaN}',b'\xff',b'']:
            with self.assertRaises(Hold): strict_json(raw,100)
        with self.assertRaises(Hold): strict_json(b'[]',1)
    def test_https_sender_has_no_redirect_cookie_proxy_or_fallback(self):
        with self.assertRaisesRegex(Hold,'destination'): HTTPSender()('https://elsewhere.example',{},b'')
        from local_work.adapter import NoRedirect
        self.assertIsNone(NoRedirect().redirect_request(None,None,302,'',{},ORIGIN+PATH))

if __name__=='__main__': unittest.main()
