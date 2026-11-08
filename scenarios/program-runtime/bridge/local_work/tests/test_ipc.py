"""Public-fixture IPC tests. Finite local children only, no listeners/network."""
import base64,copy,dataclasses,io,json,os,pathlib,subprocess,sys,tempfile,threading,time,unittest
from unittest.mock import patch
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[2]))
from local_work.ipc import SCHEMA,MAX_INPUT,PipeCredentials,parse_packet,serve_packet,Hold
from local_work.adapter import canonical
from test_adapter import policy,native,LOCAL_KEY
from test_managed import config

ENTRY=str(pathlib.Path(__file__).resolve().parents[1]/'ipc_entry.py')


def packet(root,operation='accept_native',enabled=True):
    raw,headers=native()
    now=int(time.time())
    request={'body_base64':base64.b64encode(raw).decode(),'signature':headers['X-Vrooli-Events-Signature'],
             'event_id':'event-1','idempotency_key':'event-1'}
    if operation=='tick':request={key:'' for key in request}
    return {'schema':SCHEMA,'operation':operation,'request':request,
        'config':{'enabled':enabled,'storage_root':root,'policy':dataclasses.asdict(policy()),
          'owners':dataclasses.asdict(config(not_before=now-10,expires_at=now+3600)),
          'export':{'enabled':False}}}


def fixture_response(value=LOCAL_KEY):return canonical({'value_base64':base64.b64encode(value).decode()}).encode()+b'\n'


class IPCTests(unittest.TestCase):
    def setUp(self):self.temp=tempfile.TemporaryDirectory();self.root=self.temp.name
    def tearDown(self):self.temp.cleanup()
    def execute(self,value,response=None):
        requests=io.BytesIO();responses=io.BytesIO(response if response is not None else fixture_response())
        with patch('socket.create_connection',side_effect=AssertionError('network forbidden')):
            result=serve_packet(canonical(value).encode(),requests,responses)
        return result,requests.getvalue()
    def test_native_hmac_precedes_database_and_exact_status_only(self):
        value=packet(self.root);value['request']['signature']='sha256='+'0'*64
        result,requests=self.execute(value)
        self.assertEqual(result,{'http_status':401});self.assertEqual(os.listdir(self.root),[])
        self.assertEqual(requests,b'{"purpose":"native_webhook_verify"}\n')
    def test_durable_acceptance_uses_fixed_storage_name(self):
        result,_=self.execute(packet(self.root))
        self.assertEqual(result,{'http_status':204})
        self.assertEqual(os.listdir(self.root),['local-work-outbox.sqlite3'])
    def test_disabled_does_not_touch_storage_or_credentials(self):
        value=packet('/should/not/exist',enabled=False);value['config']={'enabled':False}
        result,requests=self.execute(value,b'')
        self.assertEqual(result,{'http_status':404});self.assertFalse(requests);self.assertFalse(os.listdir(self.root))
    def test_idle_tick_has_no_credential_or_network_requests(self):
        result,requests=self.execute(packet(self.root,operation='tick'),b'')
        self.assertEqual(result,{'http_status':204});self.assertFalse(requests)
    def test_config_cannot_be_overridden_from_native_body(self):
        value=packet(self.root);value['request']['config']={'enabled':True}
        result,requests=self.execute(value)
        self.assertEqual(result,{'http_status':400});self.assertFalse(requests)
    def test_unknown_fields_and_secret_in_initial_packet_rejected(self):
        for location,key in [('packet','secrets'),('config','credentials'),('policy','command'),('owners','authorization')]:
            value=packet(self.root)
            target=value if location=='packet' else value['config'] if location=='config' else value['config'][location]
            target[key]='PRIVATE_TEST_MARKER'
            result,requests=self.execute(value)
            self.assertEqual(result,{'http_status':400});self.assertFalse(requests)
    def test_storage_root_and_b64_are_bounded(self):
        for root in ['relative','/tmp/../escape','/tmp/bad\npath']:
            result,_=self.execute(packet(root));self.assertEqual(result,{'http_status':400})
        for body in ['!invalid!',base64.b64encode(b'x'*65537).decode()]:
            value=packet(self.root);value['request']['body_base64']=body
            result,requests=self.execute(value);self.assertEqual(result,{'http_status':400});self.assertFalse(requests)
    def test_unsupported_operations_and_nonempty_tick_refused(self):
        for operation in ['recover','shell','collect_for']:
            value=packet(self.root);value['operation']=operation
            result,_=self.execute(value);self.assertEqual(result,{'http_status':400})
        value=packet(self.root);value['operation']='tick'
        result,_=self.execute(value);self.assertEqual(result,{'http_status':400})
    def test_invalid_or_unavailable_credentials_never_echo(self):
        for response in [b'{"error":"unavailable"}\n',b'{"value_base64":"","value_base64":"PRIVATE_TEST_MARKER"}\n',b'x'*17000]:
            result,_=self.execute(packet(self.root),response)
            self.assertEqual(result,{'http_status':503});self.assertNotIn('PRIVATE_TEST_MARKER',str(result));self.assertFalse(os.listdir(self.root))
    def test_credential_purpose_and_caching(self):
        requests=io.BytesIO();provider=PipeCredentials(requests,io.BytesIO(fixture_response()),'accept_native',False)
        self.assertEqual(provider('native_webhook_verify'),LOCAL_KEY);self.assertEqual(provider('native_webhook_verify'),LOCAL_KEY)
        self.assertEqual(requests.getvalue().count(b'\n'),1)
        for purpose in ['events_reader','flywheel_submit','unknown']:
            with self.assertRaises(Hold):provider(purpose)
    def test_tick_cloud_key_requires_explicit_export_scope(self):
        provider=PipeCredentials(io.BytesIO(),io.BytesIO(fixture_response(b'x'*32)),'tick',False)
        with self.assertRaises(Hold):provider('flywheel_submit')
    def test_duplicate_initial_json_keys_are_rejected(self):
        raw=canonical(packet(self.root)).encode().replace(b'{',b'{"schema":"duplicate",',1)
        result=serve_packet(raw,io.BytesIO(),io.BytesIO())
        self.assertEqual(result,{'http_status':400})


