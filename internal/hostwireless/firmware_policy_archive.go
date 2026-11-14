//go:build linux

package hostwireless

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/mikelolasagasti/xz"
)

func firmwarePolicyDecode(ctx context.Context, data []byte, suffix string, limit int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if suffix == "" {
		if int64(len(data)) > limit {
			return nil, errors.New("raw image exceeds bound")
		}
		return data, nil
	}
	var r io.Reader
	switch suffix {
	case ".zst":
		// The kernel's full-read implementation requires a declared frame size.
		var h zstd.Header
		if err := h.Decode(data); err != nil {
			return nil, err
		}
		if err := firmwarePolicySingleZstd(data, h); err != nil {
			return nil, err
		}
		if !h.HasFCS || h.FrameContentSize > uint64(limit) || h.Skippable {
			return nil, errors.New("unsupported or oversized zstd frame")
		}
		z, err := zstd.NewReader(bytes.NewReader(data), zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(uint64(limit)+1), zstd.WithDecoderMaxWindow(uint64(limit)))
		if err != nil {
			return nil, err
		}
		defer z.Close()
		r = z
	case ".initrd.xz":
		z, err := xz.NewReader(bytes.NewReader(data), 64<<20)
		if err != nil {
			return nil, err
		}
		r = z
	case ".xz":
		// firmware_loader stops on unsupported integrity checks; only none/CRC32
		// are supported by its in-kernel XZ decoder (lib/xz/xz_dec_stream.c).
		if len(data) < 12 || data[7] > 1 {
			return nil, errors.New("unsupported firmware XZ integrity check")
		}
		source := &firmwarePolicyByteReader{data: data}
		z, err := xz.NewReader(source, 64<<20)
		if err != nil {
			return nil, err
		}
		z.Multistream(false)
		r = &firmwarePolicyXZReader{Reader: z, source: source}
	case ".gz":
		z, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer z.Close()
		r = z
	default:
		return nil, errors.New("unsupported compression")
	}
	b, err := io.ReadAll(io.LimitReader(firmwarePolicyContextReader{ctx, r}, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("decompressed image exceeds bound")
	}
	return b, ctx.Err()
}

type firmwarePolicyContextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r firmwarePolicyContextReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(b)
}

