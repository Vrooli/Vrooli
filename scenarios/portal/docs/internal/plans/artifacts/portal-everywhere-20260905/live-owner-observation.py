"""Observation-only live owner IPC acceptance; pixels are hashed then discarded."""
import base64
import hashlib
import http.client
import json
import socket
import struct
import sys
from pathlib import Path

bootstrap_path = Path(sys.argv[1])
config = json.loads(bootstrap_path.read_text())
class UnixHTTP(http.client.HTTPConnection):
    def connect(self):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.settimeout(15)
        self.sock.connect(str(bootstrap_path.parent / "desktop-owner.sock"))

def rpc(method, payload):
    conn = UnixHTTP("localhost", timeout=15)
    try:
        conn.request("POST", "/vrooli.device_control.v1.desktop.DesktopOwnerService/" + method,
                     json.dumps(payload), {"Content-Type": "application/json", "Connect-Protocol-Version": "1"})
        response = conn.getresponse()
        data = response.read(48 * 1024 * 1024)
        if response.status != 200:
            raise RuntimeError(f"{method}: HTTP {response.status}: {data[:512]!r}")
        return json.loads(data)
    finally:
        conn.close()

opened = rpc("Open", {"surface": config["helper"]["surface"], "ttlSeconds": 30, "control": False})
try:
    result = rpc("Observe", {"session": opened["session"]})
    pixels = base64.b64decode(result["png"], validate=True)
    assert pixels[:8] == b"\x89PNG\r\n\x1a\n"
    width, height = struct.unpack(">II", pixels[16:24])
    assert width == result["width"] and height == result["height"]
    print(json.dumps({"session": opened["session"], "control": opened.get("control", False),
                      "width": width, "height": height, "png_bytes": len(pixels),
                      "png_sha256": hashlib.sha256(pixels).hexdigest(),
                      "captured_at": result["capturedAt"], "display_id": result["displayId"],
                      "geometry_revision": result["geometryRevision"]}, indent=2))
finally:
    rpc("Stop", {"session": opened["session"]})
