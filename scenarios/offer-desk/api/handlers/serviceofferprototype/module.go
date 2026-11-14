// Package serviceofferprototype exposes the reviewable, read-only PS2 offer
// contract prototype. It demonstrates one public-data service projected into
// human and agent doors without creating an order or causing an external side
// effect. The production extension remains subject to the Offer Desk proposal
// and approval flow.
package serviceofferprototype

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"offer-desk/internal/module"
	"offer-desk/internal/serviceoffer"

	"github.com/gorilla/mux"
)

const previewPath = "/api/v1/proposal-preview/service-offer"

type previewResponse struct {
	Offer    serviceoffer.Record   `json:"offer"`
	Surfaces serviceoffer.Surfaces `json:"surfaces"`
	Sample   string                `json:"sample"`
}

type handler struct {
	response previewResponse
	err      error
}

func Module() module.Module {
	h := handler{}
	record, files, err := serviceoffer.Prototype()
	if err != nil {
		h.err = err
	} else {
		var surfaces serviceoffer.Surfaces
		surfaces, h.err = serviceoffer.Project(record, files)
		if h.err == nil {
			var sample []byte
			sample, h.err = fs.ReadFile(files, record.Sample.Path)
			if h.err == nil {
				h.response = previewResponse{Offer: record, Surfaces: surfaces, Sample: string(sample)}
			}
		}
	}
	return module.Module{
		Name: "service-offer-prototype",
		Mount: func(r *mux.Router) {
			r.HandleFunc(previewPath, h.preview).Methods(http.MethodGet)
			r.HandleFunc(previewPath+"/sample", h.sample).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

func (h handler) preview(writer http.ResponseWriter, _ *http.Request) {
	if h.err != nil {
		http.Error(writer, "service offer prototype unavailable", http.StatusServiceUnavailable)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(writer).Encode(h.response)
}

func (h handler) sample(writer http.ResponseWriter, _ *http.Request) {
	if h.err != nil {
		http.Error(writer, "service offer prototype unavailable", http.StatusServiceUnavailable)
		return
	}
	writer.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	_, _ = writer.Write([]byte(h.response.Sample))
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "service_offer_prototype", Path: previewPath, Method: http.MethodGet, Summary: "Preview a single staged service offer and its derived purchase/discovery door contracts", Category: "proposal", RESTException: proposalPreviewRESTException},
	{ID: "service_offer_sample", Path: previewPath + "/sample", Method: http.MethodGet, Summary: "Read the public-data sample attached to the proposed service offer", Category: "proposal", RESTException: proposalPreviewRESTException},
}

var proposalPreviewRESTException = &module.RESTException{
	Reason: module.RESTReasonOpsProbe,
	Note:   "Read-only, local operator preview for a proposed Offer Desk extension; it does not create an order, publish content, or authorize an external effect.",
}
