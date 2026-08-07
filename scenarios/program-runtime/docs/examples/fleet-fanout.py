"""Read several governed scenario surfaces and keep only bounded summaries."""

import asyncio


async def main():
    calls = {
        "agent_manager_runs": vrooli.agent_manager.measures.run_volume(),
        "ai_gateway_calls": vrooli.ai_gateway.measures.total(),
        "program_runtime_bindings": vrooli.program_runtime.bindings.list(),
    }
    results = await asyncio.gather(*calls.values())
    print({name: result.head(1) for name, result in zip(calls, results)})


await main()

# Live output (2026-08-07):
# {'agent_manager_runs': [{'definitionId': 'throughput.run_volume',
#   'terminalRuns': '11', 'totalRuns': '11', ...}],
#  'ai_gateway_calls': [{'count': '15437'}],
#  'program_runtime_bindings': [{'command': 'plan',
#   'id': 'agent-manager/declarations/plan', ...}]}
