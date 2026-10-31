#!/usr/bin/env python3
"""Deterministically export the pii-detector LARGE tier to ONNX.

The large tier is ab-ai/pii_model_based_on_distilbert — an Apache-2.0, uncased
DistilBertForTokenClassification (WordPiece, plain BIO, 65 labels over ~32 PII
categories). Unlike the small tier (gravitee-io/bert-small-pii-detection, which
publishes a prebuilt model.quant.onnx on the Hub), ab-ai publishes only the
PyTorch safetensors; the ONNX graph is produced here.

Because the ONNX is a BUILD OUTPUT, its sha256 depends on the exporter versions.
The AUTHORITATIVE, reproducibly-pinnable artifacts are the Hugging Face source
files (model.safetensors, vocab.txt, config.json, tokenizer_config.json); their
sha256s are pinned in model-policy.json. The exported model.onnx sha256 is also
recorded there as a build-output fingerprint for the exact exporter below.

Exporter pinned to the environment the large-tier eval was measured on:
  torch 2.12.x (legacy TorchScript exporter: dynamo=False), onnx 1.23.x,
  transformers 4.57.x, opset 14, do_constant_folding=True.

The engine (server/engine_onnx.go) is model-agnostic: it reads the graph I/O
names, config.json id2label, and the do_lower_case flag at load time, so this
export needs no engine change — it is a pure model swap behind the NERModel seam.

Usage:
  python3 scripts/export_large_model.py --out <dir>
  # then point the resource at it:
  #   PII_DETECTOR_MODEL_DIR=<dir> PII_DETECTOR_ONNX_FILE=model.onnx \
  #   PII_DETECTOR_MODEL=ab-ai/pii_model_based_on_distilbert
"""

import argparse
import hashlib
import os
import sys

REPO = "ab-ai/pii_model_based_on_distilbert"
OPSET = 14


def sha256(path: str) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True, help="output directory for the staged model")
    ap.add_argument("--repo", default=REPO)
    args = ap.parse_args()

    import torch
    from transformers import AutoModelForTokenClassification, AutoTokenizer

    os.makedirs(args.out, exist_ok=True)
    print(f"loading {args.repo} ...", flush=True)
    tok = AutoTokenizer.from_pretrained(args.repo)
    model = AutoModelForTokenClassification.from_pretrained(args.repo)
    model.eval()

    # Stage the tokenizer + config next to the ONNX; the engine reads vocab.txt,
    # config.json (id2label + do_lower_case), and tokenizer_config.json.
    tok.save_pretrained(args.out)
    model.config.save_pretrained(args.out)

    enc = tok("My name is John Smith, email john@acme.com", return_tensors="pt")
    onnx_path = os.path.join(args.out, "model.onnx")
    print("exporting ONNX ...", flush=True)
    torch.onnx.export(
        model,
        (enc["input_ids"], enc["attention_mask"]),
        onnx_path,
        input_names=["input_ids", "attention_mask"],
        output_names=["logits"],
        dynamic_axes={
            "input_ids": {0: "batch", 1: "seq"},
            "attention_mask": {0: "batch", 1: "seq"},
            "logits": {0: "batch", 1: "seq"},
        },
        opset_version=OPSET,
        do_constant_folding=True,
        dynamo=False,
    )

    print(f"\nstaged into {args.out}:")
    for f in sorted(os.listdir(args.out)):
        p = os.path.join(args.out, f)
        if os.path.isfile(p):
            print(f"  {os.path.getsize(p):>12}  {sha256(p)}  {f}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
