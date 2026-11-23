"""Contracts of the private review page builder (synthetic flat-colour images only)."""

import json
import os
import re
import sys
import tempfile
import unittest

from PIL import Image

sys.path.insert(0, os.path.dirname(__file__))
import build_nsfw_review as b  # noqa: E402


def make_dir(items, sizes=None):
    root = tempfile.mkdtemp()
    for i, item in enumerate(items):
        if item.get("file"):
            os.makedirs(os.path.dirname(os.path.join(root, item["file"])), exist_ok=True)
            Image.new("RGB", (sizes or [(1600, 1200)])[i % len(sizes or [1])], (40 + i * 20, 90, 120)).save(os.path.join(root, item["file"]))
    with open(os.path.join(root, "manifest.json"), "w") as fh:
        json.dump(items, fh)
    return root


def item(ident, review="passed", **kw):
    base = {"id": ident, "kind": "gen", "file": f"gen/{ident.lower()}.png", "model": "m1", "prompt": "a <fictional> adult & scene",
            "seed": 7, "seconds": 12.34, "job_id": "job-" + ident, "status": "succeeded", "nsfw_score": 0.973, "review": review, "faces": 1}
    base.update(kw)
    return base


class BuildTests(unittest.TestCase):
    def test_page_is_self_contained_blurred_and_javascript_free(self):
        root = make_dir([item("G1"), item("E1", kind="edit", file="edit/e1.png", source_id="B1")])
        page = b.build_html(b._load_manifest(root), root)
        self.assertNotIn("<script", page)
        self.assertNotRegex(page, r'(?:src|href)="https?://')
        self.assertEqual(len(re.findall(r'src="data:image/jpeg;base64,', page)), 2)
        self.assertIn("filter:blur(", page)
        self.assertIn("input.rv:checked + label.shot img{filter:none", page)
        # one checkbox per card drives its own reveal
        self.assertEqual(len(re.findall(r'<input class="rv" type="checkbox" id="r-', page)), 2)
        self.assertIn('name="viewport"', page)

    def test_every_result_carries_id_model_prompt_seed_time_and_job(self):
        root = make_dir([item("G1")])
        page = b.build_html(b._load_manifest(root), root)
        for needle in ("G1", "m1", "seed</b> 7", "12.3 s", "job-G1"):
            self.assertIn(needle, page)

    def test_prompt_is_escaped(self):
        root = make_dir([item("G1")])
        page = b.build_html(b._load_manifest(root), root)
        self.assertIn("a &lt;fictional&gt; adult &amp; scene", page)
        self.assertNotIn("<fictional>", page)

    def test_only_reviewer_passed_images_are_embedded(self):
        root = make_dir([item("G1"), item("G2", review="refused"), item("G3", review="unavailable", file=None)])
        page = b.build_html(b._load_manifest(root), root)
        self.assertEqual(len(re.findall(r"data:image/", page)), 1)
        self.assertIn("Not embedded", page)
        self.assertIn("G2 (refused)", page)
        # a refused item's file need not even exist
        os.remove(os.path.join(root, "gen/g2.png"))
        b.build_html(b._load_manifest(root), root)

    def test_large_images_shrink_to_the_phone_budget_and_zero_keeps_original_bytes(self):
        root = make_dir([item("G1")])
        small = b.embed_image(os.path.join(root, "gen/g1.png"), 512)
        self.assertEqual((small["width"], small["height"]), (1600, 1200))  # reported size is the original
        self.assertLess(len(small["data_uri"]), small["bytes"] * 2)
        original = b.embed_image(os.path.join(root, "gen/g1.png"), 0)
        self.assertTrue(original["data_uri"].startswith("data:image/png;base64,"))

    def test_bad_manifest_and_corrupt_image_fail_loudly(self):
        root = tempfile.mkdtemp()
        with self.assertRaises(b.ReviewBuildError):
            b._load_manifest(root)
        open(os.path.join(root, "manifest.json"), "w").write("[]")
        with self.assertRaises(b.ReviewBuildError):
            b._load_manifest(root)
        open(os.path.join(root, "manifest.json"), "w").write(json.dumps([{"id": "G1"}, {"id": "G1"}]))
        with self.assertRaises(b.ReviewBuildError):
            b._load_manifest(root)
        root2 = make_dir([item("G1")])
        open(os.path.join(root2, "gen/g1.png"), "wb").write(b"not an image")
        with self.assertRaises(b.ReviewBuildError):
            b.build_html(b._load_manifest(root2), root2)

    def test_main_writes_review_html_next_to_the_manifest(self):
        root = make_dir([item("G1")])
        self.assertEqual(b.main([root]), 0)
        self.assertTrue(os.path.exists(os.path.join(root, "REVIEW.html")))


if __name__ == "__main__":
    unittest.main()
