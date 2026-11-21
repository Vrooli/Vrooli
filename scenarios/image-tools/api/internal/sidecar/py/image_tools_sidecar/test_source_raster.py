"""EXIF coordinate provenance matches the actual oriented pixel raster."""
import os
import tempfile
import unittest

from PIL import Image

from image_tools_sidecar._common import load_oriented_image


class SourceRasterTest(unittest.TestCase):
    def test_all_exif_maps_match_decoded_pixel_centers(self):
        with tempfile.TemporaryDirectory() as directory:
            for orientation in range(1, 9):
                with self.subTest(orientation=orientation):
                    source = Image.new("RGB", (3, 2))
                    source.putdata([(index * 30, 0, 0) for index in range(6)])
                    exif = Image.Exif()
                    exif[274] = orientation
                    path = os.path.join(directory, f"oriented-{orientation}.png")
                    source.save(path, exif=exif)
                    result, provenance = load_oriented_image(path, 6)
                    expected_size = (2, 3) if orientation >= 5 else (3, 2)
                    self.assertEqual(result.size, expected_size)
                    matrix = provenance["original_to_oriented_matrix_row_major"]
                    for y in range(2):
                        for x in range(3):
                            original = (x + .5, y + .5, 1)
                            transformed = [sum(a * b for a, b in zip(row, original)) for row in matrix]
                            self.assertEqual(result.getpixel((int(transformed[0]), int(transformed[1]))), source.getpixel((x, y)))
                    self.assertEqual(provenance["original_exif_orientation"], orientation)

    def test_pixel_limit_is_checked_before_orientation_or_full_decode(self):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "oversized.png")
            Image.new("RGB", (3, 2)).save(path)
            with self.assertRaisesRegex(ValueError, "exceeds 5 pixels"):
                load_oriented_image(path, 5)


if __name__ == "__main__":
    unittest.main()
