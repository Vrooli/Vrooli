"""Exercise ai-gateway.decide-batch in the production kernel with the governed binding faked."""
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
from host.engine import SessionKernel  # noqa: E402

BASE = ROOT / 'scenarios/ai-gateway/.vrooli/program-runtime'
SOURCE = (BASE / 'decide-batch.py').read_text()
CONTRACT = json.loads((BASE / 'decide-batch.json').read_text())
LIBRARY = dict(name='decide-batch', scenario='ai-gateway', contract=True, current=True,
               version=int(CONTRACT['version']), source=SOURCE, declaration=CONTRACT,
               digest=hashlib.sha256(SOURCE.encode()).hexdigest())
BINDING = dict(id='ai-gateway/inference/decide-batch', scenario='ai-gateway', group='inference',
               command='decide_batch', effect='read', rows_field='results', meta_fields=['usage'])

QUESTIONS = [
    {'id': 'kind', 'type': 'choice', 'instructions': 'What is the operator doing?',
     'options': {'correction': 'Correcting the agent', 'approval': 'Approving', 'question': 'Asking'}},
    {'id': 'stop', 'type': 'noul', 'instructions': 'The operator wants the agent to stop.'},
]


def answered(choice='correction', noul=0.82, method='DECISION_METHOD_DECISION_MODEL', **extra):
    """One DecideResponse in protojson form, as the bridge returns it."""
    return dict(
        answers=[{'questionId': 'kind', 'type': 'DECISION_QUESTION_TYPE_CHOICE', 'choice': choice, 'confidence': 0.61,
                  'probabilities': [{'option': 'correction', 'probability': 0.74}, {'option': 'approval', 'probability': 0.11},
                                    {'option': 'question', 'probability': 0.15}]},
                 {'questionId': 'stop', 'type': 'DECISION_QUESTION_TYPE_NOUL', 'noul': noul}],
        provider='ollaya', model='laya:en', method=method, locality='local', latencyMs='6',
        usage={'inputTokens': '103'},
        candidates=[{'provider': 'ollaya', 'resourceRole': 'decide.local', 'status': 'DECISION_CANDIDATE_STATUS_ANSWERED'},
                    {'provider': 'ollaya', 'resourceRole': 'decide.jev.edenai', 'status': 'DECISION_CANDIDATE_STATUS_NOT_ATTEMPTED'}],
        **extra)


class Response:
    def __init__(self, data):
        self.data = json.dumps(data).encode()

    def read(self):
        return self.data

    def __enter__(self):
        return self

    def __exit__(self, *_):
        pass


