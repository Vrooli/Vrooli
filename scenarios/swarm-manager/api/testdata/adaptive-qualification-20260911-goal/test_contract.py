import unittest

from allowance import remaining
from evidence import pending


class AllowanceContract(unittest.TestCase):
    def test_reserved_usage_is_not_available(self):
        self.assertEqual(remaining(100, 30, 40), 30)
        self.assertEqual(remaining(100, 60, 50), 0)
        self.assertEqual(remaining(100, 0, 0), 100)


class EvidenceContract(unittest.TestCase):
    def test_absent_required_outcomes_stay_pending(self):
        self.assertEqual(pending(["alpha", "beta", "gamma"], {"alpha": True, "beta": False}), ["beta", "gamma"])
        self.assertEqual(pending(["alpha"], {}), ["alpha"])
        self.assertEqual(pending(["alpha"], {"alpha": True, "extra": False}), [])


if __name__ == "__main__":
    unittest.main()
