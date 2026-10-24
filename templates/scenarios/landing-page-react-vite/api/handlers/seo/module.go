// Package seo is the seo domain's API contribution: the generated SeoService
// Connect handler plus the raw sitemap.xml / robots.txt endpoints (which serve
// text/xml and cannot use a generated Connect client). Business logic lives in
// internal/seo.
package seo

import (
	"errors"
	"fmt"
	"landing-page-react-vite-api/internal/module"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internalseo "landing-page-react-vite-api/internal/seo"
)

var errSlugRequired = errors.New("variant slug required")

// Module returns the seo domain's contribution: the SeoService Connect handler
// plus raw sitemap.xml / robots.txt routes.
func Module(svc *internalseo.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewSeoServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "seo",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
			r.HandleFunc("/sitemap.xml", sitemapHandler(svc)).Methods(http.MethodGet)
			r.HandleFunc("/robots.txt", robotsHandler(svc)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — seo owns no tables; overrides live on the variant row.
func Schema() string { return "" }

func sitemapHandler(svc *internalseo.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		xml, err := svc.SitemapXML(r.Context(), fmt.Sprintf("%s://%s", scheme, r.Host))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(xml))
	}
}

func robotsHandler(svc *internalseo.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(svc.RobotsTXT(r.Context())))
	}
}