// Supported boot images contain zero padding, one or more uncompressed newc
// archives (e.g. early microcode), then optionally a gzip/xz/zstd stream whose
// complete output is one or more newc archives. Mixed trailing compressed
// formats, unknown codecs and malformed/truncated trailers fail closed. No
// archive contents are extracted or interpreted as commands.
func firmwarePolicyArchive(ctx context.Context, data []byte) ([]string, error) {
	var names []string
	seen := map[string]uint64{}
	remaining := data
	decoded := false
	archives := 0
	var expanded int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		remaining = bytes.TrimLeft(remaining, "\x00")
		if len(remaining) == 0 {
			if archives == 0 {
				return nil, errors.New("empty initramfs inventory")
			}
			return names, nil
		}
		if !bytes.HasPrefix(remaining, []byte("070701")) && !bytes.HasPrefix(remaining, []byte("070702")) {
			if decoded {
				return nil, errors.New("unsupported trailing initramfs segment")
			}
			suffix := ""
			switch {
			case bytes.HasPrefix(remaining, []byte{0x1f, 0x8b}):
				suffix = ".gz"
			case bytes.HasPrefix(remaining, []byte{0xfd, '7', 'z', 'X', 'Z', 0}):
				suffix = ".xz"
			case bytes.HasPrefix(remaining, []byte{0x28, 0xb5, 0x2f, 0xfd}):
				suffix = ".zst"
			default:
				return nil, errors.New("unsupported initramfs compression or segment")
			}
			// Unlike firmware, initramfs zstd streams need not declare content size.
			var err error
			if suffix == ".zst" {
				remaining, err = firmwarePolicyInitrdZstd(ctx, remaining, firmwarePolicyInitramfsLimit-expanded)
			} else {
				if suffix == ".xz" {
					suffix = ".initrd.xz"
				}
				remaining, err = firmwarePolicyDecode(ctx, remaining, suffix, firmwarePolicyInitramfsLimit-expanded)
			}
			if err != nil {
				return nil, err
			}
			decoded = true
			continue
		}
		archives++
		consumed, archiveNames, err := firmwarePolicyCPIO(ctx, remaining, seen)
		if err != nil {
			return nil, err
		}
		expanded += int64(consumed)
		if expanded > firmwarePolicyInitramfsLimit {
			return nil, errors.New("initramfs expansion exceeds bound")
		}
		names = append(names, archiveNames...)
		if len(names) > 200000 {
			return nil, errors.New("initramfs entry bound exceeded")
		}
		remaining = remaining[consumed:]
	}
}
func firmwarePolicyInitrdZstd(ctx context.Context, data []byte, limit int64) ([]byte, error) {
	z, err := zstd.NewReader(bytes.NewReader(data), zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(firmwarePolicyInitramfsLimit+1), zstd.WithDecoderMaxWindow(firmwarePolicyInitramfsLimit))
	if err != nil {
		return nil, err
	}
	defer z.Close()
	b, err := io.ReadAll(io.LimitReader(firmwarePolicyContextReader{ctx, z}, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("initramfs expansion exceeds bound")
	}
	return b, nil
}
func firmwarePolicyCPIO(ctx context.Context, data []byte, seen map[string]uint64) (int, []string, error) {
	offset := 0
	var names []string
	for {
		if err := ctx.Err(); err != nil {
			return 0, nil, err
		}
		if len(names) > 200000 {
			return 0, nil, errors.New("initramfs entry bound exceeded")
		}
		if len(data)-offset < 110 {
			return 0, nil, errors.New("truncated cpio header")
		}
		h := data[offset : offset+110]
		if string(h[:6]) != "070701" && string(h[:6]) != "070702" {
			return 0, nil, errors.New("invalid cpio magic")
		}
		fields := make([]uint64, 13)
		for i := range fields {
			v, err := strconv.ParseUint(string(h[6+i*8:14+i*8]), 16, 32)
			if err != nil {
				return 0, nil, errors.New("malformed cpio header")
			}
			fields[i] = v
		}
		nameSize, size := fields[11], fields[6]
		if nameSize < 1 || nameSize > 4096 || size > firmwarePolicyInitramfsLimit {
			return 0, nil, errors.New("invalid cpio entry size")
		}
		nameEnd := uint64(offset) + 110 + nameSize
		contentStart := (nameEnd + 3) &^ uint64(3)
		contentEnd := contentStart + size
		next := (contentEnd + 3) &^ uint64(3)
		if next > uint64(len(data)) {
			return 0, nil, errors.New("truncated cpio entry")
		}
		rawName := data[uint64(offset)+110 : nameEnd]
		if rawName[len(rawName)-1] != 0 || bytes.IndexByte(rawName[:len(rawName)-1], 0) >= 0 {
			return 0, nil, errors.New("invalid cpio name termination")
		}
		name := strings.TrimPrefix(string(rawName[:len(rawName)-1]), "./")
		if string(h[:6]) == "070702" {
			var sum uint32
			for i, b := range data[contentStart:contentEnd] {
				if i%(1<<20) == 0 {
					if err := ctx.Err(); err != nil {
						return 0, nil, err
					}
				}
				sum += uint32(b)
			}
			if uint64(sum) != fields[12] {
				return 0, nil, errors.New("cpio checksum differs")
			}
		}
		offset = int(next)
		if name == "TRAILER!!!" {
			if size != 0 {
				return 0, nil, errors.New("nonempty cpio trailer")
			}
			return offset, names, nil
		}
		if name == "." {
			continue
		}
		if !firmwarePolicyPath(name) {
			return 0, nil, fmt.Errorf("unsafe or unsupported cpio name %q", name)
		}
		mode := fields[1] & 0170000
		if old, exists := seen[name]; exists {
			if mode != 0040000 || old != mode {
				return 0, nil, errors.New("ambiguous duplicate initramfs entry")
			}
		} else {
			seen[name] = mode
			names = append(names, name)
		}
	}
}

// Byte-sized reads make the XZ end boundary observable despite decoder buffering.
type firmwarePolicyByteReader struct {
	data   []byte
	offset int
}

func (r *firmwarePolicyByteReader) Read(b []byte) (int, error) {
	if r.offset == len(r.data) {
		return 0, io.EOF
	}
	if len(b) == 0 {
		return 0, nil
	}
	b[0] = r.data[r.offset]
	r.offset++
	return 1, nil
}

type firmwarePolicyXZReader struct {
	io.Reader
	source *firmwarePolicyByteReader
}

func (r *firmwarePolicyXZReader) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	if err == io.EOF && r.source.offset != len(r.source.data) {
		return n, errors.New("trailing XZ data requires separate qualification")
	}
	return n, err
}
func firmwarePolicySingleZstd(data []byte, h zstd.Header) error {
	offset := h.HeaderSize
	for {
		if len(data)-offset < 3 {
			return io.ErrUnexpectedEOF
		}
		v := uint32(data[offset]) | uint32(data[offset+1])<<8 | uint32(data[offset+2])<<16
		offset += 3
		kind := (v >> 1) & 3
		size := int(v >> 3)
		if kind == 3 {
			return errors.New("reserved zstd block")
		}
		if kind == 1 {
			size = 1
		}
		if size > len(data)-offset {
			return io.ErrUnexpectedEOF
		}
		offset += size
		if v&1 != 0 {
			break
		}
	}
	if h.HasCheckSum {
		offset += 4
	}
	if offset != len(data) {
		return errors.New("truncated or multiple zstd frames")
	}
	return nil
}
