package releases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	domain "music-library/internal/releases"
)

type musicToolsDerivationVerifier struct {
	baseURL func(context.Context) (string, error)
	client  *http.Client
}

var musicToolsJobIDRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$`)
var musicToolsHashRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

type musicToolsDerivationReceipt struct {
	SchemaVersion        string  `json:"schema_version"`
	JobID                string  `json:"job_id"`
	Kind                 string  `json:"kind"`
	SourceTrackSHA256    string  `json:"source_track_sha256"`
	DerivedTrackSHA256   string  `json:"derived_track_sha256"`
	SourceStartSeconds   float64 `json:"source_start_seconds"`
	SourceEndSeconds     float64 `json:"source_end_seconds"`
	DerivedStartSeconds  float64 `json:"derived_start_seconds"`
	DerivedEndSeconds    float64 `json:"derived_end_seconds"`
	ResultRef            string  `json:"result_ref"`
	OperationVersion     string  `json:"operation_version"`
	ToolName             string  `json:"tool_name"`
	ToolVersion          string  `json:"tool_version"`
	ModelID              string  `json:"model_id,omitempty"`
	ModelRevision        string  `json:"model_revision,omitempty"`
	LicenseLane          string  `json:"license_lane,omitempty"`
	InheritedRightsScope string  `json:"inherited_rights_scope"`
}

func newMusicToolsDerivationVerifier() *musicToolsDerivationVerifier {
	return &musicToolsDerivationVerifier{
		baseURL: musicToolsDerivationBaseURL,
		client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
	}
}

func musicToolsDerivationBaseURL(ctx context.Context) (string, error) {
	base := strings.TrimSpace(os.Getenv("MUSIC_TOOLS_URL"))
	if base == "" {
		var err error
		base, err = discovery.ResolveScenarioURLDefault(ctx, "music-tools")
		if err != nil {
			return "", err
		}
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	ip := net.ParseIP(host)
	if u.Scheme != "http" || u.Port() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || !(host == "localhost" || (ip != nil && ip.IsLoopback())) {
		return "", errors.New("Music Tools URL must be explicit loopback HTTP")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func (v *musicToolsDerivationVerifier) VerifyDerivation(ctx context.Context, item domain.Derivation) (bool, error) {
	if item.OperationOwner != "music-tools" || !musicToolsJobIDRE.MatchString(item.OperationID) || item.EvidenceRef != "music-tools:"+item.OperationID ||
		(item.Kind != "remix" && item.Kind != "sample") || !musicToolsHashRE.MatchString(item.SourceTrackSHA256) || !musicToolsHashRE.MatchString(item.DerivedTrackSHA256) {
		return false, nil
	}
	baseURL := v.baseURL
	if baseURL == nil {
		baseURL = musicToolsDerivationBaseURL
	}
	base, err := baseURL(ctx)
	if err != nil {
		return false, nil
	}
	client := v.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	target, err := url.Parse(base)
	if err != nil {
		return false, nil
	}
	target.Path = strings.TrimRight(target.Path, "/") + "/api/v1/jobs/" + item.OperationID + "/derivation-receipt"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return false, nil
	}
	request.Header.Set("Accept", "application/json")
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	request = request.WithContext(ctx)
	response, err := client.Do(request)
	if err != nil {
		return false, nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "application/json") {
		return false, nil
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(body) == 0 || len(body) > 64<<10 {
		return false, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var receipt musicToolsDerivationReceipt
	if err := decoder.Decode(&receipt); err != nil {
		return false, nil
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return false, nil
	}
	return receiptMatches(item, receipt), nil
}

func receiptMatches(item domain.Derivation, receipt musicToolsDerivationReceipt) bool {
	return receipt.SchemaVersion == "1" && receipt.JobID == item.OperationID && receipt.Kind == item.Kind &&
		receipt.SourceTrackSHA256 == item.SourceTrackSHA256 && receipt.DerivedTrackSHA256 == item.DerivedTrackSHA256 &&
		receipt.SourceStartSeconds == item.SourceStartSeconds && receipt.SourceEndSeconds == item.SourceEndSeconds &&
		receipt.DerivedStartSeconds == item.DerivedStartSeconds && receipt.DerivedEndSeconds == item.DerivedEndSeconds &&
		receipt.ResultRef != "" && receipt.OperationVersion != "" && receipt.ToolName != "" && receipt.ToolVersion != "" &&
		finiteReceiptRange(receipt.SourceStartSeconds, receipt.SourceEndSeconds) &&
		finiteReceiptRange(receipt.DerivedStartSeconds, receipt.DerivedEndSeconds) &&
		receipt.InheritedRightsScope == item.InheritedRights &&
		((receipt.ModelID == "" && receipt.ModelRevision == "" && receipt.LicenseLane == "") ||
			(receipt.ModelID != "" && receipt.ModelRevision != "" && receipt.LicenseLane != ""))
}

func finiteReceiptRange(start, end float64) bool {
	return !math.IsNaN(start) && !math.IsInf(start, 0) && start >= 0 && !math.IsNaN(end) && !math.IsInf(end, 0) && end > start
}
