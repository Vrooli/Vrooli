package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name           string
		data           interface{}
		statusCode     int
		wantStatus     int
		wantBody       string
		wantJSONHeader bool
		skipBodyCheck  bool
	}{
		{
			name:           "SuccessWithMap",
			data:           map[string]string{"status": "ok"},
			statusCode:     200,
			wantStatus:     200,
			wantBody:       `{"status":"ok"}`,
			wantJSONHeader: true,
		},
		{
			name:           "ErrorWithCustomStatus",
			data:           map[string]string{"error": "not found"},
			statusCode:     404,
			wantStatus:     404,
			wantBody:       `{"error":"not found"}`,
			wantJSONHeader: true,
		},
		{
			name:           "CreatedStatus",
			data:           map[string]interface{}{"id": "123", "created": true},
			statusCode:     201,
			wantStatus:     201,
			wantBody:       `{"created":true,"id":"123"}`,
			wantJSONHeader: true,
		},
		{
			name:           "EmptyMap",
			data:           map[string]string{},
			statusCode:     200,
			wantStatus:     200,
			wantBody:       `{}`,
			wantJSONHeader: true,
		},
		{
			name:           "UnencodableData",
			data:           make(chan int), // channels cannot be JSON encoded
			statusCode:     200,
			wantStatus:     200,
			wantJSONHeader: true,
			skipBodyCheck:  true, // encoder will fail, but we can't check the body
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a ResponseRecorder to capture the response
			rr := httptest.NewRecorder()

			// Call the function
			writeJSON(rr, tt.data, tt.statusCode)

			// Check status code
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("writeJSON() status = %v, want %v", status, tt.wantStatus)
			}

			// Check Content-Type header
			contentType := rr.Header().Get("Content-Type")
			if tt.wantJSONHeader && contentType != "application/json" {
				t.Errorf("writeJSON() Content-Type = %v, want application/json", contentType)
			}

			// Check body (trim newline that json.Encoder adds)
			if !tt.skipBodyCheck {
				body := rr.Body.String()
				if len(body) > 0 && body[len(body)-1] == '\n' {
					body = body[:len(body)-1]
				}
				if body != tt.wantBody {
					t.Errorf("writeJSON() body = %v, want %v", body, tt.wantBody)
				}
			}
		})
	}
}
