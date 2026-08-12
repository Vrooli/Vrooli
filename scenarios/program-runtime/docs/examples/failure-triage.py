"""Find recurring program failure shapes without materializing the corpus."""


failures = vrooli.program_runtime.programs.mine(include_operator=False)
print({"top_failure_shapes": failures.group_by("shape")})

# Live output (2026-08-12):
# {'top_failure_shapes': {'modulenotfounderror': 1, 'runtimeerror': 1}}
