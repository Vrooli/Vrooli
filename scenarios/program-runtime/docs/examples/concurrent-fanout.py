"""Run independent read bindings concurrently and report bounded counts."""

import asyncio
import time


started = time.perf_counter()
results = await asyncio.gather(
    vrooli.agent_manager.measures.run_volume(),
    vrooli.ai_gateway.measures.total(),
    vrooli.program_runtime.programs.mine(include_operator=False),
)
print(
    {
        "elapsed_seconds": round(time.perf_counter() - started, 3),
        "result_counts": [result.count() for result in results],
    }
)

# Live output (2026-08-07):
# {'elapsed_seconds': 0.111, 'result_counts': [1, 1, 1]}
