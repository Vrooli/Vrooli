package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Downloader fetches a remote artifact to a local path. It is a seam so a catalog
// installer is testable without network access; the production implementation is
// HTTPDownloader. emit reports byte progress (total may be -1 if unknown).
type Downloader interface {
	Download(ctx context.Context, rawURL, destPath string, emit func(done, total int64)) error
}

// HTTPDownloader is the production Downloader: a streaming HTTP GET.
type HTTPDownloader struct {
	Client *http.Client
}

// Download streams rawURL to destPath, reporting byte progress.
func (d HTTPDownloader) Download(ctx context.Context, rawURL, destPath string, emit func(done, total int64)) error {
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", rawURL, resp.Status)
	}

	f, err := os.Create(destPath) //nolint:gosec // destPath is engine-constructed under Root
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	src := &progressReader{r: resp.Body, total: total, emit: emit}
	if _, err := io.Copy(f, src); err != nil {
		return err
	}
	return nil
}

type progressReader struct {
	r     io.Reader
	done  int64
	total int64
	emit  func(done, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.done += int64(n)
	if pr.emit != nil {
		pr.emit(pr.done, pr.total)
	}
	return n, err
}
