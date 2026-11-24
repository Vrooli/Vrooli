package delivery

import "testing"

func TestValidateStorefrontLink(t *testing.T) {
	tests := []struct {
		name    string
		store   string
		url     string
		wantErr bool
	}{
		{name: "regional Apple listing", store: "app_store", url: "https://apps.apple.com/us/app/aquila/id123456"},
		{name: "Apple listing without regional prefix", store: "app_store", url: "https://apps.apple.com/app/id123456"},
		{name: "legacy Apple listing host", store: "app_store", url: "https://itunes.apple.com/us/app/aquila/id123456"},
		{name: "Apple non-listing path", store: "app_store", url: "https://apps.apple.com/us/app/aquila", wantErr: true},
		{name: "Apple id must be numeric", store: "app_store", url: "https://apps.apple.com/us/app/aquila/id-app", wantErr: true},
		{name: "encoded Apple id rejected", store: "app_store", url: "https://apps.apple.com/us/app/aquila/id%31%32%33", wantErr: true},
		{name: "Apple path must not have extra segments", store: "app_store", url: "https://apps.apple.com/us/app/aquila/extra/id123456", wantErr: true},
		{name: "Google Play listing", store: "play_store", url: "https://play.google.com/store/apps/details?id=org.vrooli.aquila"},
		{name: "HTTP rejected", store: "app_store", url: "http://apps.apple.com/us/app/aquila/id123456", wantErr: true},
		{name: "wrong official store", store: "app_store", url: "https://play.google.com/store/apps/details?id=org.vrooli.aquila", wantErr: true},
		{name: "deceptive suffix rejected", store: "play_store", url: "https://play.google.com.evil.test/store/apps/details?id=org.vrooli.aquila", wantErr: true},
		{name: "userinfo rejected", store: "play_store", url: "https://user@play.google.com/store/apps/details?id=org.vrooli.aquila", wantErr: true},
		{name: "empty userinfo rejected", store: "play_store", url: "https://@play.google.com/store/apps/details?id=org.vrooli.aquila", wantErr: true},
		{name: "empty fragment marker rejected", store: "play_store", url: "https://play.google.com/store/apps/details?id=org.vrooli.aquila#", wantErr: true},
		{name: "wrong listing path", store: "play_store", url: "https://play.google.com/about/", wantErr: true},
		{name: "missing Google package id", store: "play_store", url: "https://play.google.com/store/apps/details", wantErr: true},
		{name: "unknown store rejected", store: "custom", url: "https://example.com/app", wantErr: true},
		{name: "relative URL rejected", store: "app_store", url: "/us/app/aquila/id123456", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStorefrontLink(tt.store, tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateStorefrontLink(%q, %q) error = %v, wantErr %v", tt.store, tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeStorefrontLinksTrimsOwnerFieldsBeforePersistence(t *testing.T) {
	got, err := NormalizeStorefrontLinks([]Storefront{{
		Store: " play_store ", Label: " Google Play ",
		URL: " https://play.google.com/store/apps/details?id=org.vrooli.aquila ", Badge: " Open ",
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := Storefront{
		Store: "play_store", Label: "Google Play",
		URL: "https://play.google.com/store/apps/details?id=org.vrooli.aquila", Badge: "Open",
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("normalized storefront = %+v, want %+v", got, want)
	}
}
