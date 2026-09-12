"""Exercise shipped programs in the production kernel with governed calls replaced."""
import copy
import hashlib
import io
import json
import sys
import unittest
import urllib.error
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[5]
sys.dont_write_bytecode = True
sys.path.insert(0, str(ROOT / 'scenarios/program-runtime/kernel'))
from host.engine import SessionKernel


def library(scenario, name):
    base = ROOT / 'scenarios' / scenario / '.vrooli/program-runtime'
    source = (base / (name + '.py')).read_text()
    declaration = json.loads((base / (name + '.json')).read_text())
    return dict(name=name, scenario=scenario, contract=True, current=True,
                version=int(declaration['version']), source=source, declaration=declaration,
                digest=hashlib.sha256(source.encode()).hexdigest())


def validated(label='infra', **extra):
    return dict(valueJson=json.dumps(label), validated=True, usage={},
                provider='fixture', model='fixture-role', **extra)


class Response:
    def __init__(self, data):
        self.data = json.dumps(data).encode()

    def read(self):
        return self.data

    def __enter__(self):
        return self

    def __exit__(self, *_):
        pass


class ClassificationPrograms(unittest.TestCase):
    def setUp(self):
        self.calls = []
        self.batch = {'results': [validated()], 'usage': {}}
        self.refusal = None
        self.failed_collect = None
        self.specs = [library('ai-gateway', 'classify-batch')]
        self.specs += [library('program-runtime', name) for name in ('batch-inference', 'typed-inference', 'watch-set')]

    def request(self, request, **_):
        body = json.loads(request.data)
        self.calls.append((request.full_url, body))
        if request.full_url.endswith('/start'):
            return Response({'execution_id': 'run-' + body['input']['id']})
        if request.full_url.endswith('/collect'):
            if body['execution_id'] == self.failed_collect:
                raise urllib.error.HTTPError(request.full_url, 503, 'Unavailable', {}, io.BytesIO(b'{"error":"collection unavailable"}'))
            return Response({'execution_id': body['execution_id'], 'status': 'completed'})
        self.assertEqual('ai-gateway/inference/run-batch', body['binding_id'])
        if self.refusal:
            raise urllib.error.HTTPError(request.full_url, 403, 'Refused', {}, io.BytesIO(json.dumps({'error': self.refusal}).encode()))
        return Response(self.batch)

    def run_program(self, name='ai_gateway.classify_batch', inputs=None, nested=True):
        values = inputs if inputs is not None else {'corpus': ['provider timeout'], 'labels': ['infra'], 'instruction': 'Classify the failure.'}
        kernel = SessionKernel(libraries=self.specs, session_id='fixture-session',
                               bridge_url='http://fixture/execute', agent_bridge_url='http://fixture/delegate')
        encoded = json.dumps(json.dumps(values))
        if nested:
            source = f'import json\nprint(json.dumps(lib.{name}(**json.loads({encoded})).head(1)[0]))'
        else:
            source = f'import json\ninputs=json.loads({encoded})\n' + self.specs[0]['source']
        with patch('host.engine.urllib.request.urlopen', side_effect=self.request):
            result = kernel.execute(source, include_materialized=True, program_id='fixture-program', provenance='test')
        self.assertTrue(result['ok'], result.get('error'))
        self.assertEqual(1, len(result['stdout'].splitlines()))
        return json.loads(result['stdout'])

    def test_direct_and_nested_calls_retain_order_usage_and_attribution(self):
        self.batch = {'results': [validated('infra'), validated('user')],
                      'usage': {'inputTokens': '19', 'outputTokens': '4', 'costMicros': '7'}}
        args = {'corpus': ['timeout', 'invalid request'], 'labels': ['infra', 'user'], 'instruction': 'Distinguish causes.'}
        before = copy.deepcopy(args)
        for nested in (False, True):
            result = self.run_program(inputs=args, nested=nested)
            self.assertEqual('ok', result['status'])
            self.assertEqual(['infra', 'user'], result['signals']['labels'])
            self.assertEqual(7, result['signals']['usage']['costMicros'])
            request = self.calls[-1][1]
            self.assertEqual('Distinguish causes.', request['args']['instruction'])
            self.assertEqual('classify.fast', request['args']['role'])
            self.assertEqual({'type': 'string', 'enum': ['infra', 'user']}, json.loads(request['args']['schema_json']))
            self.assertEqual('fixture-program', request['program_id'])
            self.assertEqual('test', request['provenance'])
        self.assertEqual(before, args)
        self.assertEqual(2, len(self.calls))

    def test_invalid_and_unvalidated_items_never_become_successful_labels(self):
        bad = [{'valueJson': '"infra"', 'validated': False}, validated('outside'),
               {'valueJson': '{broken', 'validated': True}, None,
               {'valueJson': '"infra"', 'validated': True, 'error': {'code': 'PROVIDER_FAILED'}}]
        for row in bad:
            with self.subTest(row=row):
                self.batch = {'results': [validated(), row], 'usage': {}}
                result = self.run_program(inputs={'corpus': ['a', 'b'], 'labels': ['infra'], 'instruction': 'Classify.'})
                self.assertEqual('partial', result['status'])
                self.assertEqual(['infra', None], result['signals']['labels'])
                self.assertEqual({'infra': 1}, result['signals']['by_label'])
                self.assertEqual(1, result['signals']['results'][1]['index'])
                self.assertFalse(result['signals']['results'][1]['validated'])
                self.assertEqual('classify:1', result['errors'][0]['where'])

    def test_missing_and_excess_results_are_explicit(self):
        result = self.run_program(inputs={'corpus': ['a', 'b'], 'labels': ['infra'], 'instruction': 'Classify.'})
        self.assertEqual(['infra', None], result['signals']['labels'])
        self.assertEqual('missing_result', result['signals']['results'][1]['error']['class'])
        self.batch['results'].append(validated())
        result = self.run_program()
        self.assertEqual('failed', result['status'])
        self.assertEqual('invalid_response', result['errors'][0]['class'])
        self.assertEqual([], result['signals']['labels'])

    def test_all_invalid_and_malformed_responses_fail(self):
        for batch in ({}, {'results': {}}, {'results': []}, {'results': [validated('outside')]}):
            self.batch = batch
            self.assertEqual('failed', self.run_program()['status'])

    def test_input_bounds_reject_before_inference(self):
        base = {'corpus': ['a'], 'labels': ['infra'], 'instruction': 'Classify.'}
        for key, value in [('corpus', []), ('corpus', ['a'] * 33), ('corpus', ['']),
                           ('corpus', ['é' * 8193]), ('corpus', ['a' * 16384] * 5),
                           ('corpus', [3]), ('labels', []), ('labels', ['x', 'x']),
                           ('labels', [' ']), ('labels', ['é' * 33]), ('labels', [3]),
                           ('instruction', ''), ('instruction', 'é' * 2049)]:
            with self.subTest(key=key, value=str(value)[:30]):
                self.assertEqual('failed', self.run_program(inputs={**base, key: value})['status'])
        self.assertEqual([], self.calls)

    def test_maximum_unicode_evidence_stays_inside_output_budget(self):
        labels = ['😀' * 15 + str(i) for i in range(32)]
        self.batch = {'results': [dict(validated(label), provider='😀' * 120, model='😀' * 120) for label in labels], 'usage': {}}
        result = self.run_program(inputs={'corpus': ['x'] * 32, 'labels': labels, 'instruction': 'Classify.'})
        self.assertEqual('ok', result['status'])
        self.assertLessEqual(len(json.dumps(result).encode()), 65536)

    def test_zero_usage_is_distinct_from_unknown_usage(self):
        self.assertEqual(0, self.run_program()['signals']['usage']['costMicros'])
        for value in (None, {'costMicros': -1}, {'costMicros': True}, {'costMicros': 'bad'}):
            self.batch['usage'] = value
            self.assertIsNone(self.run_program()['signals']['usage'])

    def test_consumers_preserve_failed_evidence_and_child_identity(self):
        for name in ('program_runtime.batch_inference', 'program_runtime.typed_inference'):
            for valid in (True, False):
                self.batch = {'results': [validated('infra' if valid else 'outside')], 'usage': {}}
                result = self.run_program(name, {'corpus': ['x'], 'labels': ['infra']})
                self.assertEqual('ok' if valid else 'failed', result['status'])
                self.assertEqual(1 if valid else 0, result['signals']['validated'])
                self.assertEqual(64, len(result['evidence'][-1]['artifact']['digest']))

    def test_refusal_is_not_retried_or_reported_as_a_label(self):
        for message, status, kind in [('requires an explicit grant', 'refused', 'no_grant'),
                                      ('inference_spend_exceeded', 'refused', 'inference_spend_exceeded'),
                                      ('binding bridge unavailable', 'unavailable', 'scenario_unreachable')]:
            self.calls = []
            self.refusal = message
            result = self.run_program('program_runtime.batch_inference', {'corpus': ['x'], 'labels': ['infra']})
            self.assertEqual(status, result['status'])
            self.assertEqual(kind, result['errors'][0]['class'])
            self.assertEqual([], result['signals']['labels'])
            self.assertEqual(1, len(self.calls))

    def test_delegation_collection_gaps_preserve_original_run_indices(self):
        self.failed_collect = 'run-0'
        requests = [{'owner': 'fixture', 'workflow_key': 'fixture', 'input': {'id': str(i)}} for i in range(2)]
        result = self.run_program('program_runtime.watch_set', {'requests': requests, 'labels': ['infra'], 'wait_seconds': 0})
        self.assertEqual('partial', result['status'])
        self.assertEqual(1, result['signals']['runs'][0]['index'])
        self.assertEqual('infra', result['signals']['runs'][0]['label'])
        self.assertEqual(5, len(self.calls))  # Two starts, two collects, one batch.
        self.batch = {'results': [validated('outside')], 'usage': {}}
        failed = self.run_program('program_runtime.watch_set', {'requests': requests, 'labels': ['infra'], 'wait_seconds': 0})
        self.assertEqual('classify:1', failed['errors'][-1]['where'])
        self.assertFalse(failed['signals']['runs'][0]['validated'])


if __name__ == '__main__':
    unittest.main()
