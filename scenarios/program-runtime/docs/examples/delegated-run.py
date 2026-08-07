"""Start a governed Agent Manager workflow and retain its bounded evidence handle."""


result = vrooli.agent.run(
    owner="development-toolchain-validator",
    workflow_key="development-toolchain-validator/skill-experiment-audit",
    input={
        "experiment": {
            "name": "program-runtime-example",
            "objective": "Check that a bounded delegated audit returns structured evidence.",
        },
        "assignments": [{"id": "sample", "token": "delegated runtime smoke"}],
    },
)
print(result.head(1))

# Live validation (program `prog_d693e30f-fcfd-4e1d-b4fc-2207707b1616`) returned
# a succeeded workflow with execution
# `3c0f9202-8f38-4178-ac4b-45f640c8fdbe`, one challenged evidence finding, and
# zero gaming findings. This workflow is intentionally bounded and terminal;
# it does not mutate files or wait for an operator approval signal.
