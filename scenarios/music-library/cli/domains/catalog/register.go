package catalog

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/catalog"
	svc "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/catalog/catalog_v1connect"
)

func verifySource(path, expected string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("source path must be absolute")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	if fmt.Sprintf("%x", h.Sum(nil)) != expected {
		return fmt.Errorf("source SHA-256 does not match")
	}
	return nil
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	http, base := cliapp.NewConnectHTTPClient(core)
	c := svc.NewCatalogServiceClient(http, base)
	put := func(x cliapp.OperationContext) (*pb.PutTrackResponse, error) {
		if err := verifySource(x.Flag("path"), x.Flag("sha256")); err != nil {
			return nil, err
		}
		tags := []string{}
		for _, t := range strings.Split(x.Flag("tags"), ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		r, e := c.PutTrack(context.Background(), connect.NewRequest(&pb.PutTrackRequest{Track: &pb.Track{Sha256: x.Flag("sha256"), Title: x.Flag("title"), SourcePath: x.Flag("path"), TakeId: x.Flag("take"), ReceiptPath: x.Flag("receipt"), Tags: tags, Approval: x.Flag("approval"), Feedback: x.Flag("feedback")}}))
		if e != nil {
			return nil, e
		}
		return r.Msg, nil
	}
	putReport := func(_ cliapp.OperationContext, r *pb.PutTrackResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Saved %s (%s)", r.Track.Title, r.Track.Sha256)}}
	}
	list := func(x cliapp.OperationContext) (*pb.ListTracksResponse, error) {
		unused := false
		if value := x.Flag("unused"); value != "" {
			var err error
			unused, err = strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("invalid --unused: %w", err)
			}
		}
		r, e := c.ListTracks(context.Background(), connect.NewRequest(&pb.ListTracksRequest{Tag: x.Flag("tag"), UnusedOnly: unused, Project: x.Flag("project")}))
		if e != nil {
			return nil, e
		}
		approval := x.Flag("approval")
		if approval != "" && approval != "liked" && approval != "unreviewed" && approval != "rejected" {
			return nil, fmt.Errorf("invalid --approval: use liked, unreviewed or rejected")
		}
		if approval != "" {
			filtered := make([]*pb.Track, 0, len(r.Msg.Tracks))
			for _, track := range r.Msg.Tracks {
				if track.Approval == approval {
					filtered = append(filtered, track)
				}
			}
			r.Msg.Tracks = filtered
		}
		return r.Msg, nil
	}
	listReport := func(_ cliapp.OperationContext, r *pb.ListTracksResponse) cliapp.ListReport {
		lines := []string{}
		for _, t := range r.Tracks {
			lines = append(lines, fmt.Sprintf("%s — %s [%s] tags=%s uses=%d", t.Sha256, t.Title, t.Approval, strings.Join(t.Tags, ","), len(t.Uses)))
		}
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d tracks.", len(lines))}, ResultsHeading: "Tracks", Results: lines}
	}
	use := func(x cliapp.OperationContext) (*pb.RecordUseResponse, error) {
		r, e := c.RecordUse(context.Background(), connect.NewRequest(&pb.RecordUseRequest{Sha256: x.Flag("sha256"), Use: &pb.TrackUse{Project: x.Flag("project"), Revision: x.Flag("revision"), Status: x.Flag("status")}}))
		if e != nil {
			return nil, e
		}
		return r.Msg, nil
	}
	useReport := func(_ cliapp.OperationContext, r *pb.RecordUseResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded use for %s.", r.Track.Title)}}
	}
	return cliapp.LoadFromManifestPrimitives(manifest, "catalog", map[string]cliapp.PrimitiveHandler{"CatalogService.PutTrack": cliapp.ProtoMutation(put, putReport), "CatalogService.ListTracks": cliapp.ProtoList(list, listReport), "CatalogService.RecordUse": cliapp.ProtoMutation(use, useReport)})
}