@unittest.skipUnless(os.name=='posix','production pipe runner targets POSIX')
class FiniteEntryTests(unittest.TestCase):
    def setUp(self):self.temp=tempfile.TemporaryDirectory();self.root=self.temp.name
    def tearDown(self):self.temp.cleanup()
    def simple_child(self,raw):
        return subprocess.run([sys.executable,'-I','-S',ENTRY],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE,
                              env={},timeout=8,check=True)
    def test_disabled_entry_without_secret_fds_is_inert(self):
        value=packet(self.root,enabled=False);value['config']={'enabled':False}
        result=self.simple_child(canonical(value).encode())
        self.assertEqual(result.stdout,b'{"http_status":404}\n');self.assertFalse(result.stderr);self.assertFalse(os.listdir(self.root))
    def test_unpinned_parent_cannot_shadow_standard_library(self):
        import shutil
        parent=pathlib.Path(self.root)/'bundle';parent.mkdir()
        package=parent/'local_work';shutil.copytree(pathlib.Path(ENTRY).parent,package,
            ignore=shutil.ignore_patterns('__pycache__','tests'))
        (parent/'base64.py').write_text("raise RuntimeError('UNTRUSTED_SHADOW_LOADED')\n")
        value=packet(self.root,enabled=False);value['config']={'enabled':False}
        result=subprocess.run([sys.executable,'-I','-S',str(package/'ipc_entry.py')],input=canonical(value).encode(),
            stdout=subprocess.PIPE,stderr=subprocess.PIPE,env={},timeout=8,check=True)
        self.assertEqual(result.stdout,b'{"http_status":404}\n');self.assertFalse(result.stderr)
    def test_source_loader_ignores_unreviewed_bytecode(self):
        import shutil,py_compile
        parent=pathlib.Path(self.root)/'bundle';parent.mkdir();package=parent/'local_work'
        shutil.copytree(pathlib.Path(ENTRY).parent,package,ignore=shutil.ignore_patterns('__pycache__','tests'))
        source=package/'ipc.py';original=source.read_bytes();metadata=source.stat()
        malicious=b"raise RuntimeError('UNTRUSTED_BYTECODE_LOADED')\n"
        source.write_bytes(malicious+b'#'*(len(original)-len(malicious)))
        os.utime(source,(metadata.st_atime,metadata.st_mtime));py_compile.compile(str(source),doraise=True)
        source.write_bytes(original);os.utime(source,(metadata.st_atime,metadata.st_mtime))
        value=packet(self.root,enabled=False);value['config']={'enabled':False}
        result=subprocess.run([sys.executable,'-I','-S',str(package/'ipc_entry.py')],input=canonical(value).encode(),
            stdout=subprocess.PIPE,stderr=subprocess.PIPE,env={},timeout=8,check=True)
        self.assertEqual(result.stdout,b'{"http_status":404}\n');self.assertFalse(result.stderr)
    def test_source_loader_ignores_unreviewed_package_and_extension(self):
        import shutil,importlib.machinery
        parent=pathlib.Path(self.root)/'bundle';parent.mkdir();package=parent/'local_work'
        shutil.copytree(pathlib.Path(ENTRY).parent,package,ignore=shutil.ignore_patterns('__pycache__','tests'))
        (package/'ipc').mkdir();(package/'ipc'/'__init__.py').write_text("raise RuntimeError('UNTRUSTED_PACKAGE_LOADED')\n")
        (package/('ipc'+importlib.machinery.EXTENSION_SUFFIXES[0])).write_bytes(b'untrusted invalid extension')
        value=packet(self.root,enabled=False);value['config']={'enabled':False}
        result=subprocess.run([sys.executable,'-I','-S',str(package/'ipc_entry.py')],input=canonical(value).encode(),
            stdout=subprocess.PIPE,stderr=subprocess.PIPE,env={},timeout=8,check=True)
        self.assertEqual(result.stdout,b'{"http_status":404}\n');self.assertFalse(result.stderr)
    def test_oversize_and_malformed_input_return_only_fixed_status(self):
        for raw,expected in [(b'x'*(MAX_INPUT+1),413),(b'{"secret":"PRIVATE_TEST_MARKER"}',400)]:
            result=self.simple_child(raw)
            self.assertEqual(result.stdout,f'{{"http_status":{expected}}}\n'.encode());self.assertFalse(result.stderr)
    def test_real_entry_private_lazy_pipes_and_durable_ack(self):
        request_read,request_write=os.pipe();response_read,response_write=os.pipe()
        # Test-only finite launcher maps already-open pipes. No credential is in
        # args/env/stdin. Production Go runner owns ExtraFiles and process reaping.
        launcher="import os,sys; a=int(sys.argv[1]); b=int(sys.argv[2]); os.dup2(a,3); os.dup2(b,4); [os.close(x) for x in {a,b} if x not in {3,4}]; os.execve(sys.executable,[sys.executable,'-I','-S',sys.argv[3]],{})"
        process=subprocess.Popen([sys.executable,'-I','-S','-c',launcher,str(request_write),str(response_read),ENTRY],
            stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,pass_fds=(request_write,response_read),env={})
        os.close(request_write);os.close(response_read);observed=[];errors=[]
        def provider():
            try:
                with os.fdopen(request_read,'rb') as source,os.fdopen(response_write,'wb',buffering=0) as target:
                    for line in source:
                        observed.append(json.loads(line));target.write(fixture_response())
            except Exception as error:errors.append(type(error).__name__)
        worker=threading.Thread(target=provider);worker.start()
        try:
            stdout,stderr=process.communicate(canonical(packet(self.root)).encode(),timeout=8)
        finally:
            if process.poll() is None:process.kill();process.wait()
            worker.join(timeout=3)
        self.assertFalse(worker.is_alive());self.assertFalse(errors)
        self.assertEqual(process.returncode,0);self.assertEqual(stdout,b'{"http_status":204}\n');self.assertFalse(stderr)
        self.assertEqual(observed,[{'purpose':'native_webhook_verify'}])
        self.assertEqual(os.listdir(self.root),['local-work-outbox.sqlite3'])

if __name__=='__main__':unittest.main()
