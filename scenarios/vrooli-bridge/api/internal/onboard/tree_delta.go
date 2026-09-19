package onboard

// A working-tree ship updates the node's checkout IN PLACE. It writes the files
// that changed since the previous ship, deletes only the files an earlier ship
// delivered and the new snapshot no longer contains, and leaves everything else
// alone. Everything else includes every gitignored path the node itself owns:
// scenario SQLite databases under scenarios/*/data, build outputs, node_modules,
// and the checkout's .git directory.
//
// The previous design extracted the snapshot into a staging directory and swapped
// it over the checkout. Because a snapshot never contains ignored files, every
// ship deleted every in-tree scenario database on the node and left running
// services with a deleted working directory (2026-09-15, minimouse: 13 scenarios
// reporting "unable to open database file" after a month of ships).
//
// The control plane records what it delivered to each node directory
// (shipRecord). The node keeps only the digest of the last complete ship. When
// the two digests agree, the next ship transfers only changed files; when they
// disagree (first ship, interrupted ship, or a record lost on the control
// plane), every file is transferred again. Deletions always come from the
// control plane's record, so they can only ever name a path Bridge itself put
// there.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// syncDigestMarker prefixes the probe's report of the node's last complete ship.
const syncDigestMarker = "VBSYNCDIGEST="

// shipRecord is what the control plane last delivered to one node directory.
type shipRecord struct {
	Dest    string            `json:"dest"`
	Digest  string            `json:"digest"`
	Entries map[string]string `json:"entries"`
	At      time.Time         `json:"at"`
}

// shipRecordStore persists ship records on the control plane.
type shipRecordStore interface {
	Load(key string) (shipRecord, bool)
	Save(key string, record shipRecord) error
}

// fileShipRecordStore keeps one JSON record per node directory under dir.
type fileShipRecordStore struct {
	dir string
}

func (s fileShipRecordStore) path(key string) string {
	return filepath.Join(s.dir, key+".json")
}

func (s fileShipRecordStore) Load(key string) (shipRecord, bool) {
	data, err := os.ReadFile(s.path(key))
	if err != nil {
		return shipRecord{}, false
	}
	var record shipRecord
	if err := json.Unmarshal(data, &record); err != nil || record.Entries == nil {
		return shipRecord{}, false
	}
	return record, true
}

func (s fileShipRecordStore) Save(key string, record shipRecord) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, ".ship-record-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.path(key))
}

// shipRecordKey names the record for one SSH identity and node directory.
func shipRecordKey(conn Conn, dest string) string {
	sum := sha256.Sum256([]byte(conn.User + "@" + conn.Host + ":" + strconv.Itoa(conn.Port) + "|" + dest))
	return hex.EncodeToString(sum[:16])
}

// treeDelta is the work one ship does on the node.
type treeDelta struct {
	// Transfer is the repo-relative files to write.
	Transfer []string
	// Delete is the repo-relative files an earlier ship delivered that the new
	// snapshot no longer contains.
	Delete []string
	// Incremental reports that Transfer holds only the changed files.
	Incremental bool
}

// planTreeDelta compares the new snapshot with the previous ship. entries maps
// each snapshot file to its content fingerprint; a nil map disables the
// incremental comparison and transfers every file.
func planTreeDelta(files []string, entries map[string]string, prev shipRecord, havePrev bool, nodeDigest string) treeDelta {
	current := make(map[string]struct{}, len(files))
	for _, file := range files {
		current[file] = struct{}{}
	}
	var delta treeDelta
	if havePrev {
		for file := range prev.Entries {
			if _, keep := current[file]; keep || !safeShipPath(file) {
				continue
			}
			delta.Delete = append(delta.Delete, file)
		}
		sort.Strings(delta.Delete)
	}
	delta.Incremental = havePrev && entries != nil && nodeDigest != "" && nodeDigest == prev.Digest
	if !delta.Incremental {
		delta.Transfer = append([]string(nil), files...)
		return delta
	}
	for _, file := range files {
		if fingerprint, ok := entries[file]; !ok || fingerprint != prev.Entries[file] || mustRefreshOnEveryShip(file) {
			delta.Transfer = append(delta.Transfer, file)
		}
	}
	return delta
}

