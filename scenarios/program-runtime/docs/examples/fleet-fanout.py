"""Read several governed scenario surfaces and keep only bounded summaries."""

def main():
    calls = {
        "agent_manager_runs": lambda: vrooli.agent_manager.measures.run_volume(),
        "ai_gateway_calls": lambda: vrooli.ai_gateway.measures.total(),
        "program_runtime_bindings": lambda: vrooli.program_runtime.bindings.list(),
    }
    results = vrooli.gather(*calls.values())
    print({name: result.head(1) for name, result in zip(calls, results)})


main()

# Live output (2026-08-07):
# {'agent_manager_runs': [{'definitionId': 'throughput.run_volume',
#   'terminalRuns': '20', 'totalRuns': '24', ...}],
#  'ai_gateway_calls': [{'count': '18289'}],
#  'program_runtime_bindings': [{'command': 'plan',
#   'id': 'agent-manager/declarations/plan', ...}]}
