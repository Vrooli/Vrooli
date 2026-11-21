#!/usr/bin/env python3
"""Focused regression for the v7 coplanar-cap removal derivative."""

import hashlib
import json
import struct
import unittest
from pathlib import Path

import derive_smart_display_v7 as v7

ROOT = Path(__file__).resolve().parents[3]
ASSET_ROOT = ROOT / "api/internal/physicalruntime/assets"
OLD_HASHES = {
    3: "9890e82e4ded55f098fb2eb99548bb3d0331c1a599193ee2041c9773a338cf50",
    4: "7bf1f3b0c835f5817c592b12693d5b3d1deb6d289e06e727270ce245ebd75333",
    5: "3ee98ae0dd5ce505f0de566055fffe8134fa1cccdad9c4ababab40a5a1ec84c2",
    6: v7.EXPECTED_V6_SHA,
}


def glb_doc_and_binary(path: Path) -> tuple[dict, bytearray]:
    return v7.parse_glb(path.read_bytes())


class SmartDisplayV7Tests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.manifest = v7.build()
        cls.v6_doc, cls.v6_bin = glb_doc_and_binary(ASSET_ROOT / "smart-display-v6/generic-smart-display-v6.glb")
        cls.v7_doc, cls.v7_bin = glb_doc_and_binary(v7.OUT)

    def test_previous_versions_remain_byte_identical(self):
        for version, expected in OLD_HASHES.items():
            path = ASSET_ROOT / f"smart-display-v{version}/generic-smart-display-v{version}.glb"
            with self.subTest(version=version):
                self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), expected)

    def test_only_expected_front_cap_is_removed_and_no_coplanar_shell_cap_remains(self):
        source_mesh = self.v6_doc["meshes"][4]["primitives"][0]
        result_mesh = self.v7_doc["meshes"][4]["primitives"][0]
        source_indices, _ = v7.read_indices(self.v6_doc, self.v6_bin, source_mesh["indices"])
        result_indices, _ = v7.read_indices(self.v7_doc, self.v7_bin, result_mesh["indices"])
        self.assertEqual(len(source_indices) // 3 - len(result_indices) // 3, v7.EXPECTED_FRONT_TRIANGLES)

        positions = v7.read_float_vec3(self.v7_doc, self.v7_bin, result_mesh["attributes"]["POSITION"])
        front = v7.read_float_vec3(self.v7_doc, self.v7_bin, self.v7_doc["meshes"][5]["primitives"][0]["attributes"]["NORMAL"])[0]
        for offset in range(0, len(result_indices), 3):
            tri = result_indices[offset : offset + 3]
            a, b, c = (positions[i] for i in tri)
            ab = tuple(b[k] - a[k] for k in range(3))
            ac = tuple(c[k] - a[k] for k in range(3))
            cross = (ab[1] * ac[2] - ab[2] * ac[1], ab[2] * ac[0] - ab[0] * ac[2], ab[0] * ac[1] - ab[1] * ac[0])
            length = sum(x * x for x in cross) ** 0.5
            if length < 1e-14:
                continue
            dot = sum(cross[k] / length * front[k] for k in range(3))
            plane_distances = [sum(front[k] * positions[i][k] for k in range(3)) for i in tri]
            self.assertFalse(dot > v7.FRONT_DOT_MIN and all(abs(d - v7.PLANE_DISTANCE) <= v7.PLANE_TOLERANCE for d in plane_distances))

    def test_display_plane_and_hierarchy_are_unchanged(self):
        self.assertEqual(self.v7_doc["nodes"], self.v6_doc["nodes"])
        self.assertEqual(self.v7_doc["scenes"], self.v6_doc["scenes"])
        self.assertEqual(self.v7_doc["scene"], self.v6_doc["scene"])
        self.assertEqual(self.v7_doc["meshes"][6], self.v6_doc["meshes"][6])
        self.assertEqual(self.v7_doc["materials"], self.v6_doc["materials"])
        display_accessor_indices = self.v6_doc["meshes"][6]["primitives"][0]["attributes"]
        for semantic, accessor_index in display_accessor_indices.items():
            source_meta, source_data = v7.accessor_bytes(self.v6_doc, self.v6_bin, accessor_index)
            result_accessor_index = self.v7_doc["meshes"][6]["primitives"][0]["attributes"][semantic]
            result_meta, result_data = v7.accessor_bytes(self.v7_doc, self.v7_bin, result_accessor_index)
            self.assertEqual(result_meta, source_meta)
            self.assertEqual(result_data, source_data)

    def test_result_is_deterministic_and_manifest_pins_lineage(self):
        first = v7.OUT.read_bytes()
        manifest = v7.build()
        self.assertEqual(v7.OUT.read_bytes(), first)
        self.assertEqual(manifest["source"]["sha256"], v7.EXPECTED_V6_SHA)
        self.assertEqual(manifest["derivation"]["removed_triangle_count"], v7.EXPECTED_FRONT_TRIANGLES)
        self.assertEqual(manifest["status"], "candidate_only_unqualified")


if __name__ == "__main__":
    unittest.main()
