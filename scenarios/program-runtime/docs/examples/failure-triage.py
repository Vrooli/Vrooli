"""Find recurring program failure shapes without materializing the corpus."""


failures = vrooli.program_runtime.programs.mine(include_operator=False)
print({"top_failure_shapes": failures.group_by("failureShape")})

# Live output (2026-08-07):
# {'top_failure_shapes': {None: 1}}
