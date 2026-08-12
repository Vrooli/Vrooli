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

# Live validation (program `prog_283582d6-bdb7-4623-ab6e-85f4b79288c1`) returned
# a succeeded workflow with execution `ee87b3cb-7be6-4712-ac1a-0d209a59f017`,
# one challenged evidence finding, and zero gaming findings. This workflow is
# intentionally bounded and terminal; it does not mutate files or wait for an
# operator approval signal.
