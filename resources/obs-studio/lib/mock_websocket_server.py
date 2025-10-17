#!/usr/bin/env python3
"""
OBS Studio Mock WebSocket Server
Provides a minimal WebSocket server that responds to OBS WebSocket protocol
"""

import asyncio
import json
import websockets
import sys
import os
import signal

PORT = int(os.environ.get('OBS_PORT', 4455))

async def handle_websocket(websocket):
    """Handle WebSocket connections"""
    try:
        # Send Hello message (OBS WebSocket protocol)
        hello_msg = {
            "op": 0,  # Hello
            "d": {
                "obsWebSocketVersion": "5.0.0",
                "rpcVersion": 1
            }
        }
        await websocket.send(json.dumps(hello_msg))
        
        async for message in websocket:
            try:
                data = json.loads(message)
                op = data.get("op", 0)
                
                # Handle Identify message
                if op == 1:  # Identify
                    response = {
                        "op": 2,  # Identified
                        "d": {
                            "negotiatedRpcVersion": 1
                        }
                    }
                    await websocket.send(json.dumps(response))
                
                # Handle Request message
                elif op == 6:  # Request
                    request_type = data.get("d", {}).get("requestType")
                    request_id = data.get("d", {}).get("requestId")
                    
                    # Mock responses for common requests
                    response_data = {}
                    if request_type == "GetVersion":
                        response_data = {
                            "obsVersion": "29.0.0",
                            "obsWebSocketVersion": "5.0.0",
                            "rpcVersion": 1
                        }
                    elif request_type == "GetSceneList":
                        response_data = {
                            "currentProgramSceneName": "Main",
                            "scenes": [
                                {"sceneName": "Main", "sceneIndex": 0},
                                {"sceneName": "Screen Share", "sceneIndex": 1}
                            ]
                        }
                    elif request_type == "GetRecordStatus":
                        response_data = {
                            "outputActive": False,
                            "outputPaused": False,
                            "outputTimecode": "00:00:00.000",
                            "outputDuration": 0,
                            "outputBytes": 0
                        }
                    
                    response = {
                        "op": 7,  # RequestResponse
                        "d": {
                            "requestType": request_type,
                            "requestId": request_id,
                            "requestStatus": {
                                "result": True,
                                "code": 100
                            },
                            "responseData": response_data
                        }
                    }
                    await websocket.send(json.dumps(response))
                    
            except json.JSONDecodeError:
                pass
                
    except websockets.exceptions.ConnectionClosed:
        pass
    except Exception as e:
        print(f"Error handling WebSocket: {e}", file=sys.stderr)

async def main():
    """Start the WebSocket server"""
    print(f"Starting OBS Mock WebSocket Server on port {PORT}")
    
    async with websockets.serve(handle_websocket, "localhost", PORT):
        # Keep server running
        await asyncio.Future()

def signal_handler(sig, frame):
    """Handle shutdown signals"""
    print("\nShutting down OBS Mock WebSocket Server")
    sys.exit(0)

if __name__ == "__main__":
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nServer stopped")