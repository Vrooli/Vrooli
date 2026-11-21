//go:build !pii_onnx

package main

// This stub is the default build: it provides no native NER inference, so the
// server runs the deterministic regex/checksum pass only. Building with the
// `pii_onnx` tag selects engine_onnx.go instead, which runs the real BERT
// token-classification model via the onnxruntime_go binding.
//
// The default build intentionally does NOT depend on the ONNX Runtime binding or
// any staged model, so a host with neither still builds and serves the
// deterministic regex/checksum pass. The real NER path (engine_onnx.go) is fully
// implemented and validated (server/engine_onnx_test.go, eval_onnx_test.go); it
// needs the staged, checksum-pinned small-tier model (model-policy.json
// models.*.artifact) plus libonnxruntime.so at runtime.

func newNERModelFromEnv() (NERModel, error) {
	if modelLoadDisabled() {
		return UnavailableModel{reason: "PII_DETECTOR_LOAD_MODEL=off: capacity ladder pinned the server to the regex-only rung"}, nil
	}
	return UnavailableModel{reason: "built without the pii_onnx tag; rebuild with -tags pii_onnx and a staged ONNX model"}, nil
}
