#!/usr/bin/env python3
"""
Check Health - Verify Agent S2 is running

This example shows how to check if Agent S2 is running
and get information about its capabilities.
"""

from agent_s2.client import AgentS2Client
import sys

def main():
    print("Agent S2 - Health Check")
    print("======================")
    
    # Create a client
    client = AgentS2Client()
    
    # Check if Agent S2 is healthy
    print("\nChecking Agent S2 health...")
    if client.health_check():
        print("✅ Agent S2 is healthy and running!")
    else:
        print("❌ Agent S2 is not running.")
        print("   Start it with: ./manage.sh --action start")
        sys.exit(1)
    
    # Get detailed health information
    import requests
    response = requests.get(f"{client.base_url}/health")
    if response.ok:
        health_data = response.json()
        
        print("\nHealth Details:")
        print(f"  Status: {health_data['status']}")
        print(f"  Display: {health_data['display']}")
        print(f"  Screen Size: {health_data['screen_size']['width']}x{health_data['screen_size']['height']}")
        print(f"  Tasks Processed: {health_data['tasks_processed']}")
        
        ai_status = health_data['ai_status']
        print(f"\nAI Status:")
        print(f"  Available: {ai_status['available']}")
        print(f"  Enabled: {ai_status['enabled']}")
        print(f"  Initialized: {ai_status['initialized']}")
        if ai_status['initialized']:
            print(f"  Provider: {ai_status['provider']}")
            print(f"  Model: {ai_status['model']}")
    
    # Get capabilities
    response = requests.get(f"{client.base_url}/capabilities")
    if response.ok:
        capabilities = response.json()
        
        print(f"\nCapabilities:")
        for cap, enabled in capabilities['capabilities'].items():
            status = "✅" if enabled else "❌"
            print(f"  {status} {cap}")
        
        print(f"\nSupported Tasks: {len(capabilities['supported_tasks'])}")
        for task in capabilities['supported_tasks'][:5]:
            print(f"  - {task}")
        if len(capabilities['supported_tasks']) > 5:
            print(f"  ... and {len(capabilities['supported_tasks']) - 5} more")

if __name__ == "__main__":
    main()