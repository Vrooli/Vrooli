"""No sockets, credentials or owner calls. All native clients use fake transports."""
import json,pathlib,sys,tempfile,unittest
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[2]))
from local_work.adapter import Hold,Outbox,canonical
from local_work.owners import OwnerClient,OwnerConfig,DurableCollector,LIB,PROGRAMS,NAME
from local_work.recovery import Checkpoints,RecoveryApproval
from local_work.managed import ManagedConfig,ManagedRunner
from test_adapter import policy,receipt,native,envelope,NOW,LOCAL_KEY


def config(**changes):
    values=dict(events_origin='http://127.0.0.1:18001',runtime_origin='http://127.0.0.1:18002',
                expected_digest='a'*64,enabled=True,approval_ref='PUBLIC-TEST-ONLY',not_before=NOW-10,expires_at=NOW+3600)
    values.update(changes);return OwnerConfig(**values)


def program(status='PROGRAM_STATUS_SUCCEEDED',pid='prog_fixture'):
    return {'id':pid,'status':status,'programName':NAME,'programDigest':'a'*64,'stdout':receipt()['program']['stdout']}


class OwnerTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.path=str(pathlib.Path(self.temp.name)/'outbox.db')
        self.box=Outbox(self.path,policy());self.at=NOW+60;self.calls=[]
        self.get={'admissionContractVersion':2,'found':False};self.run={'program':program()}
        self.wait={'terminal':True,'program':program()}
        self.client=OwnerClient(config(),self.transport,lambda owner:'Bearer PUBLIC-TEST-ONLY',lambda:self.at)
        self.collector=DurableCollector(self.box,self.client)
    def tearDown(self):self.box.close();self.temp.cleanup()
    def transport(self,method,url,headers,body,timeout,maximum):
        self.calls.append((method,url,headers,json.loads(body) if body else None,timeout))
        if method=='GET':result=[envelope()]
        elif url.endswith('GetDeclaredExecution'):result=self.get
        elif url.endswith('RunDeclaredProgram'):result=self.run
        elif url.endswith('WaitForProgram'):result=self.wait
        else:self.fail('unexpected operation')
        return 200,canonical(result).encode()
    def collect(self,ids=('event-1',)):
        with self.client.bounded_cycle():return self.collector.collect_for(ids,self.at)
    def test_loopback_origin_and_input_contracts_fail_closed(self):
        for origin in ['http://localhost:1234','https://127.0.0.1:1234','http://127.0.0.1:1/path','http://user@127.0.0.1:1','http://192.168.1.1:1234']:
            with self.assertRaises(Hold):config(events_origin=origin)
        with self.assertRaises(Hold):config(inputs_json='{"unknown":"x"}')
    def test_disabled_access_performs_no_request(self):
        self.client.config=config(enabled=False)
        with self.client.bounded_cycle(),self.assertRaisesRegex(Hold,'disabled'):self.client.resolve('event-1')
        self.assertFalse(self.calls)
    def test_exact_id_resolution_uses_verified_owner_shape(self):
        with self.client.bounded_cycle():self.assertEqual(self.client.resolve('event-1'),envelope())
        self.assertIn('/api/v1/events?event_id=event-1&limit=2',self.calls[0][1])
    def test_wrong_or_multiple_exact_results_rejected(self):
        for rows in [[envelope('wrong')],[envelope(),envelope()]]:
            self.client.transport=lambda *args:(200,canonical(rows).encode())
            with self.client.bounded_cycle(),self.assertRaises(Hold):self.client.resolve('event-1')
    def test_empty_exact_result_is_not_fake_occurrence(self):
        self.client.transport=lambda *args:(200,b'[]')
        with self.client.bounded_cycle():self.assertIsNone(self.client.resolve('event-1'))
    def test_owner_permission_denial_is_distinct(self):
        self.client.transport=lambda *args:(403,b'PRIVATE_ERROR')
        with self.client.bounded_cycle(),self.assertRaisesRegex(Hold,'permission_hold'):self.client.resolve('event-1')
    def test_budget_required_and_timeout_clamped(self):
        with self.assertRaisesRegex(Hold,'budget'):self.client.resolve('event-1')
        with self.client.bounded_cycle(2):self.client.resolve('event-1')
        self.assertEqual(self.calls[0][4],2)
    def test_expired_budget_stops_without_request(self):
        with self.client.bounded_cycle(1):
            self.at+=2
            with self.assertRaisesRegex(Hold,'deadline'):self.client.resolve('event-1')
        self.assertFalse(self.calls)
    def test_principal_is_injected_never_invented(self):
        self.client.authorization=lambda owner:''
        with self.client.bounded_cycle(),self.assertRaisesRegex(Hold,'principal'):self.client.resolve('event-1')
        self.assertFalse(self.calls)
    def test_pinned_durable_admission_contract(self):
        result=self.collect();self.assertEqual(result['program']['program_digest'],'a'*64)
        request=self.calls[1][3]
        self.assertEqual(request['name'],NAME);self.assertTrue(request['async'])
        self.assertEqual(request['provenance'],'PROVENANCE_AGENT');self.assertNotIn('caller',request)
        self.assertIn('admissionDeadline',request);self.assertIn('idempotencyKey',request)
    def test_lost_admission_response_resumes_original_program(self):
        run_calls=[]
        original=self.client.transport
        def uncertain(method,url,headers,body,timeout,maximum):
            if url.endswith('RunDeclaredProgram'):
                run_calls.append(json.loads(body));self.get={'admissionContractVersion':2,'found':True,'program':program('PROGRAM_STATUS_RUNNING')}
                raise TimeoutError()
            return original(method,url,headers,body,timeout,maximum)
        self.client.transport=uncertain
        with self.assertRaises(TimeoutError):self.collect()
        self.box.close();self.box=Outbox(self.path,policy());self.collector=DurableCollector(self.box,self.client)
        result=self.collect();self.assertEqual(result['program']['id'],'prog_fixture');self.assertEqual(len(run_calls),1)
        self.assertTrue(any(c[1].endswith('WaitForProgram') for c in self.calls))
    def test_expired_missing_admission_never_reexecutes(self):
        self.get={'admissionContractVersion':1}
        with self.assertRaises(Hold):self.collect()
        self.get={'admissionContractVersion':2,'found':False};self.at+=700
        with self.assertRaisesRegex(Hold,'expired'):self.collect()
        self.assertFalse(any(c[1].endswith('RunDeclaredProgram') for c in self.calls))
    def test_failed_original_program_does_not_create_new_run(self):
        self.get={'admissionContractVersion':2,'found':True,'program':program('PROGRAM_STATUS_FAILED')}
        with self.assertRaisesRegex(Hold,'terminal_failure'):self.collect()
        self.assertFalse(any(c[1].endswith('RunDeclaredProgram') for c in self.calls))
    def test_live_program_waits_once_then_preserves_identity(self):
        self.get={'admissionContractVersion':2,'found':True,'program':program('PROGRAM_STATUS_RUNNING')}
        self.wait={'program':program('PROGRAM_STATUS_RUNNING')} # proto false omitted
        with self.assertRaisesRegex(Hold,'still_running'):self.collect()
        self.assertEqual(sum(c[1].endswith('WaitForProgram') for c in self.calls),1)
        self.get={'admissionContractVersion':2,'found':False}
        with self.assertRaisesRegex(Hold,'identity_lost'):self.collect()
    def test_program_digest_and_wait_id_cannot_drift(self):
        self.get={'admissionContractVersion':2,'found':True,'program':program('PROGRAM_STATUS_RUNNING')}
        self.wait={'terminal':True,'program':program(pid='other')}
        with self.assertRaisesRegex(Hold,'wait_identity'):self.collect()
    def test_changed_scope_refused_without_new_admission(self):
        self.collect();self.client.config=config(inputs_json='{"row_limit":2}')
        with self.assertRaisesRegex(Hold,'scope_drift'):self.collect()
    def test_stale_collector_cannot_admit_or_update_intent(self):
        self.box.db.execute("INSERT INTO meta VALUES ('collection_lease',?)",(canonical({'generation':2,'until':self.at+300}),))
        with self.client.bounded_cycle(),self.assertRaisesRegex(Hold,'lease_lost'):
            self.collector.collect_for(['event-1'],self.at,collection_generation=1)
        self.assertFalse(self.calls)
        self.assertEqual(self.box.db.execute('SELECT count(*) FROM collector_intents').fetchone()[0],0)
    def test_paths_tokens_and_receipt_text_not_saved_in_intents(self):
        self.client.config=config(inputs_json='{"research_root":"/tmp/PRIVATE_RESEARCH_PATH"}')
        self.collect();raw=pathlib.Path(self.path).read_bytes()
        for token in [b'PRIVATE_RESEARCH_PATH',b'PRIVATE_TEXT',b'Bearer PUBLIC-TEST-ONLY']:self.assertNotIn(token,raw)


