package teams

import (
	"encoding/json"
	"net/url"
	"testing"
)

type fakeContext struct {
	t           *testing.T
	gotMethod   string
	gotPath     string
	gotPayload  []byte
	response    interface{}
	getResponse interface{}
	err         error
}

func (f *fakeContext) record(method, path string, payload interface{}) error {
	f.gotMethod, f.gotPath = method, path
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		f.gotPayload = raw
	}
	return f.err
}

func (f *fakeContext) writeResult(result interface{}) error {
	if f.response == nil || result == nil {
		return nil
	}
	raw, err := json.Marshal(f.response)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

func (f *fakeContext) Get(path string, result interface{}) error {
	if err := f.record("GET", path, nil); err != nil {
		return err
	}
	if f.getResponse != nil && result != nil {
		raw, err := json.Marshal(f.getResponse)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, result)
	}
	return f.writeResult(result)
}

func (f *fakeContext) GetWithQuery(path string, query url.Values, result interface{}) error {
	if query != nil {
		path += "?" + query.Encode()
	}
	return f.Get(path, result)
}

func (f *fakeContext) Post(path string, payload, result interface{}) error {
	if err := f.record("POST", path, payload); err != nil {
		return err
	}
	return f.writeResult(result)
}

func (f *fakeContext) Put(path string, payload, result interface{}) error {
	if err := f.record("PUT", path, payload); err != nil {
		return err
	}
	return f.writeResult(result)
}

func (f *fakeContext) Delete(path string) error { return f.record("DELETE", path, nil) }

func (f *fakeContext) DeleteWithQuery(path string, _ url.Values, result interface{}) error {
	if err := f.record("DELETE", path, nil); err != nil {
		return err
	}
	return f.writeResult(result)
}

func (f *fakeContext) assertMethodPath(t *testing.T, method, path string) {
	t.Helper()
	if f.gotMethod != method {
		t.Errorf("method = %q, want %q", f.gotMethod, method)
	}
	if f.gotPath != path {
		t.Errorf("path = %q, want %q", f.gotPath, path)
	}
}
