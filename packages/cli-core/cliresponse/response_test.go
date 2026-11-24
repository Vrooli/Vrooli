package cliresponse

import (
	"strings"
	"testing"
)

func TestDecodeUnwrapsSuccessDataAndPreservesFalseSuccessPresence(t *testing.T) {
	var got struct {
		Value string `json:"value"`
	}
	body := []byte(`{"success":false,"data":{"value":"kept"}}`)
	if err := Decode(body, &got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Value != "kept" {
		t.Fatalf("decoded value = %q, want kept", got.Value)
	}
}

func TestDecodeUsesRawObjectWhenDataFieldHasNoSuccessMarker(t *testing.T) {
	var got struct {
		Data struct {
			Literal bool `json:"literal"`
		} `json:"data"`
	}
	body := []byte(`{"data":{"literal":true}}`)
	if err := Decode(body, &got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !got.Data.Literal {
		t.Fatalf("Decode did not preserve the unwrapped response object: %#v", got)
	}
}

func TestDecodeRetainsContextSpecificErrorPrefixes(t *testing.T) {
	var dest struct {
		Count int `json:"count"`
	}
	err := Decode([]byte(`{"success":true,"data":false}`), &dest)
	if err == nil || !strings.HasPrefix(err.Error(), "parse response data:") {
		t.Fatalf("envelope decode error = %v", err)
	}
	err = Decode([]byte("not json"), &dest)
	if err == nil || !strings.HasPrefix(err.Error(), "parse response:") {
		t.Fatalf("raw decode error = %v", err)
	}
}

func TestEnvelopeMessageTrimsOnlyRecognizedEnvelopeMessages(t *testing.T) {
	if got := EnvelopeMessage([]byte(`{"success":true,"data":{"message":"  ready  "}}`)); got != "ready" {
		t.Fatalf("EnvelopeMessage(envelope) = %q", got)
	}
	if got := EnvelopeMessage([]byte(`{"message":"raw"}`)); got != "" {
		t.Fatalf("EnvelopeMessage(raw body) = %q, want empty", got)
	}
}
