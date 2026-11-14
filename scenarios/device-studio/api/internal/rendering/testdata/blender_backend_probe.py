import hashlib
import json
import os
import sys
import time

sys.dont_write_bytecode = True
import bpy


PROFILE_ID = "blender-5.2.2-linux-x64"
VERSION = "Blender 5.2.2 LTS"
TREE_SHA256 = "5fca420feab408276778511a84d7a2f5659bbc09aa5cb43b6d24ae59b220cb55"
BINARY_SHA256 = "bafec183e960819c76fe213c000cdb2284c53a87ef6791812f8728b51f07145b"
ARTIFACT = "a1660c40fcc34e79aaa339f846b95d5973981a23b9e4b87e50ee2c633159762e"
OUTPUT = "qualification.png"
RECEIPT = "qualification-receipt.json"
BACKENDS = {"CPU", "CUDA", "OPTIX"}


def sha256_file(path):
    digest = hashlib.sha256()
    with open(path, "rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def fail(message):
    raise RuntimeError(message)


if "--" not in sys.argv:
    fail("fixed qualification manifest argument is missing")
args = sys.argv[sys.argv.index("--") + 1 :]
if len(args) != 1 or args[0] != "qualification-manifest.json":
    fail("qualification accepts only its fixed manifest name")
manifest_path = os.path.join(os.getcwd(), args[0])
with open(manifest_path, "r", encoding="utf-8") as source:
    manifest_bytes = source.read().encode("utf-8")
manifest = json.loads(manifest_bytes)
expected_keys = {
    "schema_version", "profile_id", "artifact_identity", "tree_sha256",
    "binary_sha256", "blender_version", "adapter_sha256", "backend",
    "width", "height", "samples",
}
if set(manifest) != expected_keys:
    fail("qualification manifest fields do not match the fixed schema")
if manifest["schema_version"] != 1 or manifest["profile_id"] != PROFILE_ID:
    fail("qualification manifest profile is unsupported")
if manifest["tree_sha256"] != TREE_SHA256 or manifest["binary_sha256"] != BINARY_SHA256 or manifest["artifact_identity"] != ARTIFACT or manifest["blender_version"] != VERSION:
    fail("qualification manifest is not bound to the pinned Blender build")
if manifest["backend"] not in BACKENDS or (manifest["width"], manifest["height"]) != (256, 256):
    fail("qualification backend or raster is outside the fixed profile")
if manifest["samples"] != 4 or tuple(bpy.app.version) != (5, 2, 2):
    fail("Blender version or render settings differ from the fixed profile")
script_sha = sha256_file(__file__)
if script_sha != manifest["adapter_sha256"]:
    fail("qualification adapter bytes changed")
if os.path.lexists(OUTPUT) or os.path.lexists(RECEIPT):
    fail("qualification output already exists")

started = time.monotonic()
backend = manifest["backend"]
device = "CPU"
status = "rendered"
probe_error = ""
scene = bpy.context.scene
scene.render.engine = "CYCLES"
scene.cycles.samples = 4
scene.cycles.use_denoising = False
scene.render.resolution_x = 256
scene.render.resolution_y = 256
scene.render.resolution_percentage = 100
scene.render.image_settings.file_format = "PNG"
scene.render.filepath = os.path.join(os.getcwd(), OUTPUT)
if backend == "CPU":
    scene.cycles.device = "CPU"
else:
    preferences = bpy.context.preferences.addons["cycles"].preferences
    try:
        preferences.compute_device_type = backend
        preferences.get_devices()
        selected = [item for item in preferences.devices if item.type == backend]
    except Exception as error:
        status = "backend_probe_failed"
        probe_error = str(error)[:1024]
        selected = []
    if status == "backend_probe_failed":
        pass
    elif not selected:
        status = "backend_unavailable"
    else:
        for item in preferences.devices:
            item.use = item in selected
        scene.cycles.device = "GPU"
        device = ",".join(sorted(item.name for item in selected))[:256]

output_sha = ""
output_size = 0
if status == "rendered":
    bpy.ops.render.render(write_still=True)
    output_size = os.path.getsize(OUTPUT)
    if output_size <= 0 or output_size > 8 * 1024 * 1024:
        fail("qualification PNG exceeded the fixed output bound")
    output_sha = sha256_file(OUTPUT)

receipt = {
    "schema_version": 1,
    "status": status,
    "manifest_sha256": hashlib.sha256(manifest_bytes).hexdigest(),
    "adapter_sha256": script_sha,
    "profile_id": PROFILE_ID,
    "artifact_identity": manifest["artifact_identity"],
    "tree_sha256": TREE_SHA256,
    "binary_sha256": manifest["binary_sha256"],
    "blender_version": bpy.app.version_string,
    "blender_app_version": list(bpy.app.version),
    "blender_build_hash": bpy.app.build_hash.hex(),
    "backend_requested": backend,
    "backend_selected": backend if status == "rendered" else "",
    "device": device if status == "rendered" else "",
    "width": 256,
    "height": 256,
    "samples": 4,
    "output_name": OUTPUT if status == "rendered" else "",
    "output_sha256": output_sha,
    "output_size_bytes": output_size,
    "error": probe_error,
    "elapsed_ms": int((time.monotonic() - started) * 1000),
}
with open(RECEIPT, "x", encoding="utf-8") as destination:
    json.dump(receipt, destination, sort_keys=True, separators=(",", ":"))
    destination.write("\n")
