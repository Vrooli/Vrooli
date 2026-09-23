package capturequalification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func screenshot(t *testing.T, paint [3]byte) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	for _, p := range []struct {
		x, y int
		c    color.RGBA
	}{{0, 0, color.RGBA{210, 30, 50, 255}}, {640, 0, color.RGBA{20, 180, 70, 255}},
		{0, 360, color.RGBA{20, 60, 210, 255}}, {640, 360, color.RGBA{230, 185, 20, 255}}} {
		draw.Draw(im, image.Rect(p.x, p.y, p.x+640, p.y+360), image.NewUniform(p.c), image.Point{}, draw.Src)
	}
	draw.Draw(im, image.Rect(8, 400, 40, 432), image.NewUniform(color.RGBA{paint[0], paint[1], paint[2], 255}), image.Point{}, draw.Src)
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func goodResponse(t *testing.T, root, nonce, id string, paint [3]byte) captureResponse {
	t.Helper()
	var c captureResponse
	c.ExecutionID, c.DurationMS, c.Readiness.Outcome = id, 500, "ready"
	c.DOMTreeJSON = fmt.Sprintf(`{"rect":{"width":1280,"height":720},"children":[{"id":"ready","text":"Capture fixture %s"},{"tagName":"SECTION","computed":{"backgroundColor":"rgb(210, 30, 50)"}},{"tagName":"SECTION"},{"tagName":"SECTION"},{"tagName":"SECTION"}]}`, nonce)
	c.Artifacts = []artifact{
		{Type: "screenshot", Primary: true, Path: filepath.Join(root, id+".png"), Reference: "bas-capture://" + id + "/screenshot"},
		{Type: "dom-tree", Path: filepath.Join(root, id+".json"), Reference: "bas-capture://" + id + "/dom_tree"},
	}
	if err := os.WriteFile(c.Artifacts[0].Path, screenshot(t, paint), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.Artifacts[1].Path, []byte(c.DOMTreeJSON), 0600); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestOracleRejectsFalseSuccessfulCapture(t *testing.T) {
	for _, fault := range []string{"none", "readiness", "duration", "operation", "observation", "DPR", "input", "duplicate-observation", "tree", "style", "missing-image", "missing-tree", "primary", "stored-tree", "tree-owner", "blank-image", "stale-image", "scaled-image", "corrupt-image"} {
		t.Run(fault, func(t *testing.T) {
			nonce, trial, captureURL := "nonce", "007", "http://127.0.0.1/fixture?trial=007"
			c := goodResponse(t, t.TempDir(), nonce, "owned", paintColor(nonce, trial))
			o := []observation{{captureURL, 1280, 720, 1, "Capture fixture " + nonce, "fixture-value"}}
			switch fault {
			case "readiness":
				c.Readiness.Outcome = "timeout"
			case "duration":
				c.DurationMS = 0
			case "operation":
				c.ExecutionID = ""
			case "observation":
				o = nil
			case "DPR":
				o[0].DPR = 2
			case "input":
				o[0].Value = "wrong"
			case "duplicate-observation":
				o = append(o, o[0])
			case "tree":
				c.DOMTreeJSON = `{}`
			case "style":
				c.DOMTreeJSON = strings.ReplaceAll(c.DOMTreeJSON, "rgb(210, 30, 50)", "rgb(0, 0, 0)")
			case "missing-image":
				c.Artifacts = c.Artifacts[1:]
			case "missing-tree":
				c.Artifacts = c.Artifacts[:1]
			case "primary":
				c.Artifacts[0].Primary = false
			case "stored-tree":
				_ = os.WriteFile(c.Artifacts[1].Path, []byte(`{}`), 0600)
			case "tree-owner":
				c.Artifacts[1].Reference = "bas-capture://another/dom_tree"
			case "stale-image":
				_ = os.WriteFile(c.Artifacts[0].Path, screenshot(t, paintColor(nonce, "006")), 0600)
			case "blank-image", "scaled-image":
				width := 1280
				if fault == "scaled-image" {
					width = 2560
				}
				var b bytes.Buffer
				if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, width, 720))); err != nil {
					t.Fatal(err)
				}
				_ = os.WriteFile(c.Artifacts[0].Path, b.Bytes(), 0600)
			case "corrupt-image":
				_ = os.WriteFile(c.Artifacts[0].Path, []byte("not a PNG"), 0600)
			}
			artifacts, err := verifyCapture(c, o, nonce, trial, captureURL)
			if (err == nil) != (fault == "none") {
				t.Fatalf("fault=%s error=%v", fault, err)
			}
			if err == nil && (len(artifacts) != 2 || len(artifacts[0].SHA256) != 64 || artifacts[0].Bytes == 0) {
				t.Fatal("missing retained artifact identity")
			}
		})
	}
}

func testConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	for _, path := range []string{"api/internal/capturequalification/fixture.go", "api/cmd/capture-cohort/main.go", "docs/internal/REFRACTOR_CONTRACT.json"} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(path), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return Config{Output: filepath.Join(root, "output"), ScenarioRoot: root, APIURL: "managed-api", DriverURL: "managed-driver"}
}

func fixedIdentity(context.Context) (candidate, error) {
	return candidate{BuildIdentity: "sha256:" + strings.Repeat("a", 64), BrowserVersion: "136", DriverVersion: "2"}, nil
}

func TestCohortRetainsEveryFirstAttemptAndNeverRetries(t *testing.T) {
	cfg := testConfig(t)
	calls := 0
	receipt, err := run(t.Context(), cfg, func(context.Context, string, string) ([]byte, []byte, error) {
		calls++
		return []byte(`{"execution_id":"failed-owner"}`), []byte("admission failed"), errors.New("transport failed")
	}, fixedIdentity)
	if err == nil || calls != 101 || len(receipt.Attempts) != 101 {
		t.Fatalf("calls=%d receipt=%+v error=%v", calls, receipt, err)
	}
	for i, a := range receipt.Attempts {
		if a.Index != i-1 || a.Warmup != (i == 0) || a.Status != "failed" || a.Error == "" {
			t.Fatalf("lost failure %+v", a)
		}
		b, readErr := os.ReadFile(filepath.Join(cfg.Output, a.StderrFile))
		if readErr != nil || string(b) != "admission failed" {
			t.Fatalf("lost stderr: %q %v", b, readErr)
		}
	}
	var saved Receipt
	b, err := os.ReadFile(filepath.Join(cfg.Output, "receipt.json"))
	if err != nil || json.Unmarshal(b, &saved) != nil || len(saved.Attempts) != 101 || len(saved.Errors) == 0 {
		t.Fatal("terminal failed receipt missing")
	}
	_, err = run(t.Context(), cfg, nil, nil)
	if err == nil {
		t.Fatal("existing receipt directory was reused")
	}
}

func TestCancelledCohortCannotShrinkDenominator(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	receipt, err := run(ctx, testConfig(t), func(context.Context, string, string) ([]byte, []byte, error) {
		calls++
		cancel()
		return nil, nil, context.Canceled
	}, fixedIdentity)
	if err == nil || calls != 1 || len(receipt.Attempts) != 101 {
		t.Fatalf("calls=%d attempts=%d err=%v", calls, len(receipt.Attempts), err)
	}
	for _, a := range receipt.Attempts[1:] {
		if a.Status != "not_attempted" {
			t.Fatalf("cancelled sample %v", a)
		}
	}
}

