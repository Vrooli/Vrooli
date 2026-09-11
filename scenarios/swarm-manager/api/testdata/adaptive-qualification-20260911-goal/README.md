# Disposable unattended goal-session fixture

This fixture is for the unattended self-improvement plan only. It contains two
independently broken implementation modules and one protected contract module.
Qualification workers may repair `allowance.py` and `evidence.py` in separate
plan phases. They must not change `test_contract.py`.

The live goal-session qualification uses the Agent Manager global work-time
ceiling, currently 7,200 seconds; the disposable item may use a smaller grant
for a bounded attempt, but the declared workflow must not exceed that ceiling.

The fixture is disposable. Do not use it as product code or as a dependency of
the Swarm Manager service.
