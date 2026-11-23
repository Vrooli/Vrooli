"""LPIPS sidecar contracts that need no model weights: tiling and backbone mapping."""

import unittest

from image_tools_sidecar.lpips_metric import load_backbone, tiles


class _Param:
    def __init__(self):
        self.data = self
        self.value = None

    def copy_(self, v):
        self.value = v


class _Module:
    def __init__(self, has_params=True):
        if has_params:
            self.weight, self.bias = _Param(), _Param()


class _Slice:
    def __init__(self, names, conv):
        self._children = [(n, _Module(n in conv)) for n in names]

    def named_children(self):
        return iter(self._children)


class _Backbone:
    # AlexNet feature indices: convs at 0,3,6,8,10; ReLU/MaxPool carry no tensors.
    def __init__(self):
        conv = {"0", "3", "6", "8", "10"}
        self.slice1 = _Slice(["0", "1"], conv)
        self.slice2 = _Slice(["2", "3", "4"], conv)
        self.slice3 = _Slice(["5", "6", "7"], conv)
        self.slice4 = _Slice(["8", "9"], conv)
        self.slice5 = _Slice(["10", "11"], conv)


class _Model:
    def __init__(self):
        self.net = _Backbone()


class TilesTest(unittest.TestCase):
    def test_tiles_cover_the_image_exactly_once(self):
        for w, h in [(3024, 4032), (1024, 1024), (1500, 700), (1030, 1100), (50, 40)]:
            boxes = tiles(w, h)
            self.assertEqual(sum((x1 - x0) * (y1 - y0) for x0, y0, x1, y1 in boxes), w * h, (w, h))

    def test_a_sliver_merges_into_its_neighbour(self):
        boxes = tiles(1030, 1024)
        self.assertEqual(boxes, [(0, 0, 1030, 1024)])
        for x0, y0, x1, y1 in tiles(3024, 4032):
            self.assertGreaterEqual(x1 - x0, 64)
            self.assertGreaterEqual(y1 - y0, 64)


class BackboneTest(unittest.TestCase):
    def test_every_conv_tensor_is_loaded_by_features_index(self):
        state = {f"features.{i}.{p}": f"{i}{p}" for i in (0, 3, 6, 8, 10) for p in ("weight", "bias")}
        state["classifier.1.weight"] = "ignored"
        model = _Model()
        self.assertEqual(load_backbone(model, state), 10)
        self.assertEqual(model.net.slice2._children[1][1].weight.value, "3weight")
        self.assertEqual(model.net.slice5._children[0][1].bias.value, "10bias")

    def test_a_foreign_state_dict_loads_nothing(self):
        self.assertEqual(load_backbone(_Model(), {"layer1.0.weight": 1}), 0)


if __name__ == "__main__":
    unittest.main()
