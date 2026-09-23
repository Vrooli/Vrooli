// Package capturequalification owns the fixed native capture workload and its
// independent oracle. It emits observations; Performance Health owns qualification.
package capturequalification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
)

const fixtureHTML = `<!doctype html><meta name="viewport" content="width=device-width, initial-scale=1"><title>Capture qualification</title><style>html,body{margin:0;width:100%%;height:100%%;overflow:hidden}section{position:absolute;width:50vw;height:50vh}h1{font:16px sans-serif;color:white;position:absolute;top:4px;left:8px;margin:0}#control{position:absolute;left:24px;top:70px;width:180px}#paint{position:absolute;left:8px;top:400px;width:32px;height:32px;background:rgb(%d,%d,%d)}</style><section style="left:0;top:0;background:rgb(210,30,50)"><h1 id="ready">Capture fixture %s</h1><label for="control">Independent input</label><input id="control" value="fixture-value"></section><section style="left:50vw;top:0;background:rgb(20,180,70)"></section><section style="left:0;top:50vh;background:rgb(20,60,210)"></section><section style="left:50vw;top:50vh;background:rgb(230,185,20)"></section><div id="paint"></div><script>addEventListener('load',()=>fetch('/observe',{method:'POST',body:JSON.stringify({url:location.href,width:innerWidth,height:innerHeight,dpr:devicePixelRatio,heading:document.querySelector('h1').textContent,value:document.querySelector('input').value})}));</script>`

type observation struct {
	URL     string  `json:"url"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	DPR     float64 `json:"dpr"`
	Heading string  `json:"heading"`
	Value   string  `json:"value"`
}

type fixture struct {
	*httptest.Server
	nonce string
	mu    sync.Mutex
	seen  map[string][]observation
}

func newFixture(nonce string) *fixture {
	f := &fixture{nonce: nonce, seen: make(map[string][]observation)}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	return f
}

func (f *fixture) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/observe" {
		var o observation
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&o); err != nil {
			http.Error(w, "invalid observation", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.seen[o.URL] = append(f.seen[o.URL], o)
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet || r.URL.Path != "/fixture" {
		http.NotFound(w, r)
		return
	}
	c := paintColor(f.nonce, r.URL.Query().Get("trial"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(w, fixtureHTML, c[0], c[1], c[2], f.nonce)
}

func paintColor(nonce, trial string) [3]byte {
	h := sha256.Sum256([]byte(nonce + ":" + trial))
	return [3]byte{h[0], h[1], h[2]}
}

func digest(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (f *fixture) observations(url string) []observation {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]observation(nil), f.seen[url]...)
}
