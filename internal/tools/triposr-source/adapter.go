package triposrsource

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/binaryfetch"
)

const AdapterVersion = "vrooli-triposr-config-skimage-v6"

var sourceFileHashes = map[string]string{
	"tsr/system.py":            "7aec93e421394ad7c746b6d63a0189532b57b26af9a199fc75ac3bb7c993c94f",
	"tsr/utils.py":             "d9d1a645fef7ea489615bff3471fd01a4941a2c0e138c5dff5daae619563279b",
	"tsr/models/isosurface.py": "94c9a2ec27f8fe3c2fb57558ef28be253eb77074d2c171d49a41cf73d95d6f5e",
}

const configAdapter = `"""Strict, data-only configuration adapter for Vrooli's pinned TripoSR model."""
from dataclasses import MISSING, fields, is_dataclass
from typing import Any, get_args, get_origin, get_type_hints

import yaml


def _convert(expected, value, path):
    origin = get_origin(expected)
    args = get_args(expected)
    if origin is not None and type(None) in args:
        if value is None:
            return None
        expected = next(arg for arg in args if arg is not type(None))
        origin = get_origin(expected)
        args = get_args(expected)
    if is_dataclass(expected):
        if not isinstance(value, dict):
            raise TypeError(f"{path}: expected mapping")
        return _strict_dataclass(expected, value, path)
    if origin in (dict,):
        if not isinstance(value, dict):
            raise TypeError(f"{path}: expected mapping")
        kt, vt = args if len(args) == 2 else (Any, Any)
        return {_convert(kt, k, path + ".<key>"): _convert(vt, v, path + "." + str(k)) for k, v in value.items()}
    if origin in (list, tuple):
        if not isinstance(value, (list, tuple)):
            raise TypeError(f"{path}: expected sequence")
        item_type = args[0] if args else Any
        converted = [_convert(item_type, item, f"{path}[{i}]") for i, item in enumerate(value)]
        return tuple(converted) if origin is tuple else converted
    if expected is Any:
        return value
    if expected is bool:
        if type(value) is not bool:
            raise TypeError(f"{path}: expected boolean")
    elif expected is int:
        if type(value) is not int:
            raise TypeError(f"{path}: expected integer")
    elif expected is float:
        if type(value) not in (float, int):
            raise TypeError(f"{path}: expected number")
        value = float(value)
    elif expected is str and not isinstance(value, str):
        raise TypeError(f"{path}: expected string")
    return value


def _strict_dataclass(cls, values, path):
    definitions = {field.name: field for field in fields(cls)}
    unknown = set(values) - set(definitions)
    if unknown:
        raise ValueError(f"{path}: unknown fields: {', '.join(sorted(unknown))}")
    hints = get_type_hints(cls)
    kwargs = {}
    for name, field in definitions.items():
        if name in values:
            kwargs[name] = _convert(hints.get(name, Any), values[name], path + "." + name)
        elif field.default is MISSING and field.default_factory is MISSING:
            raise ValueError(f"{path}: missing required field {name}")
    return cls(**kwargs)


def parse_structured(fields_cls, cfg=None):
    if isinstance(cfg, fields_cls):
        return cfg
    return _strict_dataclass(fields_cls, cfg or {}, "config")


def load_config(path, fields_cls):
    with open(path, "r", encoding="utf-8") as stream:
        raw = yaml.safe_load(stream)
    if not isinstance(raw, dict):
        raise TypeError("model config must be a mapping")
    # The pinned checkpoint contains exactly one literal reference. Resolve
    # only that typed field/value pair; no resolver expressions are executed.
    backbone = raw.get("backbone")
    tokenizer = raw.get("tokenizer")
    if isinstance(backbone, dict) and backbone.get("in_channels") == "${tokenizer.num_channels}":
        if not isinstance(tokenizer, dict) or tokenizer.get("num_channels") != 1024:
            raise ValueError("config.backbone.in_channels: pinned tokenizer.num_channels must equal 1024")
        backbone["in_channels"] = 1024
    # Any other OmegaConf interpolation or resolver syntax is unsupported.
    def reject_interpolation(value, location="config"):
        if isinstance(value, str) and "${" in value:
            raise ValueError(f"{location}: interpolation is unsupported")
        if isinstance(value, dict):
            for key, child in value.items(): reject_interpolation(child, location + "." + str(key))
        elif isinstance(value, list):
            for index, child in enumerate(value): reject_interpolation(child, f"{location}[{index}]")
    reject_interpolation(raw)
    return _strict_dataclass(fields_cls, raw, "config")
`

const adapterNotice = `Vrooli maintains this TripoSR inference adapter over upstream VAST-AI-Research/TripoSR.
Upstream source is pinned at commit 107cefdc244c39106fa830359024f6a2f1c78871 and retained separately.
Changes: strict PyYAML/dataclass config loading replaces OmegaConf and resolves only the checkpoint's literal
tokenizer.num_channels reference (validated as 1024); unused rembg and imageio imports are lazy;
CPU scikit-image marching cubes replaces compiled torchmcubes, with axis conversion and outward triangle winding
validated on an asymmetric sphere. Checkpoint weights are unchanged.
The upstream MIT license and copyright notices are retained in LICENSE.
`