// setup materializes the selected Proto artifact over packages/proto/gen. That
// node-owned mutation can make a previously delivered working-tree file stale
// without changing the Bridge ship record. Re-send the Go contracts on every
// working-tree ship so setup cannot hide the operator's dirty contract from the
// next bootstrap/build.
func mustRefreshOnEveryShip(file string) bool {
	const prefix = "packages/proto/gen/go/"
	return strings.HasPrefix(filepath.ToSlash(file), prefix)
}

// safeShipPath accepts only a clean relative path inside the checkout. Records
// are written by this control plane, but a deletion is the one operation that
// must never be able to leave the checkout.
func safeShipPath(rel string) bool {
	if rel == "" || strings.ContainsRune(rel, 0) || strings.HasPrefix(rel, "/") || strings.Contains(rel, `\`) {
		return false
	}
	if path.Clean(rel) != rel {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." || part == "." {
			return false
		}
	}
	return true
}

func nulList(files []string) []byte {
	var b bytes.Buffer
	for _, file := range files {
		b.WriteString(file)
		b.WriteByte(0)
	}
	return b.Bytes()
}

// ---- remote commands ----
//
// Every POSIX command starts from the same destination and marker names. The
// commands run in the node user's login shell (zsh on macOS), so they avoid
// unmatched globs and keep shell-specific syntax inside explicit `sh -c`.

func posixTreePrelude(destDir string) string {
	assign := `dest="$HOME/vrooli"`
	if d := strings.TrimSpace(destDir); d != "" {
		assign = "dest=" + shellQuote(d)
	}
	return assign + `; parent=$(dirname "$dest"); base=$(basename "$dest"); marker="$parent/.$base.bridge-ship-digest"`
}

// buildTreeProbeCommand ensures the destination exists, removes staging and
// backup directories left by the retired swap design, and reports the
// destination plus the node's last complete ship digest.
func buildTreeProbeCommand(destDir, targetOS string) string {
	if targetOS == "windows" {
		return windowsTreeCommand(destDir, `New-Item -ItemType Directory -Force -Path $dest | Out-Null; Get-ChildItem -Force -LiteralPath $parent -Directory -Filter ('.'+$base+'.bridge-*') | Remove-Item -Recurse -Force; [Console]::WriteLine('`+syncDestMarker+`'+$dest); if(Test-Path -LiteralPath $marker){[Console]::WriteLine('`+syncDigestMarker+`'+(Get-Content -Raw -LiteralPath $marker).Trim())}`)
	}
	return posixTreePrelude(destDir) + `; mkdir -p "$dest" && find "$parent" -maxdepth 1 -type d \( -name ".$base.bridge-sync-*" -o -name ".$base.bridge-old-*" \) -exec rm -rf {} + ; printf '` + syncDestMarker + `%s\n' "$dest"; if [ -f "$marker" ]; then printf '` + syncDigestMarker + `%s\n' "$(cat "$marker")"; fi`
}

// buildTreeCleanStaleCommand removes non-ignored files from a target checkout
// before an authoritative working-tree ship. It first removes the old index
// entries from Git's view; otherwise git clean cannot remove tracked files from
// an older source layout. It deliberately never uses -x, so ignored node-owned
// data is preserved.
func buildTreeCleanStaleCommand(destDir, targetOS string) string {
	if targetOS == "windows" {
		return windowsTreeCommand(destDir, `if(Test-Path -LiteralPath (Join-Path $dest '.git')){ git -C $dest rm -r --cached --quiet . ':!.gitignore' ':!*/.gitignore'; git -C $dest clean -fd --quiet }`)
	}
	return posixTreePrelude(destDir) + `; if git -C "$dest" rev-parse --is-inside-work-tree >/dev/null 2>&1; then git -C "$dest" rm -r --cached --quiet . ':!.gitignore' ':!*/.gitignore'; git -C "$dest" clean -fd --quiet; fi`
}

// buildTreeReconcileStaleCommand removes tracked files from an older source
// layout after the current full snapshot has been extracted. git clean alone
// cannot remove those files because they are still tracked by the target's
// previous index; staging the received tree first turns obsolete paths into
// untracked files. Ignored node-owned files are never staged or cleaned.
func buildTreeReconcileStaleCommand(destDir, targetOS string) string {
	if targetOS == "windows" {
		return windowsTreeCommand(destDir, `if(Test-Path -LiteralPath (Join-Path $dest '.git')){ git -C $dest add -A -- .; git -C $dest clean -fd --quiet }`)
	}
	return posixTreePrelude(destDir) + `; if git -C "$dest" rev-parse --is-inside-work-tree >/dev/null 2>&1; then git -C "$dest" add -A -- . && git -C "$dest" clean -fd --quiet; fi`
}

// buildTreeDeleteCommand removes the NUL-separated relative paths on stdin and
// any directory those removals left empty. The digest marker goes first, so an
// interrupted ship is never mistaken for a complete one.
func buildTreeDeleteCommand(destDir, targetOS string) string {
	if targetOS == "windows" {
		return windowsTreeCommand(destDir, `if(Test-Path -LiteralPath $marker){Remove-Item -Force -LiteralPath $marker}; $list=[Console]::In.ReadToEnd(); foreach($p in $list.Split([char]0)){ if($p){ $f=Join-Path $dest $p; if(Test-Path -LiteralPath $f -PathType Leaf){Remove-Item -Force -LiteralPath $f} } }`)
	}
	script := `for p do rm -f -- "$p"; d=$(dirname -- "$p"); while [ "$d" != . ] && [ "$d" != / ] && rmdir -- "$d" 2>/dev/null; do d=$(dirname -- "$d"); done; done`
	return posixTreePrelude(destDir) + `; rm -f "$marker"; cd "$dest" && xargs -0 sh -c ` + shellQuote(script) + ` sh`
}

// buildTreeExtractCommand writes the tar on stdin into the checkout. -U unlinks
// each file before writing it, so a running process keeps the inode it opened
// instead of seeing its file rewritten underneath it.
func buildTreeExtractCommand(destDir, targetOS string) string {
	if targetOS == "windows" {
		return windowsTreeCommand(destDir, `if(Test-Path -LiteralPath $marker){Remove-Item -Force -LiteralPath $marker}; New-Item -ItemType Directory -Force -Path $dest | Out-Null; tar.exe -xUf - -C $dest; if($LASTEXITCODE -ne 0){exit $LASTEXITCODE}`)
	}
	return posixTreePrelude(destDir) + `; rm -f "$marker"; mkdir -p "$dest" && tar -xUf - -C "$dest"`
}

// buildTreeFinalizeCommand records the digest of the ship that just completed.
func buildTreeFinalizeCommand(destDir, targetOS, digest string) string {
	if targetOS == "windows" {
		return windowsTreeCommand(destDir, `Set-Content -NoNewline -LiteralPath ($marker+'.tmp') -Value '`+windowsPowerShellLiteral(digest)+`'; Move-Item -Force -LiteralPath ($marker+'.tmp') -Destination $marker`)
	}
	return posixTreePrelude(destDir) + `; printf '%s' ` + shellQuote(digest) + ` > "$marker.tmp" && mv -f "$marker.tmp" "$marker"`
}

func windowsTreeCommand(destDir, body string) string {
	dest := strings.TrimSpace(destDir)
	assign := `$dest='` + windowsPowerShellLiteral(dest) + `'`
	if dest == "" {
		assign = `$dest=$env:USERPROFILE+'\vrooli'`
	} else if strings.HasPrefix(dest, "$env:") {
		assign = `$dest=` + dest
	}
	return `powershell.exe -NoProfile -NonInteractive -Command "$ErrorActionPreference='Stop'; ` + assign + `; $parent=Split-Path -Parent $dest; $base=Split-Path -Leaf $dest; $marker=Join-Path $parent ('.'+$base+'.bridge-ship-digest'); ` + body + `"`
}

// shipSummary renders the step detail for a completed ship.
func shipSummary(res SyncResult, snapshotFiles int) string {
	mode := "full"
	if res.Incremental {
		mode = "incremental"
	}
	return fmt.Sprintf("%s ship: wrote %d of %d file(s) (%s), removed %d", mode, res.FilesTransferred, snapshotFiles, humanBytes(res.BytesTransferred), res.FilesDeleted)
}
