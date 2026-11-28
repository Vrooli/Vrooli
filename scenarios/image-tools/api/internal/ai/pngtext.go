package ai

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"io"
	"os"
	"strconv"
)

// A sidecar that renders an image writes a receipt (call arguments, placement,
// GPU peak) into the PNG as a tEXt chunk, so the evidence travels with the
// artifact. Two things in this package depend on it: a Go-side rewrite of the
// pixels (matching the source frame) must carry the chunks across, and the job
// record must report the measured peak the receipt holds.

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// receiptKeywords are the tEXt keywords the sidecars write their receipt under.
var receiptKeywords = []string{"vrooli_diffusers_provenance", "vrooli_masked_edit_provenance"}

type pngChunk struct {
	kind string
	raw  []byte // length, type, data and CRC exactly as stored
	data []byte
}

// pngChunks splits a PNG into its chunks; ok is false for anything that is not a well-formed PNG.
func pngChunks(png []byte) (chunks []pngChunk, ok bool) {
	if !bytes.HasPrefix(png, pngSignature) {
		return nil, false
	}
	for rest := png[len(pngSignature):]; len(rest) > 0; {
		if len(rest) < 12 {
			return nil, false
		}
		size := int(binary.BigEndian.Uint32(rest[:4]))
		if size < 0 || len(rest) < 12+size {
			return nil, false
		}
		chunks = append(chunks, pngChunk{kind: string(rest[4:8]), raw: rest[:12+size], data: rest[8 : 8+size]})
		rest = rest[12+size:]
	}
	return chunks, len(chunks) > 0 && chunks[0].kind == "IHDR"
}

func isPNGTextChunk(kind string) bool { return kind == "tEXt" || kind == "zTXt" || kind == "iTXt" }

// pngTextChunks returns the text chunks of a PNG, in order.
func pngTextChunks(png []byte) []pngChunk {
	chunks, ok := pngChunks(png)
	if !ok {
		return nil
	}
	var text []pngChunk
	for _, c := range chunks {
		if isPNGTextChunk(c.kind) {
			text = append(text, c)
		}
	}
	return text
}

// withPNGTextChunks inserts text chunks right after IHDR of a PNG that has none
// of its own, leaving the pixel data untouched.
func withPNGTextChunks(png []byte, text []pngChunk) []byte {
	chunks, ok := pngChunks(png)
	if !ok || len(text) == 0 {
		return png
	}
	var out bytes.Buffer
	out.Write(pngSignature)
	out.Write(chunks[0].raw)
	for _, c := range text {
		out.Write(c.raw)
	}
	for _, c := range chunks[1:] {
		out.Write(c.raw)
	}
	return out.Bytes()
}

// newPNGTextChunk builds a tEXt chunk (used by tests and by callers that stamp a receipt).
func newPNGTextChunk(keyword, text string) pngChunk {
	data := append(append([]byte(keyword), 0), text...)
	raw := make([]byte, 8, 12+len(data))
	binary.BigEndian.PutUint32(raw[:4], uint32(len(data)))
	copy(raw[4:8], "tEXt")
	raw = append(raw, data...)
	sum := crc32.ChecksumIEEE(raw[4:])
	raw = binary.BigEndian.AppendUint32(raw, sum)
	return pngChunk{kind: "tEXt", raw: raw, data: data}
}

// receiptMeta reads the sidecar's receipt from the PNG at path and returns the
// measured facts the job record should keep: where the run was placed and its
// CUDA peak. It scans chunk headers only and stops at the pixel data, so a large
// artifact is never read whole. It returns nil for an output with no receipt (a
// CPU or cloud route, a non-PNG).
func receiptMeta(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()
	reader := bufio.NewReader(file)
	signature := make([]byte, len(pngSignature))
	if _, err := io.ReadFull(reader, signature); err != nil || !bytes.Equal(signature, pngSignature) {
		return nil
	}
	for {
		var head [8]byte
		if _, err := io.ReadFull(reader, head[:]); err != nil {
			return nil
		}
		size, kind := int64(binary.BigEndian.Uint32(head[:4])), string(head[4:])
		if kind == "IDAT" || kind == "IEND" {
			return nil
		}
		if kind != "tEXt" || size > maxReceiptBytes {
			if _, err := reader.Discard(int(size) + 4); err != nil {
				return nil
			}
			continue
		}
		data := make([]byte, size)
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil
		}
		if _, err := reader.Discard(4); err != nil {
			return nil
		}
		if meta := parseReceipt(data); meta != nil {
			return meta
		}
	}
}

// maxReceiptBytes bounds one text chunk the scan will read.
const maxReceiptBytes = 1 << 20

// parseReceipt turns one tEXt chunk's data into job meta when it is a receipt.
func parseReceipt(data []byte) map[string]string {
	keyword, text, found := bytes.Cut(data, []byte{0})
	if !found || !isReceiptKeyword(string(keyword)) {
		return nil
	}
	var receipt struct {
		Device    string `json:"execution_device"`
		Placement string `json:"placement"`
		Allocated *int64 `json:"gpu_peak_allocated_bytes"`
		Reserved  *int64 `json:"gpu_peak_reserved_bytes"`
	}
	if json.Unmarshal(text, &receipt) != nil {
		return nil
	}
	meta := map[string]string{}
	if receipt.Device != "" {
		meta["inference_device"] = receipt.Device
	}
	if receipt.Placement != "" {
		meta["inference_placement"] = receipt.Placement
	}
	if receipt.Allocated != nil {
		meta["inference_cuda_peak_allocated_bytes"] = strconv.FormatInt(*receipt.Allocated, 10)
	}
	if receipt.Reserved != nil {
		meta["inference_cuda_peak_reserved_bytes"] = strconv.FormatInt(*receipt.Reserved, 10)
	}
	return meta
}

func isReceiptKeyword(keyword string) bool {
	for _, k := range receiptKeywords {
		if k == keyword {
			return true
		}
	}
	return false
}
