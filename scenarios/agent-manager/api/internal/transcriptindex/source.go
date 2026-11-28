package transcriptindex

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// A transcript source is a file whose decoded bytes are one JSON object per
// line. Harnesses store finished transcripts in more than one encoding (Codex
// compresses old rollouts to .jsonl.zst), so every reader goes through
// OpenTranscript and every offset, size and head fingerprint the indexer keeps
// is in decoded bytes. A compressed transcript is sealed: its writer never
// appends to it again.

// transcriptEncoding is one on-disk transcript encoding.
type transcriptEncoding struct {
	suffix string
	// sealed encodings are written once; their decoded bytes never change.
	sealed bool
	// decode wraps the raw file; nil reads it as is.
	decode func(io.Reader) (io.ReadCloser, error)
	// decodedSize reports the decoded length from the raw file; nil means the
	// raw size is the decoded size.
	decodedSize func(*os.File) (int64, error)
}

// transcriptEncodings lists every encoding discovery accepts, most specific
// suffix first. Adding a harness's encoding is one entry here.
var transcriptEncodings = []transcriptEncoding{
	{suffix: ".jsonl.zst", sealed: true, decode: decodeZstd, decodedSize: zstdDecodedSize},
	{suffix: ".jsonl"},
}

func encodingOf(path string) (transcriptEncoding, bool) {
	for _, encoding := range transcriptEncodings {
		if strings.HasSuffix(path, encoding.suffix) {
			return encoding, true
		}
	}
	return transcriptEncoding{}, false
}

// IsTranscriptName reports whether a file name has a transcript encoding.
func IsTranscriptName(name string) bool {
	_, ok := encodingOf(name)
	return ok
}

// TranscriptStem is a transcript file name without its encoding suffix.
func TranscriptStem(name string) string {
	if encoding, ok := encodingOf(name); ok {
		return strings.TrimSuffix(name, encoding.suffix)
	}
	return name
}

// Sealed reports a transcript whose encoding is never appended to.
func Sealed(path string) bool {
	encoding, ok := encodingOf(path)
	return ok && encoding.sealed
}

// OpenTranscript opens a transcript's decoded bytes.
func OpenTranscript(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	encoding, _ := encodingOf(path)
	if encoding.decode == nil {
		return file, nil
	}
	decoded, err := encoding.decode(bufio.NewReader(file))
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return &decodedFile{ReadCloser: decoded, file: file}, nil
}

// OpenTranscriptAt opens a transcript's decoded bytes positioned at offset.
func OpenTranscriptAt(path string, offset int64) (io.ReadCloser, error) {
	source, err := OpenTranscript(path)
	if err != nil {
		return nil, err
	}
	if offset <= 0 {
		return source, nil
	}
	if seeker, ok := source.(io.Seeker); ok {
		if _, err := seeker.Seek(offset, io.SeekStart); err != nil {
			source.Close()
			return nil, err
		}
		return source, nil
	}
	if skipped, err := io.CopyN(io.Discard, source, offset); err != nil {
		source.Close()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("transcript %s ends at %d before offset %d", path, skipped, offset)
		}
		return nil, err
	}
	return source, nil
}

type decodedFile struct {
	io.ReadCloser
	file *os.File
}

func (d *decodedFile) Close() error {
	return errors.Join(d.ReadCloser.Close(), d.file.Close())
}

func decodeZstd(raw io.Reader) (io.ReadCloser, error) {
	decoder, err := zstd.NewReader(raw, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true))
	if err != nil {
		return nil, err
	}
	return decoder.IOReadCloser(), nil
}

// zstdDecodedSize reads the decoded length the frame header declares (zstd
// writes it for a file compressed whole, as Codex rollouts are), or decodes
// the file to measure it when the header omits it.
func zstdDecodedSize(file *os.File) (int64, error) {
	head := make([]byte, zstd.HeaderMaxSize)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return 0, err
	}
	var header zstd.Header
	if header.Decode(head[:n]) == nil && header.HasFCS && !header.Skippable {
		return int64(header.FrameContentSize), nil
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	decoded, err := decodeZstd(bufio.NewReader(file))
	if err != nil {
		return 0, err
	}
	defer decoded.Close()
	return io.Copy(io.Discard, decoded)
}

// decodedSizes memoises sealed files' decoded lengths by raw identity: a walk
// runs every tick, and a sealed file's length never changes.
var decodedSizes sync.Map // decodedSizeKey -> int64

type decodedSizeKey struct {
	key       string
	raw       int64
	mtimeNano int64
}

// transcriptSize is the decoded length of the transcript whose raw file info
// is info.
func transcriptSize(path, key string, info os.FileInfo) (int64, error) {
	encoding, _ := encodingOf(path)
	if encoding.decodedSize == nil {
		return info.Size(), nil
	}
	memo := decodedSizeKey{key: key, raw: info.Size(), mtimeNano: info.ModTime().UnixNano()}
	if size, ok := decodedSizes.Load(memo); ok {
		return size.(int64), nil
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	size, err := encoding.decodedSize(file)
	if err != nil {
		return 0, fmt.Errorf("measure %s: %w", path, err)
	}
	decodedSizes.Store(memo, size)
	return size, nil
}
