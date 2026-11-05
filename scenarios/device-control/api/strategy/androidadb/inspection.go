package androidadb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

// Inspect uses a fixed read-only vocabulary. Probe failures remain unknown;
// defaults must not select a client architecture on the operator's behalf.
func (a *Adapter) Inspect(ctx context.Context) (strategy.DeviceInspection, error) {
	result := strategy.DeviceInspection{Unavailable: map[string]string{}}
	probe := func(field string, args ...string) string {
		out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err != nil || len(out) == 0 || len(out) > 128*1024 {
			result.Unavailable[field] = "read-only probe failed or returned an invalid response size"
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	if value := probe("abis", "shell", "getprop", "ro.product.cpu.abilist"); value != "" {
		for _, abi := range strings.Split(value, ",") {
			if abi = strings.TrimSpace(abi); abi != "" {
				result.ABIs = append(result.ABIs, abi)
			}
		}
	}
	mem := probe("memory_total_bytes", "shell", "cat", "/proc/meminfo")
	if match := regexp.MustCompile(`(?m)^MemTotal:\s+(\d+)\s+kB`).FindStringSubmatch(mem); len(match) == 2 {
		result.MemoryTotalBytes = kibibytes(match[1])
	}
	if result.MemoryTotalBytes == nil {
		result.Unavailable["memory_total_bytes"] = "MemTotal was not readable"
	}
	disk := probe("data_storage", "shell", "df", "-k", "/data")
	for _, line := range strings.Split(disk, "\n") {
		fields := strings.Fields(line)
		// `df /data` can report an OEM's backing mount name, such as /mnt/…,
		// rather than /data. The command already scopes the filesystem.
		if len(fields) >= 6 {
			total, available := kibibytes(fields[1]), kibibytes(fields[3])
			if total != nil && available != nil {
				result.DataTotalBytes, result.DataAvailableBytes = total, available
			}
		}
	}
	if result.DataTotalBytes == nil || result.DataAvailableBytes == nil {
		result.Unavailable["data_storage"] = "data filesystem size was not readable"
	}
	provider := probe("webview", "shell", "dumpsys", "webviewupdate")
	if match := regexp.MustCompile(`Current WebView package \(name, version\):\s*\(([^,\s]+),\s*([^\)]+)\)`).FindStringSubmatch(provider); len(match) == 3 {
		result.WebViewPackage, result.WebViewVersion = match[1], strings.TrimSpace(match[2])
	} else {
		result.Unavailable["webview"] = "active WebView provider was not identifiable"
	}
	if user, err := a.currentAndroidUser(ctx); err == nil {
		result.HomeComponent, _ = a.resolveHome(ctx, user, "")
	}
	if result.HomeComponent == "" {
		result.Unavailable["home_component"] = "default Home activity was not resolvable"
	}
	lowRAM := probe("low_ram", "shell", "getprop", "ro.config.low_ram")
	if lowRAM == "true" || lowRAM == "false" {
		value := lowRAM == "true"
		result.LowRAM = &value
	} else {
		result.Unavailable["low_ram"] = "low-RAM property was not explicitly set"
	}
	result.Voice = a.inspectVoice(ctx)
	var codecs []byte
	var codecErr error
	if bounded, ok := a.runner.(boundedRunner); ok {
		codecs, codecErr = bounded.RunBounded(ctx, "adb", 1024*1024, a.args("shell", "dumpsys", "media.player")...)
	} else {
		codecs, codecErr = a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "media.player")...)
	}
	if codecErr == nil && len(codecs) <= 1024*1024 {
		result.VideoEncoders = parseVideoEncoders(string(codecs))
	}
	if len(result.VideoEncoders) == 0 {
		result.Unavailable["video_encoders"] = "video encoder inventory was unavailable or unrecognized"
	} else if len(result.VideoEncoders) == 64 {
		result.Unavailable["video_encoders"] = "video encoder inventory reached the 64-entry limit and may be incomplete"
	}
	result.ObservedAt = time.Now().UTC()
	if len(result.ABIs) == 0 && result.MemoryTotalBytes == nil && result.HomeComponent == "" {
		return result, fmt.Errorf("device inspection could not establish any hardware or Home facts")
	}
	return result, nil
}

// parseVideoEncoders retains only Android's encoder identity and explicit
// acceleration flags, never unrelated media sessions or raw device output.
func parseVideoEncoders(data string) []strategy.VideoEncoder {
	section := strings.Index(data, "Encoder infos by media types:")
	if section < 0 {
		return nil
	}
	mimePattern := regexp.MustCompile(`^Media type '(video/[A-Za-z0-9.+_-]+)':$`)
	encoderPattern := regexp.MustCompile(`^\s*Encoder "([A-Za-z0-9._-]{1,160})" supports\s*$`)
	flagPattern := regexp.MustCompile(`^\s*(software-only|hw-accelerated):\s*([01])\b`)
	var result []strategy.VideoEncoder
	var mime string
	seen := map[string]bool{}
	for _, line := range strings.Split(data[section:], "\n") {
		if strings.HasPrefix(line, "Media type '") {
			mime = ""
			if match := mimePattern.FindStringSubmatch(strings.TrimSpace(line)); len(match) == 2 {
				mime = match[1]
			}
			continue
		}
		if match := encoderPattern.FindStringSubmatch(line); len(match) == 2 {
			if mime != "" && len(result) < 64 && !seen[mime+"/"+match[1]] {
				result = append(result, strategy.VideoEncoder{MIMEType: mime, Name: match[1]})
				seen[mime+"/"+match[1]] = true
			} else {
				mime = ""
			}
			continue
		}
		if len(result) == 0 || mime == "" {
			continue
		}
		if match := flagPattern.FindStringSubmatch(line); len(match) == 3 {
			value := match[2] == "1"
			if match[1] == "software-only" {
				result[len(result)-1].SoftwareOnly = &value
			} else {
				result[len(result)-1].HardwareAccelerated = &value
			}
		}
	}
	return result
}

func kibibytes(value string) *uint64 {
	number, err := strconv.ParseUint(value, 10, 64)
	if err != nil || number > ^uint64(0)/1024 {
		return nil
	}
	number *= 1024
	return &number
}
