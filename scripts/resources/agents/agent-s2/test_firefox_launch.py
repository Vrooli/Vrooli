#\!/usr/bin/env python3
import asyncio
import sys
sys.path.insert(0, '/home/matthalloran8/Vrooli/scripts/resources/agents/agent-s2')

from agent_s2.environment.discovery import EnvironmentDiscovery
from agent_s2.environment.context import ModeContext

# Test application discovery
discovery = EnvironmentDiscovery()
context = ModeContext(discovery=discovery)

print("Testing application discovery...")
print("\nAvailable applications:")
apps = context.context_data.get("applications", {})
for app_name, app_info in apps.items():
    print(f"- {app_name}: {app_info.get('name', 'Unknown')} ({app_info.get('command', 'no command')})")

print("\nTesting Firefox lookup:")
firefox_info = context.get_application_info("Firefox ESR")
if firefox_info:
    print(f"Found Firefox ESR: {firefox_info}")
else:
    print("Firefox ESR not found in application list")
    
# Try alternative names
for name in ["firefox", "Firefox", "firefox-esr"]:
    info = context.get_application_info(name)
    if info:
        print(f"Found {name}: {info}")
