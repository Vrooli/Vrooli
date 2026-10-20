package renderjobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"device-studio/internal/httpx"
	"device-studio/internal/jobs"
	domain "device-studio/internal/renderjobs"
	"github.com/gorilla/mux"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deps struct { Coordinator *domain.Coordinator; Jobs jobs.Service; Logger *log.Logger }
type handler struct { deps Deps }

func NewConnectHandler(deps Deps) *handler {
	if deps.Logger == nil { deps.Logger = log.Default() }
	return &handler{deps: deps}
}

func (h *handler) SubmitRender(ctx context.Context, req *connect.Request[shotsv1.SubmitRenderRequest]) (*connect.Response[shotsv1.SubmitRenderResponse], error) {
	profile := domain.OutputProfilePNG
	if req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1 { return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported PNG output profile")) }
	job, created, err := h.deps.Coordinator.Submit(ctx, domain.SubmitRequest{ProjectID:req.Msg.ProjectId, ShotID:req.Msg.ShotId, Revision:req.Msg.Revision, IdempotencyKey:req.Msg.IdempotencyKey, OutputProfile:profile})
	if err != nil { return nil, h.connectError("submit render", err) }
	return connect.NewResponse(&shotsv1.SubmitRenderResponse{Job:jobToProto(job, ""), ReusedExisting:!created}), nil
}
func (h *handler) GetRender(ctx context.Context, req *connect.Request[shotsv1.GetRenderRequest]) (*connect.Response[shotsv1.GetRenderResponse], error) {
	job, err := h.deps.Coordinator.GetProjectJob(ctx, req.Msg.ProjectId, req.Msg.RenderId)
	if err != nil { return nil, h.connectError("get render", err) }
	return connect.NewResponse(&shotsv1.GetRenderResponse{Job:jobToProto(job, req.Msg.ProjectId)}), nil
}
func (h *handler) ListRenders(ctx context.Context, req *connect.Request[shotsv1.ListRendersRequest]) (*connect.Response[shotsv1.ListRendersResponse], error) {
	limit := int(req.Msg.PageSize); if limit == 0 { limit = 50 }; if limit < 1 || limit > 100 { return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page size must be between 1 and 100")) }
	cursor, err := decodeCursor(req.Msg.PageToken); if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid render page token")) }
	rows, err := h.deps.Coordinator.ListProjectJobs(ctx, req.Msg.ProjectId, cursor, limit+1)
	if err != nil { return nil, h.connectError("list renders", err) }
	more := len(rows)>limit; if more { rows=rows[:limit] }
	out := &shotsv1.ListRendersResponse{Jobs:make([]*shotsv1.RenderJob,0,len(rows))}
	for _, job := range rows { out.Jobs=append(out.Jobs,jobToProto(job,req.Msg.ProjectId)) }
	if more && len(rows)>0 { out.NextPageToken,err=encodeCursor(jobs.Cursor{CreatedAt:rows[len(rows)-1].CreatedAt,ID:rows[len(rows)-1].ID}); if err!=nil{return nil,connect.NewError(connect.CodeInternal,errors.New("render page encoding failed"))} }
	return connect.NewResponse(out),nil
}
func (h *handler) CancelRender(ctx context.Context, req *connect.Request[shotsv1.CancelRenderRequest]) (*connect.Response[shotsv1.CancelRenderResponse], error) {
	job, err := h.deps.Coordinator.GetProjectJob(ctx, req.Msg.ProjectId, req.Msg.RenderId)
	if err != nil { return nil, h.connectError("cancel render", err) }
	job, err = h.deps.Coordinator.Cancel(ctx, req.Msg.RenderId, req.Msg.ExpectedVersion)
	if err != nil { return nil, h.connectError("cancel render", err) }
	return connect.NewResponse(&shotsv1.CancelRenderResponse{Job:jobToProto(job,req.Msg.ProjectId)}),nil
}

func (h *handler) connectError(op string, err error) error {
	switch {
	case errors.Is(err,jobs.ErrNotFound): return connect.NewError(connect.CodeNotFound,errors.New("render job not found"))
	case errors.Is(err,jobs.ErrConcurrentTransition): return connect.NewError(connect.CodeAborted,errors.New("render job changed; reload before retrying"))
	case errors.Is(err,domain.ErrNotEligible): return connect.NewError(connect.CodeFailedPrecondition,errors.New("shot revision is not eligible for this still render"))
	case errors.Is(err,domain.ErrInvalidRequest), errors.Is(err,jobs.ErrInvalid): return connect.NewError(connect.CodeInvalidArgument,errors.New("invalid render request"))
	default: h.deps.Logger.Printf("%s failed: %v",op,err); return connect.NewError(connect.CodeInternal,errors.New("render operation failed"))
	}
}

type cursorToken struct { CreatedAt int64 `json:"created_at"`; ID string `json:"id"` }
func encodeCursor(c jobs.Cursor)(string,error){b,e:=json.Marshal(cursorToken{CreatedAt:c.CreatedAt.UTC().UnixNano(),ID:c.ID});if e!=nil{return "",e};return base64.RawURLEncoding.EncodeToString(b),nil}
func decodeCursor(token string)(jobs.Cursor,error){if token==""{return jobs.Cursor{},nil};b,e:=base64.RawURLEncoding.DecodeString(token);if e!=nil{return jobs.Cursor{},e};var c cursorToken;if e=json.Unmarshal(b,&c);e!=nil||c.CreatedAt<=0||strings.TrimSpace(c.ID)==""{return jobs.Cursor{},fmt.Errorf("invalid cursor")};return jobs.Cursor{CreatedAt:time.Unix(0,c.CreatedAt).UTC(),ID:c.ID},nil}

func jobToProto(j jobs.Job, projectID string)*shotsv1.RenderJob{
	status:=shotsv1.RenderJobStatus_RENDER_JOB_STATUS_NEEDS_RECONCILIATION
	var issue *shotsv1.RenderJobIssue
	switch j.State {
	case jobs.Queued: status=shotsv1.RenderJobStatus_RENDER_JOB_STATUS_QUEUED
	case jobs.Running: if a,ok:=lastAttempt(j);ok&&a.ExecutionObserved{status=shotsv1.RenderJobStatus_RENDER_JOB_STATUS_OBSERVED}else{issue=&shotsv1.RenderJobIssue{Code:shotsv1.RenderJobIssueCode_RENDER_JOB_ISSUE_CODE_OUTCOME_UNRESOLVED,Summary:"Execution has been claimed but its running state is not yet confirmed."}}
	case jobs.CancelRequested: status=shotsv1.RenderJobStatus_RENDER_JOB_STATUS_CANCEL_REQUESTED
	case jobs.Succeeded: status=shotsv1.RenderJobStatus_RENDER_JOB_STATUS_SUCCEEDED
	case jobs.Failed: status=shotsv1.RenderJobStatus_RENDER_JOB_STATUS_FAILED
	case jobs.Cancelled: status=shotsv1.RenderJobStatus_RENDER_JOB_STATUS_CANCELLED
	default: issue=&shotsv1.RenderJobIssue{Code:shotsv1.RenderJobIssueCode_RENDER_JOB_ISSUE_CODE_OUTCOME_UNRESOLVED,Summary:"Execution outcome requires reconciliation."}
	}
	p:=&shotsv1.RenderJob{Id:j.ID,ProjectId:j.Input.ProjectID,ShotId:shotIDFromInput(j.Input),ShotRevision:j.Input.ShotRevision,Status:status,Version:j.Version,CreatedAt:timestamppb.New(j.CreatedAt),UpdatedAt:timestamppb.New(j.UpdatedAt),Issue:issue}
	if p.ProjectId==""{p.ProjectId=projectID}
	if j.State==jobs.Failed { p.Issue=&shotsv1.RenderJobIssue{Code:shotsv1.RenderJobIssueCode_RENDER_JOB_ISSUE_CODE_OUTPUT_UNAVAILABLE,Summary:"The still render failed; inspect the job and submit a corrected revision."} }
	if a,ok:=lastAttempt(j);ok&&a.Completion!=nil { for _,o:=range a.Completion.Outputs { if o.Kind=="still_png" { p.Output=&shotsv1.RenderOutput{MediaType:o.MIMEType,SizeBytes:o.SizeBytes,WidthPixels:uint32(o.Width),HeightPixels:uint32(o.Height),Sha256:o.SHA256,ContentUrl:fmt.Sprintf("/api/v1/projects/%s/renders/%s/content",url.PathEscape(j.Input.ProjectID),url.PathEscape(j.ID))}; break } } }
	return p
}
func lastAttempt(j jobs.Job)(jobs.Attempt,bool){if len(j.Attempts)==0{return jobs.Attempt{},false};return j.Attempts[len(j.Attempts)-1],true}
func shotIDFromInput(in jobs.Input)string{for _,b:=range in.Bindings{if b.Role=="shot_snapshot"{parts:=strings.Split(b.ArtifactID,":");if len(parts)==4{return parts[1]}}};return ""}

func newContentHandler(coordinator *domain.Coordinator, logger *log.Logger) http.Handler {
	if logger==nil{logger=log.Default()}
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		v:=mux.Vars(r);_,data,err:=coordinator.OpenOutput(r.Context(),v["project_id"],v["render_id"])
		if err!=nil { var status int=http.StatusInternalServerError; code:=httpx.CodeInternal; msg:="render output unavailable"; if errors.Is(err,jobs.ErrNotFound){status=http.StatusNotFound;code=httpx.CodeNotFound;msg="render output not found"}; if status==http.StatusInternalServerError{logger.Printf("render output read failed: %v",err)}; httpx.WriteError(w,status,code,msg);return }
		w.Header().Set("Content-Type","image/png");w.Header().Set("Content-Length",strconv.Itoa(len(data)));w.Header().Set("X-Content-Type-Options","nosniff");w.Header().Set("Cache-Control","private, no-store");w.WriteHeader(http.StatusOK);_,_=w.Write(data)
	})
}
