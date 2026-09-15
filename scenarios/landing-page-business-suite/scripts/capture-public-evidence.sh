#!/usr/bin/env bash
set -euo pipefail

# Capture rendered landing-page evidence. Chrome must be visible for x11grab;
# --headless is intentionally used only for still screenshots.

base_url="${1:-http://127.0.0.1:23224}"
output_dir="${2:-.vrooli/artifacts/lpbs-evidence}"
variant_slug="${3:-control}"
mkdir -p "$output_dir"

for command_name in google-chrome ffmpeg ffprobe xvfb-run xdotool; do
  command -v "$command_name" >/dev/null || {
    echo "capture failed: required command is missing: $command_name" >&2
    exit 1
  }
done

url="${base_url%/}/?variant_slug=${variant_slug}"
desktop_png="$output_dir/public-landing-${variant_slug}-desktop.png"
mobile_png="$output_dir/public-landing-${variant_slug}-mobile.png"
video="$output_dir/public-landing-${variant_slug}-desktop.mp4"

google-chrome --headless --no-sandbox --disable-gpu --virtual-time-budget=5000 \
  --no-first-run --no-default-browser-check \
  --window-size=1440,1800 --screenshot="$desktop_png" "$url" >/dev/null 2>&1
google-chrome --headless --no-sandbox --disable-gpu --virtual-time-budget=5000 \
  --no-first-run --no-default-browser-check \
  --window-size=390,844 --screenshot="$mobile_png" "$url" >/dev/null 2>&1

tmp_dir="$(mktemp -d)"
cleanup() {
  if [[ -n "${chrome_pid:-}" ]]; then
    kill "$chrome_pid" 2>/dev/null || true
    wait "$chrome_pid" 2>/dev/null || true
  fi
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

timeout 30s xvfb-run -a -s '-screen 0 1280x1024x24' bash -c '
  set -euo pipefail
  google-chrome --app="$2" --no-sandbox --disable-gpu --disable-dev-shm-usage \
    --no-first-run --no-default-browser-check \
    --test-type \
    --user-data-dir="$1/chrome" --window-size=1280,1024 >"$1/chrome.log" 2>&1 &
  chrome_pid=$!
  sleep 3
  ffmpeg -y -loglevel error -video_size 1280x1024 -framerate 12 \
    -f x11grab -draw_mouse 0 -i "$DISPLAY.0" -t 12 \
    -c:v libx264 -pix_fmt yuv420p "$3" &
  record_pid=$!
  sleep 4
  window_id="$(xdotool search --onlyvisible --class google-chrome | tail -1)"
  if [[ -n "$window_id" ]]; then
    xdotool mousemove --window "$window_id" 700 500 click 1 || true
    xdotool key --window "$window_id" End || true
    xdotool mousemove --window "$window_id" 1 1 || true
  fi
  wait "$record_pid"
  kill "$chrome_pid" 2>/dev/null || true
  wait "$chrome_pid" 2>/dev/null || true
' capture "$tmp_dir" "$url" "$video"

[[ -s "$desktop_png" && -s "$mobile_png" && -s "$video" ]] || {
  echo "capture failed: one or more evidence files are empty" >&2
  exit 1
}

duration="$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$video")"
awk -v duration="$duration" 'BEGIN { exit !(duration >= 10) }' || {
  echo "capture failed: video duration is ${duration}s" >&2
  exit 1
}

dom_file="$tmp_dir/rendered-dom.html"
google-chrome --headless --no-sandbox --disable-gpu --virtual-time-budget=5000 \
  --no-first-run --no-default-browser-check --dump-dom "$url" >"$dom_file" 2>/dev/null
rg -q 'data-experience-state="ready"' "$dom_file" || {
  echo "capture failed: rendered DOM never reached ready state" >&2
  exit 1
}
rg -q 'data-testid="(bundle-app-web-console|open-app-primary-web-console)"' "$dom_file" || {
  echo "capture failed: enabled Aquila catalog marker is missing" >&2
  exit 1
}

stats_file="$tmp_dir/signalstats.txt"
ffmpeg -hide_banner -loglevel error -i "$video" -vf \
  'signalstats,metadata=print:file=/dev/stderr' -frames:v 1 -f null - 2>"$stats_file" || true
yavg="$(awk -F= '/lavfi.signalstats.YAVG/ { print $2; exit }' "$stats_file")"
awk -v yavg="${yavg:-0}" 'BEGIN { exit !(yavg > 2) }' || {
  echo "capture failed: first frame luma average is ${yavg:-0}; refusing black evidence" >&2
  exit 1
}

black_log="$tmp_dir/blackdetect.txt"
ffmpeg -hide_banner -i "$video" -vf 'blackdetect=d=2:pix_th=0.05' -an -f null - \
  2>"$black_log" || true
if rg -q 'black_start:0 .*black_duration:[0-9.]{2,}' "$black_log"; then
  echo "capture failed: recording contains a sustained black opening" >&2
  exit 1
fi

printf 'Captured valid evidence:\n  %s\n  %s\n  %s\n' "$desktop_png" "$mobile_png" "$video"
