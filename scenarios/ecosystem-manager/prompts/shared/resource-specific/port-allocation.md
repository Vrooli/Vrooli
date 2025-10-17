# Port Allocation

For resources, scripts/resources/port_registry.sh should be the single source-of-truth for ports. Avoid hard-coding ports everywhere else.

For scenarios, the overwhelming majority will have API_PORT and UI_PORT defined as ranges (where all scenarios use the same, large range for APIs and a different, large range for all UIs) in their service.json. Only scenarios which need to be accessed directy through a secure tunnel (very rare, as most scenarios will be accessed via iframes from scenario launchers/dashboards like app-monitor) should have fixed ports. As long as scenarios are started through the proper `vrooli` CLI commands, the lifecycle logic will make sure they are available. Scenarios, like resources, should NEVER hard-code ports, even if only as a fallback (since ports are dynamically allocated, fallback port numbers don't make any sense).
