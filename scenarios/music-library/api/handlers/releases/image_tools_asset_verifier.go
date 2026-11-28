package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	jobsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"
	domain "music-library/internal/releases"
)

const maxVerifiedArtworkBytes = 32 << 20

type imageToolsAssetVerifier struct {
	baseURL func(context.Context) (string, error)
	client  *http.Client
}

func newImageToolsAssetVerifier() *imageToolsAssetVerifier {
	return &imageToolsAssetVerifier{
		baseURL: imageToolsBaseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func imageToolsBaseURL(ctx context.Context) (string, error) {
	base := strings.TrimSpace(os.Getenv("IMAGE_TOOLS_URL"))
	if base == "" {
		var err error
		base, err = discovery.ResolveScenarioURLDefault(ctx, "image-tools")
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
		return "", errors.New("Image Tools URL must be explicit loopback HTTP")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func (v *imageToolsAssetVerifier) VerifyAsset(ctx context.Context, asset domain.Asset) (bool, error) {
	if asset.OwnerScenario != "image-tools" || !strings.HasPrefix(asset.Ref, "outputs/") ||
		path.Clean(asset.Ref) != asset.Ref || strings.ContainsAny(asset.Ref, "?#") || strings.ContainsRune(asset.Ref, rune(92)) ||
		asset.OperationID == "" || asset.ModelID == "" || asset.LicenseLane == "" ||
		asset.ProvenanceRef != "image-tools:"+asset.OperationID {
		return false, nil
	}
	base, err := v.baseURL(ctx)
	if err != nil {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	httpClient := v.client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	jobs := jobsv1connect.NewJobsServiceClient(httpClient, base)
	jobResponse, err := jobs.GetJob(ctx, connect.NewRequest(&jobsv1.GetJobRequest{Id: asset.OperationID}))
	if err != nil || jobResponse.Msg.Job == nil {
		return false, nil
	}
	job := jobResponse.Msg.Job
	if job.State != jobsv1.JobState_JOB_STATE_SUCCEEDED || job.Request == nil || job.Request.ModelId != asset.ModelID || !hasResultRef(job, asset.Ref) {
		return false, nil
	}
	models := modelsv1connect.NewModelsServiceClient(httpClient, base)
	modelResponse, err := models.GetModel(ctx, connect.NewRequest(&modelsv1.GetModelRequest{Id: asset.ModelID}))
	if err != nil || modelResponse.Msg.Model == nil || modelResponse.Msg.Model.CapabilityLabels == nil {
		return false, nil
	}
	labels := modelResponse.Msg.Model.CapabilityLabels
	if labels.License == "" || labels.License != asset.LicenseLane {
		return false, nil
	}
	if asset.RightsStatus == "commercial_cleared" && labels.CommercialUse != modelsv1.CommercialUse_COMMERCIAL_USE_YES {
		return false, nil
	}

	target, err := url.Parse(base)
	if err != nil {
		return false, nil
	}
	target.Path = strings.TrimRight(target.Path, "/") + "/api/v1/blobs/" + asset.Ref
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return false, nil
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return false, nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "image/") {
		return false, nil
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxVerifiedArtworkBytes+1))
	if err != nil || len(body) == 0 || len(body) > maxVerifiedArtworkBytes {
		return false, nil
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]) == asset.SHA256, nil
}

func hasResultRef(job *jobsv1.Job, ref string) bool {
	if job.ResultRef == ref {
		return true
	}
	for _, output := range job.ResultRefs {
		if output == ref {
			return true
		}
	}
	return false
}
