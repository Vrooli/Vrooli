"""Read several governed scenario surfaces and keep only bounded summaries."""

def main():
    calls = {
        "agent_manager_runs": lambda: vrooli.agent_manager.measures.run_volume(),
        "ai_gateway_calls": lambda: vrooli.ai_gateway.measures.total(),
        "program_runtime_bindings": lambda: vrooli.program_runtime.bindings.list(),
    }
    results = vrooli.gather(*calls.values())
    print({
        name: {
            "count": result.count(),
            "first_row_keys": sorted(result.head(1)[0]) if result.count() else [],
        }
        for name, result in zip(calls, results)
    })


main()

# Live output (2026-08-12):
# {'agent_manager_runs': {'count': 1, 'first_row_keys': ['definitionId',
#   'historyFloor', 'outsideHistoryRunCount', 'provenance', 'terminalRuns',
#   'totalRuns', 'validity']}, 'ai_gateway_calls': {'count': 1,
#   'first_row_keys': ['count']}, 'program_runtime_bindings': {'count': 1167,
#   'first_row_keys': ['command', 'description', 'effect', 'group', 'id',
#   'method', 'permissions', 'requestType', 'responseType', 'runEligible',
#   'scenario', 'service', 'signature']}}