func TestCohortIndependentFixtureAndCandidateStability(t *testing.T) {
	for _, changed := range []string{"none", "build", "producer", "contract", "duplicate-operation"} {
		t.Run(changed, func(t *testing.T) {
			cfg := testConfig(t)
			artifacts := t.TempDir()
			identityCalls, captures := 0, 0
			identity := func(ctx context.Context) (candidate, error) {
				identityCalls++
				c, _ := fixedIdentity(ctx)
				if identityCalls == 2 {
					switch changed {
					case "build":
						c.BuildIdentity = "different"
					case "producer":
						_ = os.WriteFile(filepath.Join(cfg.ScenarioRoot, "api/internal/capturequalification/fixture.go"), []byte("changed"), 0600)
					case "contract":
						_ = os.WriteFile(filepath.Join(cfg.ScenarioRoot, "docs/internal/REFRACTOR_CONTRACT.json"), []byte("changed"), 0600)
					}
				}
				return c, nil
			}
			receipt, err := run(t.Context(), cfg, func(ctx context.Context, captureURL, label string) ([]byte, []byte, error) {
				captures++
				res, err := http.Get(captureURL)
				if err != nil {
					t.Fatal(err)
				}
				html, _ := io.ReadAll(res.Body)
				_ = res.Body.Close()
				nonce := regexp.MustCompile(`Capture fixture ([a-f0-9]+)`).FindStringSubmatch(string(html))[1]
				components := regexp.MustCompile(`#paint\{[^}]+background:rgb\((\d+),(\d+),(\d+)\)`).FindStringSubmatch(string(html))
				var paint [3]byte
				for i := range paint {
					n, _ := strconv.Atoi(components[i+1])
					paint[i] = byte(n)
				}
				o := observation{captureURL, 1280, 720, 1, "Capture fixture " + nonce, "fixture-value"}
				b, _ := json.Marshal(o)
				u, _ := url.Parse(captureURL)
				res, err = http.Post(u.Scheme+"://"+u.Host+"/observe", "application/json", bytes.NewReader(b))
				if err != nil {
					t.Fatal(err)
				}
				_ = res.Body.Close()
				id := fmt.Sprintf("capture-%d", captures)
				if changed == "duplicate-operation" {
					id = "reused"
				}
				c := goodResponse(t, artifacts, nonce, id, paint)
				b, _ = json.Marshal(c)
				return b, nil, nil
			}, identity)
			if (err == nil) != (changed == "none") {
				t.Fatalf("change=%s err=%v", changed, err)
			}
			if captures != 101 || len(receipt.Attempts) != 101 {
				t.Fatal("wrong first-attempt denominator")
			}
			if changed == "none" {
				for _, a := range receipt.Attempts {
					if a.Status != "verified" {
						t.Fatalf("not verified: %+v", a)
					}
				}
			}
		})
	}
}

func TestHealthRequiresActualBuildAndReadyBrowser(t *testing.T) {
	for _, fault := range []string{"none", "missing-build", "nonhex-build", "browser-unready", "missing-version", "http-error"} {
		t.Run(fault, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if fault == "http-error" {
					http.Error(w, "unavailable", 503)
					return
				}
				build := "sha256:" + strings.Repeat("a", 64)
				if fault == "missing-build" {
					build = ""
				}
				if fault == "nonhex-build" {
					build = "sha256:" + strings.Repeat("z", 64)
				}
				version := "136"
				if fault == "missing-version" {
					version = ""
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"build_identity": build, "ready": fault != "browser-unready", "version": "2", "browser": map[string]string{"version": version}})
			}))
			defer s.Close()
			_, err := readIdentity(t.Context(), Config{APIURL: s.URL, DriverURL: s.URL})
			if (err == nil) != (fault == "none") {
				t.Fatalf("fault=%s err=%v", fault, err)
			}
		})
	}
}

func TestCaptureOutputLimitIncludesCopyFastPaths(t *testing.T) {
	b := &boundedBuffer{remaining: 10}
	n, err := io.Copy(b, strings.NewReader(strings.Repeat("x", 100)))
	if n != 10 || err == nil || len(b.Bytes()) != 10 {
		t.Fatalf("unbounded copy: %d bytes, %v", n, err)
	}
}
