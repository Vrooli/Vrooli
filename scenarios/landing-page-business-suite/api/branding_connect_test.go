package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

func newBrandingConnectTestStore(t *testing.T) *ConfigStore {
	t.Helper()
	brandingPath := filepath.Join(t.TempDir(), "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{"site_name":"Test Site"}`), 0o644); err != nil {
		t.Fatalf("write branding fixture: %v", err)
	}
	store := NewConfigStore("", brandingPath, nil)
	if err := store.LoadAll(); err != nil {
		t.Fatalf("load branding fixture: %v", err)
	}
	return store
}

func TestBrandingConnectPreservesExpandedBrandingFields(t *testing.T) {
	store := newBrandingConnectTestStore(t)
	handler := brandingConnectHandler{store: store}
	smtpHost, smtpPassword, supportChatURL, comingSoon := "smtp.example.test", "not-public", "https://support.example.test", true
	updated, err := handler.UpdateBranding(context.Background(), connect.NewRequest(&lpbsv1.UpdateBrandingRequest{
		SmtpHost:          &smtpHost,
		SmtpPassword:      &smtpPassword,
		SupportChatUrl:    &supportChatURL,
		ComingSoonEnabled: &comingSoon,
	}))
	if err != nil {
		t.Fatalf("UpdateBranding() error = %v", err)
	}
	if updated.Msg.GetBranding().GetSmtpHost() != smtpHost || updated.Msg.GetBranding().GetSmtpPassword() != smtpPassword || !updated.Msg.GetBranding().GetComingSoonEnabled() {
		t.Fatalf("expanded fields not preserved: %#v", updated.Msg.GetBranding())
	}
}

func TestBrandingConnectPublicResponseRedactsSMTPPassword(t *testing.T) {
	store := newBrandingConnectTestStore(t)
	handler := brandingConnectHandler{store: store}
	password, supportChatURL, comingSoonMessage := "not-public", "https://support.example.test", "Back shortly"
	if _, err := handler.UpdateBranding(context.Background(), connect.NewRequest(&lpbsv1.UpdateBrandingRequest{
		SmtpPassword:      &password,
		SupportChatUrl:    &supportChatURL,
		ComingSoonMessage: &comingSoonMessage,
	})); err != nil {
		t.Fatalf("seed SMTP password: %v", err)
	}
	response, err := handler.GetPublicBranding(context.Background(), connect.NewRequest(&lpbsv1.GetPublicBrandingRequest{}))
	if err != nil {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	if response.Msg.GetBranding().GetSupportChatUrl() != supportChatURL || response.Msg.GetBranding().GetComingSoonMessage() != comingSoonMessage {
		t.Fatalf("public fields = %#v", response.Msg.GetBranding())
	}
	if response.Msg.GetBranding().ProtoReflect().Descriptor().Fields().ByName("smtp_password") != nil {
		t.Fatal("public branding schema exposes SMTP password")
	}
}

func TestBrandingConnectRejectsEmptyClearField(t *testing.T) {
	store := newBrandingConnectTestStore(t)
	handler := brandingConnectHandler{store: store}
	_, err := handler.ClearBrandingField(context.Background(), connect.NewRequest(&lpbsv1.ClearBrandingFieldRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want %v", got, connect.CodeInvalidArgument)
	}
}

func TestBrandingConnectRoutesProtectAdminProceduresOnly(t *testing.T) {
	router := mux.NewRouter()
	store := newBrandingConnectTestStore(t)
	registerBrandingConnectRoutes(router, store, func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusUnauthorized)
		}
	})

	adminRequest := httptest.NewRequest(http.MethodPost, lpbsconnect.BrandingServiceGetBrandingProcedure, nil)
	adminResponse := httptest.NewRecorder()
	router.ServeHTTP(adminResponse, adminRequest)
	if adminResponse.Code != http.StatusUnauthorized {
		t.Fatalf("admin procedure status = %d, want %d", adminResponse.Code, http.StatusUnauthorized)
	}

	publicRequest := httptest.NewRequest(http.MethodPost, lpbsconnect.BrandingServiceGetPublicBrandingProcedure, strings.NewReader(`{}`))
	publicRequest.Header.Set("Content-Type", "application/json")
	publicResponse := httptest.NewRecorder()
	router.ServeHTTP(publicResponse, publicRequest)
	if publicResponse.Code != http.StatusOK {
		t.Fatalf("public procedure status = %d, want %d: %s", publicResponse.Code, http.StatusOK, publicResponse.Body.String())
	}
}
