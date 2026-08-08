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

# Live validation (program `prog_f0dc97b1-3a76-4903-9cd9-f89c4ec50096`) returned
# a succeeded workflow with execution `887dac2a-ee8e-4482-981f-d2f1fb7dfe75`,
# one challenged evidence finding, and zero gaming findings. This workflow is
# intentionally bounded and terminal; it does not mutate files or wait for an
# operator approval signal.
