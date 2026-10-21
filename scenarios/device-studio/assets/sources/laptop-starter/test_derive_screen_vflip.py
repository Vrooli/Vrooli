import hashlib
import importlib.util
import pathlib
import struct
import unittest

SCRIPT = pathlib.Path(__file__).with_name("derive_screen_vflip.py")
spec = importlib.util.spec_from_file_location("derive_screen_vflip", SCRIPT)
derive = importlib.util.module_from_spec(spec)
spec.loader.exec_module(derive)


class VerticalOnlyScreenViewTests(unittest.TestCase):
    def test_vertical_flip_preserves_source_u_and_every_non_screen_payload(self):
        source = derive.SOURCE.read_bytes()
        output, source_bin = derive.derive(source)
        json_type, source_json = derive.chunks(source)[0]
        _, output_json = derive.chunks(output)[0]
        _, output_bin = derive.chunks(output)[1]
        before, after = __import__("json").loads(source_json), __import__("json").loads(output_json)
        self.assertEqual(output_bin[:len(source_bin)], source_bin)
        screen = next(node for node in after["nodes"] if node.get("name") == derive.SCREEN_NODE)
        old_screen = next(node for node in before["nodes"] if node.get("name") == derive.SCREEN_NODE)
        primitive = after["meshes"][screen["mesh"]]["primitives"][0]
        old_primitive = before["meshes"][old_screen["mesh"]]["primitives"][0]
        self.assertEqual(primitive["attributes"]["POSITION"], old_primitive["attributes"]["POSITION"])
        self.assertEqual(primitive["attributes"]["NORMAL"], old_primitive["attributes"]["NORMAL"])
        self.assertEqual(primitive["indices"], old_primitive["indices"])
        old_uv = before["accessors"][old_primitive["attributes"]["TEXCOORD_0"]]
        new_uv = after["accessors"][primitive["attributes"]["TEXCOORD_0"]]
        old_view, new_view = before["bufferViews"][old_uv["bufferView"]], after["bufferViews"][new_uv["bufferView"]]
        old_values = struct.unpack_from("<" + "f" * old_uv["count"] * 2, source_bin, old_view.get("byteOffset", 0))
        new_values = struct.unpack_from("<" + "f" * new_uv["count"] * 2, output_bin, new_view.get("byteOffset", 0))
        self.assertEqual(new_values[::2], old_values[::2])
        self.assertEqual(new_values[1::2], tuple(1.0 - value for value in old_values[1::2]))
        self.assertEqual(after["nodes"], before["nodes"])
        self.assertEqual(hashlib.sha256(output).hexdigest(), "1d833e3d08d0722952ecd9b6616cf933af11fca5716cfe4bc67aa110850272bd")


if __name__ == "__main__":
    unittest.main()
