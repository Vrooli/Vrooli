import json
import unittest
from pathlib import Path


class Handle:
    def __init__(self, rows, token):
        self.rows = rows
        self.token = token

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return {'nextPageToken': self.token}


class Runner:
    def inputs(self):
        return {'owner': 'fixture-owner'}

    def envelope(self, *_args, **_kwargs):
        return {'status': 'running', 'phase': 'validate', 'inputs': {}, 'signals': {}, 'evidence': [], 'errors': []}

    def guarded(self, fn):
        return fn

    def fail(self, *args):
        raise AssertionError(args)

    def classify(self, exc):
        return 'failed', 'fixture_error'

    def run(self, steps, step):
        while step:
            step = steps[step]()


class Binding:
    def __init__(self, name, total):
        self.name, self.total, self.calls = name, total, []

    def __call__(self, **kwargs):
        self.calls.append(kwargs)
        offset = (len(self.calls) - 1) * 50
        rows = [{'id': f'{self.name}-{i}', 'owner': 'fixture-owner', 'name': f'{self.name}-{i}'}
                for i in range(offset, min(offset + 50, self.total))]
        token = f'{self.name}-cursor-{len(self.calls)}' if offset + 50 < self.total else ''
        return Handle(rows, token)


class Namespace:
    pass


class CatalogReadPagingTests(unittest.TestCase):
    def test_follows_four_pages_then_keeps_continuation(self):
        targets, plans = Binding('target', 260), Binding('plan', 51)
        owner = Namespace()
        owner.targets, owner.plans = Namespace(), Namespace()
        owner.targets.list, owner.plans.list = targets, plans
        captured = []

        def gather(*calls):
            return [call() for call in calls]

        def emit(value):
            captured.append(value)

        env = {'program': Runner(), 'data_backup_manager': owner, 'gather': gather, 'print': emit}
        source = Path(__file__).parents[1] / 'catalog-read.py'
        exec(compile(source.read_text(), str(source), 'exec'), env)
        result = json.loads(captured[0])
        self.assertEqual(len(result['signals']['targets']), 200)
        self.assertEqual(len(result['signals']['plans']), 51)
        self.assertEqual(result['status'], 'partial')
        self.assertTrue(result['signals']['targets_next_page_token'])
        self.assertEqual(result['signals']['plans_next_page_token'], '')
        self.assertEqual(len(targets.calls), 4)
        self.assertEqual(len(plans.calls), 2)
        self.assertEqual(targets.calls[0], {'page-size': 50, 'page-token': '', 'owner': 'fixture-owner'})


if __name__ == '__main__':
    unittest.main()
