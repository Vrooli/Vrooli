"""Start two governed workflows, then collect their bounded evidence."""


request = {
    "owner": "development-toolchain-validator",
    "workflow_key": "development-toolchain-validator/skill-experiment-audit",
    "input": {
        "experiment": {
            "name": "program-runtime-example",
            "objective": "Check that a bounded delegated audit returns structured evidence.",
        },
        "assignments": [{"id": "sample", "token": "delegated runtime smoke"}],
    },
}

first = vrooli.agent.start(**request)
second = vrooli.agent.start(**request)

first_result = vrooli.agent.collect(first, wait_seconds=30)
second_result = vrooli.agent.collect(second, wait_seconds=30)
print(first_result.head(1))
print(second_result.head(1))

# `start` is intentionally non-blocking: both execution identifiers are
# persisted against the session before either result is collected. A collect
# from another session is refused even if the caller knows the identifier.