class DecideBatchProgram(unittest.TestCase):
    def setUp(self):
        self.calls = []
        self.batch = {'results': [answered(), answered('approval', 0.04)], 'usage': {'inputTokens': '206', 'costMicros': '0'}}
        self.failure = None

    def request(self, request, **_):
        if request.full_url.endswith('/reachability'):
            return Response({'ai-gateway': {'reachable': True}})
        body = json.loads(request.data)
        self.calls.append(body)
        self.assertEqual('ai-gateway/inference/decide-batch', body['binding_id'])
        if self.failure:
            status, payload = self.failure
            raise urllib.error.HTTPError(request.full_url, status, 'fixture', {}, io.BytesIO(json.dumps(payload).encode()))
        return Response(self.batch)

    def run_program(self, inputs=None):
        values = inputs if inputs is not None else {'states': ['No, stop, revert it.', 'Looks great, ship it.'], 'questions': QUESTIONS}
        kernel = SessionKernel(bindings=[BINDING], libraries=[LIBRARY], session_id='fixture-session',
                               bridge_url='http://fixture/execute', agent_bridge_url='http://fixture/delegate')
        source = f'import json\ninputs=json.loads({json.dumps(json.dumps(values))})\n' + SOURCE
        with patch('host.engine.urllib.request.urlopen', side_effect=self.request):
            result = kernel.execute(source, include_materialized=True, program_id='fixture-program', provenance='test')
        self.assertTrue(result['ok'], result.get('error'))
        self.assertEqual(1, len(result['stdout'].splitlines()), 'the envelope prints exactly once')
        return json.loads(result['stdout'])

    def test_answers_keep_order_distribution_method_and_usage(self):
        inputs = {'states': ['No, stop, revert it.', 'Looks great, ship it.'], 'questions': QUESTIONS}
        before = copy.deepcopy(inputs)
        result = self.run_program(inputs)
        self.assertEqual(before, inputs)
        self.assertEqual('ok', result['status'])
        signals = result['signals']
        self.assertEqual(2, signals['answered'])
        first = signals['results'][0]
        self.assertEqual('correction', first['answers']['kind']['choice'])
        self.assertEqual({'kind': ['correction', 'approval', 'question'], 'stop': None}, result['inputs']['questions'])
        self.assertEqual([0.74, 0.11, 0.15], first['answers']['kind']['probabilities'], 'listed in declared option order')
        self.assertEqual(0.82, first['answers']['stop']['noul'])
        self.assertEqual(['laya:en', 'local', 6], [first['model'], first['locality'], first['latency_ms']])
        self.assertEqual({'kind': {'correction': 1, 'approval': 1}}, signals['by_choice'])
        self.assertEqual({'DECISION_METHOD_DECISION_MODEL': 2}, signals['methods'])
        self.assertEqual({'inputTokens': 206, 'outputTokens': 0, 'costMicros': 0}, signals['usage'])
        self.assertEqual('DECISION_CANDIDATE_STATUS_NOT_ATTEMPTED', signals['candidates']['ollaya/decide.jev.edenai'])
        self.assertNotIn('revert', json.dumps(result), 'state text never reaches the envelope')

        request = self.calls[0]['args']
        self.assertEqual('decide.fast', request['role'])
        self.assertEqual([{'state': 'No, stop, revert it.'}, {'state': 'Looks great, ship it.'}], request['items'])
        self.assertEqual({'id': 'kind', 'type': 'DECISION_QUESTION_TYPE_CHOICE', 'instructions': 'What is the operator doing?',
                          'options': [{'label': 'correction', 'description': 'Correcting the agent'},
                                      {'label': 'approval', 'description': 'Approving'},
                                      {'label': 'question', 'description': 'Asking'}]}, request['questions'][0])
        self.assertEqual('DECISION_QUESTION_TYPE_NOUL', request['questions'][1]['type'])

    def test_skipped_hosted_candidates_are_reported_by_reason(self):
        row = answered(method='DECISION_METHOD_LLM_ENUM')
        row['candidates'] = [{'provider': 'ollaya', 'resourceRole': 'decide.jev.edenai', 'status': 'DECISION_CANDIDATE_STATUS_SKIPPED', 'reason': 'unconfigured'}]
        self.batch = {'results': [row], 'usage': {}}
        result = self.run_program({'states': ['x'], 'questions': QUESTIONS})
        self.assertEqual({'ollaya/decide.jev.edenai': 'unconfigured'}, result['signals']['candidates'])
        self.assertEqual({'DECISION_METHOD_LLM_ENUM': 1}, result['signals']['methods'])

    def test_answers_outside_the_closed_set_never_count(self):
        partial = answered()
        partial['answers'][0]['probabilities'] = partial['answers'][0]['probabilities'][:2]
        bad_rows = [answered('refund'), answered(noul=1.4), dict(answered(), answers=[]), partial,
                    {'error': {'code': 'INFERENCE_ERROR_CODE_UNAVAILABLE'}}, None]
        for row in bad_rows:
            with self.subTest(row=str(row)[:60]):
                self.batch = {'results': [answered(), row], 'usage': {}}
                result = self.run_program()
                self.assertEqual('partial', result['status'])
                self.assertEqual(1, result['signals']['answered'])
                self.assertIsNone(result['signals']['results'][1]['answers'])
                self.assertEqual('decide:1', result['errors'][0]['where'])
        self.batch = {'results': [answered(), {'error': {'code': 'INFERENCE_ERROR_CODE_UNAVAILABLE'}}], 'usage': {}}
        self.assertEqual('INFERENCE_ERROR_CODE_UNAVAILABLE', self.run_program()['signals']['results'][1]['error']['code'])

    def test_score_answers_are_bounded_by_their_levels(self):
        questions = [{'id': 'frustration', 'type': 'score', 'levels': ['calm', 'annoyed', 'angry']}]
        row = {'answers': [{'questionId': 'frustration', 'type': 'DECISION_QUESTION_TYPE_SCORE', 'score': 1.6, 'confidence': 0.55,
                            'probabilities': [{'option': '0', 'probability': 0.1}, {'option': '1', 'probability': 0.2}, {'option': '2', 'probability': 0.7}]}]}
        self.batch = {'results': [row], 'usage': {}}
        answer = self.run_program({'states': ['x'], 'questions': questions})['signals']['results'][0]['answers']['frustration']
        self.assertEqual((1.6, [0.1, 0.2, 0.7]), (answer['score'], answer['probabilities']))
        row['answers'][0]['score'] = 2.5
        self.assertEqual('failed', self.run_program({'states': ['x'], 'questions': questions})['status'])

    def test_missing_and_excess_results_are_explicit(self):
        self.batch = {'results': [answered()], 'usage': {}}
        result = self.run_program()
        self.assertEqual('missing_result', result['signals']['results'][1]['error']['class'])
        self.batch = {'results': [answered()] * 3, 'usage': {}}
        result = self.run_program()
        self.assertEqual('failed', result['status'])
        self.assertEqual('invalid_response', result['errors'][0]['class'])

    def test_input_bounds_reject_before_any_call(self):
        base = {'states': ['a'], 'questions': QUESTIONS}
        cases = [('states', []), ('states', ['a'] * 33), ('states', ['']), ('states', [3]), ('states', ['x' * 16385]),
                 ('questions', []), ('questions', QUESTIONS * 3), ('questions', [QUESTIONS[0], QUESTIONS[0]]),
                 ('questions', [{'id': 'k', 'type': 'choice', 'options': ['only']}]),
                 ('questions', [{'id': 'k', 'type': 'choice', 'options': [str(i) for i in range(13)]}]),
                 ('questions', [{'id': 'k', 'type': 'score', 'levels': ['one']}]),
                 ('questions', [{'id': 'k', 'type': 'box'}]), ('role', 'classify.fast')]
        for key, value in cases:
            with self.subTest(key=key, value=str(value)[:40]):
                result = self.run_program({**base, key: value})
                self.assertEqual('failed', result['status'])
                self.assertEqual('invalid_input', result['errors'][0]['class'])
        self.assertEqual([], self.calls)

    def test_bridge_refusals_keep_their_class(self):
        for status, payload, expected in [(403, {'error': 'requires an explicit grant'}, ('refused', 'no_grant')),
                                          (400, {'error': 'bad', 'class': 'remote_error', 'status': 'failed', 'http_status': 400}, ('failed', 'remote_error'))]:
            with self.subTest(status=status):
                self.calls, self.failure = [], (status, payload)
                result = self.run_program()
                self.assertEqual(expected, (result['status'], result['errors'][0]['class']))
                self.assertEqual(1, len(self.calls), 'a refusal is not retried')

    def test_maximum_batch_stays_inside_output_budget(self):
        options = {f'label-{i:02d}-' + 'x' * 22: 'd' for i in range(12)}
        questions = [{'id': f'q{i}' + 'y' * 62, 'type': 'choice', 'options': options} for i in range(4)]
        labels = list(options)
        row = {'answers': [{'questionId': q['id'], 'type': 'DECISION_QUESTION_TYPE_CHOICE', 'choice': labels[0], 'confidence': 0.123456,
                            'probabilities': [{'option': label, 'probability': 1 / 12} for label in labels]} for q in questions],
               'provider': 'p' * 200, 'model': 'm' * 200, 'method': 'DECISION_METHOD_DECISION_MODEL', 'locality': 'local', 'latencyMs': '6'}
        self.batch = {'results': [row] * 32, 'usage': {}}
        result = self.run_program({'states': ['x'] * 32, 'questions': questions})
        self.assertEqual('ok', result['status'])
        self.assertLessEqual(len(json.dumps(result).encode()), CONTRACT['budget']['output_bytes'])


if __name__ == '__main__':
    unittest.main()
