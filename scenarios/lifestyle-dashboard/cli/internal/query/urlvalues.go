package query

import "net/url"

func ToURLValues(params map[string]string) url.Values {
	if len(params) == 0 {
		return nil
	}
	values := url.Values{}
	for key, val := range params {
		values.Set(key, val)
	}
	return values
}