class RecoveryTests(unittest.TestCase):
    def setUp(self):self.box=Outbox(':memory:',policy());self.cp=Checkpoints(self.box)
    def tearDown(self):self.box.close()
    def approval(self,ref='approval-1'):
        return RecoveryApproval(ref,'swarminator','fixture-epoch',self.cp.inspect()['state_digest'],NOW-1,NOW+600)
    def test_checkpoint_remains_local_unknown_coverage(self):
        result=self.cp.record_baseline(self.approval(),receipt(),NOW)
        self.assertEqual(result['historical_coverage'],'unknown');self.assertIn('unknown_checkpoint',result['remaining_holds'])
        self.assertEqual(self.box.pending(),[])
    def test_baseline_permission_hold_survives_clean_collection(self):
        self.cp.record_baseline(self.approval(),receipt(errors=[{'class':'no_grant'}]),NOW)
        self.box.accept_native(*native(),LOCAL_KEY,NOW);self.box.qualify_event('event-1',envelope(),NOW)
        packet=self.box.prepare(['event-1'],receipt(),NOW)
        self.assertTrue(packet['snapshot']['permission_hold'])
    def test_baseline_uses_strict_shared_snapshot_boundary(self):
        value=receipt();value['program']['stdout']=value['program']['stdout'].replace('{','{"program":"duplicate",',1)
        with self.assertRaises(Hold):self.cp.record_baseline(self.approval(),value,NOW)
        self.assertEqual(self.box.db.execute('SELECT count(*) FROM recovery_checkpoints').fetchone()[0],0)
    def test_checkpoint_cas_rejects_intervening_event(self):
        approval=self.approval();self.box.accept_native(*native(),LOCAL_KEY,NOW)
        with self.assertRaisesRegex(Hold,'state_changed'):self.cp.record_baseline(approval,receipt(),NOW)
    def test_checkpoint_retry_is_idempotent(self):
        approval=self.approval();self.cp.record_baseline(approval,receipt(),NOW)
        self.assertEqual(self.cp.record_baseline(approval,receipt(),NOW)['state'],'duplicate')
    def test_resume_preserves_holds_and_original_selection(self):
        self.box.db.execute("INSERT INTO meta VALUES ('collection_retry',?)",(canonical({'attempts':5,'due':NOW,'held':True}),))
        self.box.db.execute("INSERT INTO meta VALUES ('active_collection_ids','[\"event-1\"]')")
        self.box.db.execute("INSERT INTO holds VALUES ('permission_hold',?)",(NOW,))
        result=self.cp.resume_same_collection(self.approval(),NOW)
        self.assertFalse(result['new_execution_authorized']);self.assertIn('permission_hold',result['remaining_holds'])
        self.assertIsNotNone(self.box.db.execute("SELECT 1 FROM meta WHERE key='active_collection_ids'").fetchone())
    def test_recovery_cannot_clear_identity_conflict_or_live_lease(self):
        self.box.db.execute("INSERT INTO meta VALUES ('collection_retry',?)",(canonical({'attempts':5,'held':True}),))
        self.box.db.execute("INSERT INTO meta VALUES ('collection_lease',?)",(canonical({'generation':1,'until':NOW+300}),))
        with self.assertRaisesRegex(Hold,'in_flight'):self.cp.resume_same_collection(self.approval(),NOW)
        self.box.db.execute("DELETE FROM meta WHERE key='collection_lease'")
        self.box.db.execute("INSERT INTO holds VALUES ('event_identity_conflict',?)",(NOW,))
        with self.assertRaisesRegex(Hold,'identity_conflict'):self.cp.resume_same_collection(self.approval(),NOW)


class ManagedTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.path=str(pathlib.Path(self.temp.name)/'outbox.db');self.calls=[]
    def tearDown(self):self.temp.cleanup()
    def runner(self,enabled=False):
        def credentials(purpose):self.calls.append(purpose);return LOCAL_KEY
        return ManagedRunner(ManagedConfig(policy(),config(),enabled=enabled),lambda:NOW,
            lambda p:Outbox(self.path,p),credentials,lambda *a:self.fail('owner transport'),lambda *a:self.fail('cloud transport'))
    def test_disabled_runner_has_no_side_effect(self):
        self.assertEqual(self.runner().dispatch('accept_native')['http_status'],404)
        self.assertFalse(pathlib.Path(self.path).exists());self.assertFalse(self.calls)
    def test_native_commit_matches_go_204_abi(self):
        raw,headers=native();result=self.runner(True).dispatch('accept_native',raw,
            headers['X-Vrooli-Events-Signature'],'event-1','event-1')
        self.assertEqual(result['http_status'],204)
        b=Outbox(self.path,policy());self.assertEqual(b.pending()[0]['state'],'awaiting_envelope');b.close()
    def test_bad_signature_is_fixed_401(self):
        raw,headers=native();result=self.runner(True).dispatch('accept_native',raw,'sha256='+'0'*64,'event-1','event-1')
        self.assertEqual(result,{'http_status':401,'state':'native_auth_refused'})
    def test_idle_tick_does_not_read_credentials_or_owners(self):
        self.assertEqual(self.runner(True).dispatch('tick')['http_status'],204);self.assertFalse(self.calls)
    def test_recovery_is_not_a_worker_operation(self):
        self.assertEqual(self.runner(True).dispatch('resume_same_collection')['http_status'],400)
        self.assertFalse(pathlib.Path(self.path).exists())

if __name__=='__main__':unittest.main()