func sha256Hex(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func patchFile(root, name, expected, replacement string) error {
	path := filepath.Join(root, filepath.FromSlash(name))
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if sha256Hex(b) != expected {
		return fmt.Errorf("pinned adapter source hash mismatch for %s", name)
	}
	out := []byte(replacement)
	return os.WriteFile(path, out, 0644)
}

func applySourceAdapter(original, adapted string) (string, error) {
	if _, err := os.Stat(filepath.Join(adapted, "tsr", "config_adapter.py")); err == nil {
		digest, err := binaryfetch.TreeDigest(adapted)
		return digest, err
	}
	if err := copyTree(original, adapted); err != nil {
		return "", err
	}
	for name, want := range sourceFileHashes {
		b, err := os.ReadFile(filepath.Join(original, filepath.FromSlash(name)))
		if err != nil {
			return "", err
		}
		if sha256Hex(b) != want {
			return "", fmt.Errorf("pinned adapter source hash mismatch for %s", name)
		}
	}
	// system.py: switch only checkpoint config loading to the strict pinned adapter.
	systemPath := filepath.Join(adapted, "tsr", "system.py")
	system, err := os.ReadFile(systemPath)
	if err != nil {
		return "", err
	}
	s := strings.Replace(string(system), "from omegaconf import OmegaConf\n", "from .config_adapter import load_config\n", 1)
	s = strings.Replace(s, "cfg = OmegaConf.load(config_path)\n        OmegaConf.resolve(cfg)", "cfg = load_config(config_path, cls.Config)", 1)
	if s == string(system) {
		return "", errors.New("pinned system.py adapter anchors did not match")
	}
	if err = os.WriteFile(systemPath, []byte(s), 0644); err != nil {
		return "", err
	}
	// utils.py: use strict typed construction for the source's actual dataclass configs.
	utilsPath := filepath.Join(adapted, "tsr", "utils.py")
	utils, err := os.ReadFile(utilsPath)
	if err != nil {
		return "", err
	}
	u := strings.Replace(string(utils), "import imageio\n", "", 1)
	u = strings.Replace(u, "import rembg\n", "", 1)
	u = strings.Replace(u, "from omegaconf import DictConfig, OmegaConf\n", "from .config_adapter import parse_structured\n", 1)
	u = strings.ReplaceAll(u, "cfg: Optional[Union[dict, DictConfig]] = None", "cfg: Optional[dict] = None")
	parseStructured := regexp.MustCompile(`(?s)def parse_structured\(fields: Any, cfg: Optional\[.*?\n\n\n`)
	u = parseStructured.ReplaceAllString(u, "")
	u = strings.Replace(u, "        image = rembg.remove(image, session=rembg_session, **rembg_kwargs)", "        import rembg\n        image = rembg.remove(image, session=rembg_session, **rembg_kwargs)", 1)
	u = strings.Replace(u, "    writer = imageio.get_writer(output_path, fps=fps)", "    import imageio\n    writer = imageio.get_writer(output_path, fps=fps)", 1)
	if u == string(utils) || strings.Contains(u, "omegaconf") {
		return "", errors.New("pinned utils.py adapter anchors did not match")
	}
	if err = os.WriteFile(utilsPath, []byte(u), 0644); err != nil {
		return "", err
	}
	// CPU scikit-image backend replaces the undeclared compiled torchmcubes extension.
	isoPath := filepath.Join(adapted, "tsr", "models", "isosurface.py")
	iso, err := os.ReadFile(isoPath)
	if err != nil {
		return "", err
	}
	i := strings.Replace(string(iso), "from torchmcubes import marching_cubes", "from skimage.measure import marching_cubes as skimage_marching_cubes\n\ndef marching_cubes(volume, isovalue):\n    vertices, faces, _, _ = skimage_marching_cubes(volume.detach().cpu().numpy(), level=float(isovalue), allow_degenerate=False)\n    # Reverse volume axes for the upstream x,y,z vertex convention.\n    return torch.as_tensor(vertices[..., ::-1].copy(), dtype=torch.float32), torch.as_tensor(faces.copy(), dtype=torch.long)", 1)
	if i == string(iso) {
		return "", errors.New("pinned isosurface.py adapter anchor did not match")
	}
	if err = os.WriteFile(isoPath, []byte(i), 0644); err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(adapted, "tsr", "config_adapter.py"), []byte(configAdapter), 0644); err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(adapted, "VROOLI_ADAPTER_NOTICE.txt"), []byte(adapterNotice), 0644); err != nil {
		return "", err
	}
	if err = freezeTree(adapted); err != nil {
		return "", err
	}
	return binaryfetch.TreeDigest(adapted)
}

func freezeTree(root string) error {
	var dirs []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported adapter entry %s", path)
		}
		return os.Chmod(path, 0444)
	}); err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := os.Chmod(dirs[i], 0555); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(dst, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported non-regular source entry %s", path)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
