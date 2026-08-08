"""Run independent read bindings concurrently and report bounded counts."""

import time


started = time.perf_counter()
results = vrooli.gather(
    lambda: vrooli.agent_manager.measures.run_volume(),
    lambda: vrooli.ai_gateway.measures.total(),
    lambda: vrooli.program_runtime.programs.mine(include_operator=False),
)
print(
    {
        "elapsed_seconds": round(time.perf_counter() - started, 3),
        "result_counts": [result.count() for result in results],
    }
)

# Live output (2026-08-07):
# {'elapsed_seconds': 0.147, 'result_counts': [1, 1, 1]}
