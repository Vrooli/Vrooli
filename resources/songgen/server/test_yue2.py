"""YuE2 adapter guards: no model, network, or CUDA dependency."""
import base64
import tempfile
import unittest
from pathlib import Path
import os
import subprocess
import sys
import time

from yue2 import native_environment, native_failure, native_inputs, resolved_sampling, retain_artifacts


class Yue2ContractTests(unittest.TestCase):
    def setUp(self):
        self.request = {"prompt": "Original syncopated pop", "lyrics": "[Chorus]\nKeep the receipt\n",
                        "duration": 152, "seed": 0, "steps": 8, "guidance": 1.0}
        self.profile = {"model_dir": "models/yue2", "cpu_threads": 4}

    def test_native_request_preserves_words_and_bounds_one_cuda_model(self):
        config, request = native_inputs(Path('/owned'), self.profile, self.request, 12345)
        self.assertEqual((config['host'], config['backend'], config['threads']), ('127.0.0.1', 'cuda', 4))
        self.assertFalse(config['ui'])
        self.assertFalse(config['ui_management'])
        self.assertEqual(config['max_loaded_models'], 1)
        self.assertEqual(request['request']['lyrics'], self.request['lyrics'])
        self.assertEqual(request['request']['seed'], 0)
        self.assertEqual(request['request']['options']['semantic_max_tokens'], 3800)
        self.assertEqual(request['request']['options']['cot'], 'full')

    def test_child_resolves_pinned_nccl_without_importing_another_model(self):
        env = native_environment(Path('/owned'), {'LD_LIBRARY_PATH': '/foreign', 'KEEP': 'yes'})
        self.assertEqual(env['KEEP'], 'yes')
        self.assertEqual(env['LD_LIBRARY_PATH'].split(os.pathsep),
                         ['/owned/upstream/cudart', '/owned/wheels/heartmula/nvidia/nccl/lib'])

    def test_native_failure_survives_staging_cleanup_with_bounded_diagnostics(self):
        with tempfile.TemporaryDirectory() as temp:
            stage = Path(temp)
            (stage/'native-launch.log').write_text('x'*10000 + 'libnccl.so.2 missing')
            (stage/'native.log').write_text('native stage failed')
            error = native_failure(stage, RuntimeError('child exited'))
        text = str(error)
        self.assertIn('child exited', text)
        self.assertIn('libnccl.so.2 missing', text)
        self.assertIn('native stage failed', text)
        self.assertLess(len(text), 6300)

    def test_sampling_requires_actual_runtime_witness_and_matches_all_controls(self):
        log = '[TRACE ts=0] yue2.request seed=0 cot=full stop_after=audio guidance_scale=1 num_inference_steps=8 context=24576 abc=generated\n'
        self.assertEqual(resolved_sampling(log, self.request), {'steps': 8, 'guidance': 1.0, 'seed': 0, 'score_source': 'generated'})
        for changed in ('seed=1', 'guidance_scale=2', 'num_inference_steps=32'):
            key = changed.split('=')[0]
            import re
            bad = re.sub(key + r'=[^ ]+', changed, log)
            with self.subTest(changed=changed), self.assertRaisesRegex(RuntimeError, 'sampling'):
                resolved_sampling(bad, self.request)
        with self.assertRaisesRegex(RuntimeError, 'witness'):
            resolved_sampling('request accepted, no runtime settings', self.request)

    def test_external_score_is_inline_retained_and_witnessed_by_native_runtime(self):
        score = 'X:1\nK:C\nC4 D4 E4 F4|\n'
        request = {**self.request, 'score': score}
        _, native = native_inputs(Path('/owned'), self.profile, request, 12345)
        self.assertEqual(native['request']['options']['abc'], score)
        self.assertNotIn('abc_file', native['request']['options'])
        log = 'yue2.request seed=0 guidance_scale=1 num_inference_steps=8 abc=provided\n'
        self.assertEqual(resolved_sampling(log, request)['score_source'], 'provided')
        with self.assertRaisesRegex(RuntimeError, 'conditioning witness'):
            resolved_sampling(log.replace('abc=provided', 'abc=generated'), request)
        semantic = {'id': 'semantic', 'payload': base64.b64encode(b'[1,2]').decode(),
                    'meta': {'truncated': 'false'}}
        with tempfile.TemporaryDirectory() as temp:
            retained = retain_artifacts(Path(temp), {'artifacts': [semantic]}, score)
            self.assertEqual(retained['score']['source'], 'provided')
            self.assertEqual((Path(temp)/'score.abc').read_text(), score)

    def test_duplicate_witness_is_not_silently_selected(self):
        log = 'yue2.request seed=0 guidance_scale=1 num_inference_steps=8\n'
        with self.assertRaisesRegex(RuntimeError, 'witness'):
            resolved_sampling(log + log, self.request)

    def test_generated_score_and_truncation_are_retained_without_arbitrary_paths(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            score = 'X:1\nT:Original\nK:C\nC D E F|'
            result = {'artifacts': [{'id': 'score', 'kind': 'custom',
                       'payload': base64.b64encode(score.encode()).decode(),
                       'meta': {'truncated': 'true'}},
                       {'id': 'semantic', 'kind': 'custom', 'payload': base64.b64encode(b'[1, 2, 3]').decode(),
                        'meta': {'truncated': 'false'}}]}
            retained = retain_artifacts(root, result)
            self.assertEqual((root/'score.abc').read_text(), score)
            self.assertEqual(retained['score']['truncated'], True)
            self.assertEqual(retained['score']['text'], score)
            result['artifacts'][0]['id'] = '../../escape'
            with self.assertRaisesRegex(RuntimeError, 'artifact'):
                retain_artifacts(root, result)

    def test_missing_score_cannot_pass_full_planning_receipt(self):
        with tempfile.TemporaryDirectory() as temp:
            with self.assertRaisesRegex(RuntimeError, 'score'):
                retain_artifacts(Path(temp), {'artifacts': []})

    def test_native_child_dies_when_worker_disappears(self):
        with tempfile.TemporaryDirectory() as temp:
            pid_path = Path(temp)/'child.pid'
            script = '''
import os, subprocess, sys, time
sys.path.insert(0, sys.argv[2])
from yue2 import bind_child_to_parent
parent = os.getpid()
child = subprocess.Popen([sys.executable, '-c', 'import time;time.sleep(60)'],
    preexec_fn=lambda: bind_child_to_parent(parent))
open(sys.argv[1], 'w').write(str(child.pid))
time.sleep(60)
'''
            worker = subprocess.Popen([sys.executable, '-c', script, str(pid_path), str(Path(__file__).parent)],
                                      stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
            child_pid = None
            try:
                until = time.monotonic()+3
                while not pid_path.exists() and worker.poll() is None and time.monotonic()<until:
                    time.sleep(0.01)
                self.assertTrue(pid_path.exists(), 'fixture worker did not start native child')
                child_pid = int(pid_path.read_text())
                worker.kill()
                worker.wait(timeout=3)
                until = time.monotonic()+3
                while time.monotonic()<until:
                    status = Path(f'/proc/{child_pid}/stat')
                    if not status.exists() or status.read_text().rsplit(')',1)[1].split()[0] == 'Z':
                        break
                    time.sleep(0.01)
                else:
                    self.fail('native child remained active after worker death')
            finally:
                if worker.poll() is None:
                    worker.kill(); worker.wait(timeout=3)
                if worker.stderr:
                    worker.stderr.close()
                if child_pid and Path(f'/proc/{child_pid}').exists():
                    try:
                        os.kill(child_pid, 9)
                    except ProcessLookupError:
                        pass


if __name__ == '__main__':
    unittest.main()
