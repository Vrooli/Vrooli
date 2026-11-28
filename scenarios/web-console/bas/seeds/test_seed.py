"""Black-box seed regressions: real HTTP, shell, JSON parsing and rollback."""
import json
import os
from pathlib import Path
import subprocess
import sqlite3
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

SCRIPT = Path(__file__).with_name('seed.sh')

class SeedTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.events = []
        self.fail_hooks = False
        self.drop_messages = False
        self.messages = {}
        self.routing_files = False
        self.route_flags = []
        owner = self
        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args): pass
            def do_GET(self):
                self.send_response(200); self.end_headers(); self.wfile.write(b'{}')
            def do_POST(self):
                body=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
                owner.events.append((self.path,body))
                owner.route_flags.append((self.path,self.headers.get('X-Vrooli-Test-Mode')))
                status=401 if owner.fail_hooks and '/hooks/' in self.path else 200
                if self.path.endswith('SessionsService/Delete') and body.get('confirmation')!='DELETE:'+body.get('id',''):
                    status=400
                self.send_response(status); self.end_headers()
                payload={'session':{'id':f'session-{len(owner.events)}'}} if self.path.endswith('/Create') else {}
                if self.path.endswith('/stop') and status==200 and not owner.drop_messages:
                    owner.messages.setdefault(body['web_console_session_id'],[]).append({'text':body['last_assistant_message']})
                if self.path.endswith('/InstallTestPool'):
                    payload={'activeLeaseId':body['leaseId'],'fileRootsInstalled':owner.routing_files}
                if self.path.endswith('ConversationService/Get'):
                    payload={'events':owner.messages.get(body['sessionId'],[])}
                self.wfile.write(json.dumps(payload).encode())
        self.server=ThreadingHTTPServer(('127.0.0.1',0),Handler)
        self.thread=threading.Thread(target=self.server.serve_forever,daemon=True);self.thread.start()
        self.addCleanup(self.server.server_close);self.addCleanup(self.server.shutdown)
        (self.root/'token').write_text('fixture-secret\n')
        (self.root/'web-console').write_text('#!/bin/sh\nexit 0\n');(self.root/'web-console').chmod(0o755)
        self.env={**os.environ,'BAS_TARGET_API_BASE':f'http://127.0.0.1:{self.server.server_port}',
                  'BAS_WEB_CONSOLE_HOOK_TOKEN_PATH':str(self.root/'token'), 'TMPDIR':str(self.root),
                  'PATH':str(self.root)+os.pathsep+os.environ['PATH'], 'HOME':str(self.root),'XDG_STATE_HOME':str(self.root/'state')}
        for key in ('TEST_GENIE_TARGET_API_BASE','TEST_GENIE_SEEDS','PLAYBOOKS_SQLITE_DSN',
                    'PLAYBOOKS_SQLITE_PATH','BAS_ROUTED_TEST_MODE','BAS_ROUTING_LEASE_ID','TEST_GENIE_ROUTING_LEASE_ID','BAS_LIVE_WEB_CONSOLE_API_PORT','TEST_GENIE_LIVE_WEB_CONSOLE_API_PORT'):
            self.env.pop(key,None)
    def run_seed(self):
        return subprocess.run(['bash',str(SCRIPT)],env=self.env,capture_output=True,text=True,timeout=20)
    def test_nested_create_response_and_distinct_histories_are_retained(self):
        result=self.run_seed();self.assertEqual(0,result.returncode,result.stderr)
        receipts=list(self.root.glob('web-console-bas-fixtures.*/seed-receipt.json'))
        self.assertEqual(1,len(receipts));receipt=json.loads(receipts[0].read_text())
        self.assertEqual('seeded',receipt['status']);self.assertEqual(2,len(receipt['session_ids']))
        previews=[body['last_assistant_message'] for path,body in self.events if path.endswith('/stop')]
        self.assertEqual(3,len(previews));self.assertEqual(3,len(set(previews)))
        self.assertNotIn('fixture-secret',result.stdout+result.stderr)
        again=self.run_seed();self.assertEqual(0,again.returncode,again.stderr)
        self.assertEqual(2,len(list(self.root.glob('web-console-bas-fixtures.*/seed-receipt.json'))))
    def test_missing_target_token_fails_before_mutation(self):
        self.env['BAS_WEB_CONSOLE_HOOK_TOKEN_PATH']=str(self.root/'absent')
        result=self.run_seed();self.assertNotEqual(0,result.returncode)
        self.assertIn('seed_hook_token_missing',result.stderr);self.assertEqual([],self.events)
    def test_hook_auth_failure_rolls_back_created_sessions_and_fails(self):
        self.fail_hooks=True
        result=self.run_seed();self.assertNotEqual(0,result.returncode)
        self.assertEqual(1,len([p for p,b in self.events if p.endswith('/Create')]))
        deleted=[body['id'] for path,body in self.events if path.endswith('/Delete')]
        self.assertEqual(['session-1'],deleted)
        self.assertNotIn('Seeding complete',result.stdout)
    def test_successful_hook_without_persisted_messages_is_not_success(self):
        self.drop_messages=True
        result=self.run_seed();self.assertNotEqual(0,result.returncode)
        self.assertNotIn('Seeding complete',result.stdout)

    def test_routed_seed_requires_both_storage_legs_before_sessions(self):
        self.env.update(BAS_ROUTED_TEST_MODE='1',BAS_ROUTING_LEASE_ID='test-lease',PLAYBOOKS_SQLITE_DSN='file:'+str(self.root/'test.db'))
        result=self.run_seed();self.assertNotEqual(0,result.returncode)
        self.assertEqual(1,len(self.events));self.assertTrue(self.events[0][0].endswith('/InstallTestPool'))
        self.assertEqual([],list(self.root.glob('web-console-bas-fixtures.*')))

    def test_routed_seed_retains_test_header_across_all_services(self):
        self.routing_files=True
        database=self.root/'test.db'
        connection=sqlite3.connect(database)
        connection.executescript((SCRIPT.parents[2]/'api/internal/sessions/schema.sql').read_text());connection.close()
        self.env.update(BAS_ROUTED_TEST_MODE='1',BAS_ROUTING_LEASE_ID='test-lease',PLAYBOOKS_SQLITE_DSN='file:'+str(database),VROOLI_STORAGE_ROOT=str(self.root))
        result=self.run_seed();self.assertEqual(0,result.returncode,result.stderr)
        for path,header in self.route_flags:
            if not path.endswith('/InstallTestPool'):self.assertEqual('1',header,path)
        self.assertTrue(list((self.root/'fixtures').glob('*/seed-receipt.json')))

    def test_explicit_live_target_is_refused(self):
        self.env['BAS_LIVE_WEB_CONSOLE_API_PORT']=str(self.server.server_port)
        result=self.run_seed();self.assertNotEqual(0,result.returncode)
        self.assertIn('seed_target_live_instance_refused',result.stderr);self.assertEqual([],self.events)

if __name__=='__main__': unittest.main()
