#!/usr/bin/env python3
"""Test script to verify agent-s2 stability fixes"""

import requests
import json
import time
import base64
from typing import Dict, Any, Optional

BASE_URL = "http://localhost:4113"


def test_screenshot_validation():
    """Test screenshot format validation improvements"""
    print("\n=== Testing Screenshot Format Validation ===")
    
    # Test 1: Invalid response format (should fail)
    print("\n1. Testing invalid response format...")
    try:
        response = requests.post(f"{BASE_URL}/screenshot?response_format=url")
        assert response.status_code == 422, f"Expected 422, got {response.status_code}"
        print("✅ Invalid format properly rejected")
    except Exception as e:
        print(f"❌ Test failed: {e}")
    
    # Test 2: Valid formats
    print("\n2. Testing valid formats...")
    for fmt in ["json", "binary"]:
        try:
            response = requests.post(f"{BASE_URL}/screenshot?response_format={fmt}")
            if response.status_code == 200:
                print(f"✅ Format '{fmt}' accepted")
            else:
                print(f"❌ Format '{fmt}' failed: {response.status_code}")
        except Exception as e:
            print(f"❌ Error testing format '{fmt}': {e}")
    
    # Test 3: Large screenshot handling
    print("\n3. Testing large screenshot handling...")
    try:
        # Request with size limit
        response = requests.post(
            f"{BASE_URL}/screenshot",
            params={"max_size_mb": 0.1}  # Very small limit to trigger size check
        )
        if response.status_code == 400 and "too large" in response.text:
            print("✅ Large screenshot properly handled")
        else:
            print(f"❌ Large screenshot test unexpected response: {response.status_code}")
    except Exception as e:
        print(f"❌ Large screenshot test failed: {e}")
    
    # Test 4: Invalid region
    print("\n4. Testing invalid region validation...")
    try:
        response = requests.post(
            f"{BASE_URL}/screenshot",
            json={"region": [-10, -10, 100, 100]}  # Negative coordinates
        )
        assert response.status_code == 400, f"Expected 400, got {response.status_code}"
        print("✅ Invalid region properly rejected")
    except Exception as e:
        print(f"❌ Invalid region test failed: {e}")


def test_browser_health():
    """Test browser health monitoring"""
    print("\n=== Testing Browser Health Monitoring ===")
    
    try:
        response = requests.get(f"{BASE_URL}/health")
        if response.status_code == 200:
            data = response.json()
            
            # Check if browser status is included
            if "browser_status" in data:
                browser = data["browser_status"]
                print(f"✅ Browser health monitoring active")
                print(f"   - Browser: {browser.get('browser', 'Unknown')}")
                print(f"   - Running: {browser.get('health', {}).get('running', False)}")
                print(f"   - Memory: {browser.get('health', {}).get('total_memory_mb', 0)}MB")
                print(f"   - Processes: {browser.get('health', {}).get('process_count', 0)}")
                print(f"   - Restart count: {browser.get('statistics', {}).get('restart_count', 0)}")
                print(f"   - Total crashes: {browser.get('statistics', {}).get('total_crashes', 0)}")
            else:
                print("⚠️  Browser health not included in response (may not be implemented yet)")
                
            # Check overall health
            if data.get("status") == "healthy":
                print("✅ Overall system healthy")
            elif data.get("status") == "degraded":
                print("⚠️  System degraded - check browser health")
            else:
                print(f"❌ Unexpected status: {data.get('status')}")
                
        else:
            print(f"❌ Health check failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Health check error: {e}")


def test_firefox_stability():
    """Test Firefox stability under load"""
    print("\n=== Testing Firefox Stability ===")
    
    print("\n1. Taking multiple screenshots to test memory handling...")
    success_count = 0
    for i in range(5):
        try:
            response = requests.post(
                f"{BASE_URL}/screenshot",
                params={"format": "jpeg", "quality": 80}
            )
            if response.status_code == 200:
                data = response.json()
                size_mb = len(data.get("data", "")) / (1024 * 1024)
                print(f"   Screenshot {i+1}: ✅ ({size_mb:.2f}MB)")
                success_count += 1
            else:
                print(f"   Screenshot {i+1}: ❌ ({response.status_code})")
        except Exception as e:
            print(f"   Screenshot {i+1}: ❌ ({e})")
        time.sleep(1)
    
    print(f"\nSuccess rate: {success_count}/5")
    
    # Test 2: AI task to stress Firefox
    print("\n2. Testing AI browser task...")
    try:
        response = requests.post(
            f"{BASE_URL}/ai/action",
            json={"task": "open a new tab in firefox"}
        )
        if response.status_code == 200:
            result = response.json()
            if result.get("success"):
                print("✅ AI browser task completed")
            else:
                print(f"❌ AI task failed: {result.get('error')}")
        else:
            print(f"❌ AI task request failed: {response.status_code}")
    except Exception as e:
        print(f"❌ AI task error: {e}")
    
    # Check health after stress
    print("\n3. Checking system health after stress test...")
    test_browser_health()


def test_screenshot_edge_cases():
    """Test screenshot edge cases"""
    print("\n=== Testing Screenshot Edge Cases ===")
    
    # Test 1: Empty region
    print("\n1. Testing empty region...")
    try:
        response = requests.post(
            f"{BASE_URL}/screenshot",
            json={"region": [0, 0, 0, 0]}
        )
        assert response.status_code == 400, f"Expected 400, got {response.status_code}"
        print("✅ Empty region properly rejected")
    except Exception as e:
        print(f"❌ Empty region test failed: {e}")
    
    # Test 2: Different formats
    print("\n2. Testing different image formats...")
    for fmt in ["png", "jpeg", "jpg"]:
        try:
            response = requests.post(
                f"{BASE_URL}/screenshot",
                params={"format": fmt}
            )
            if response.status_code == 200:
                data = response.json()
                if data.get("format") in ["png", "jpeg"]:
                    print(f"✅ Format '{fmt}' works correctly")
                else:
                    print(f"❌ Format '{fmt}' returned unexpected format: {data.get('format')}")
            else:
                print(f"❌ Format '{fmt}' failed: {response.status_code}")
        except Exception as e:
            print(f"❌ Format '{fmt}' error: {e}")
    
    # Test 3: Binary response
    print("\n3. Testing binary response...")
    try:
        response = requests.post(
            f"{BASE_URL}/screenshot",
            params={"response_format": "binary"}
        )
        if response.status_code == 200:
            content_type = response.headers.get("content-type", "")
            if "image/" in content_type:
                print(f"✅ Binary response works (size: {len(response.content)} bytes)")
            else:
                print(f"❌ Unexpected content type: {content_type}")
        else:
            print(f"❌ Binary response failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Binary response error: {e}")


def main():
    """Run all tests"""
    print("=" * 60)
    print("Agent-S2 Stability Test Suite")
    print("=" * 60)
    
    # Check if service is running
    try:
        response = requests.get(f"{BASE_URL}/health", timeout=5)
        if response.status_code != 200:
            print("❌ Agent-S2 service not responding correctly")
            return
    except Exception as e:
        print(f"❌ Cannot connect to Agent-S2 service: {e}")
        print("\nMake sure Agent-S2 is running on port 4113")
        return
    
    # Run tests
    test_screenshot_validation()
    test_browser_health()
    test_screenshot_edge_cases()
    test_firefox_stability()
    
    print("\n" + "=" * 60)
    print("Test suite completed")
    print("=" * 60)


if __name__ == "__main__":
    main()