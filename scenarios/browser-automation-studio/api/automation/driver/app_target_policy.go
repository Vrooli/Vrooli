package driver

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	TargetKindElectron       = "electron"
	TargetKindAndroidWebView = "android-webview"
)

// TargetURLPolicy is the admitted-navigation policy for one AppTarget kind.
// The executor asks this resolver for a policy; it does not branch on target
// platforms or invent a second target descriptor.
type TargetURLPolicy struct {
	Kind  string
	Admit func(*url.URL) bool
}

func ResolveTargetURLPolicy(kind string) (TargetURLPolicy, error) {
	switch normalizedTargetKind(kind) {
	case TargetKindElectron:
		return TargetURLPolicy{Kind: TargetKindElectron, Admit: admitDesktopRenderer}, nil
	case TargetKindAndroidWebView:
		return TargetURLPolicy{Kind: TargetKindAndroidWebView, Admit: admitWebViewRenderer}, nil
	default:
		return TargetURLPolicy{}, fmt.Errorf("unsupported app target kind %q", kind)
	}
}

func (p TargetURLPolicy) Resolve(rendererURL, route string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(rendererURL))
	if err != nil || base.Scheme == "" {
		if err == nil {
			err = fmt.Errorf("missing URL scheme")
		}
		return "", fmt.Errorf("parse admitted %s renderer URL %q: %w", p.Kind, rendererURL, err)
	}
	if !p.Admit(base) {
		return "", fmt.Errorf("%s renderer URL %q is not admitted", p.Kind, rendererURL)
	}
	if base.Scheme == "file" {
		return base.String(), nil
	}
	parsedRoute, err := url.Parse(route)
	if err != nil || parsedRoute.IsAbs() || parsedRoute.Host != "" {
		if err == nil {
			err = fmt.Errorf("scenario path must be relative")
		}
		return "", fmt.Errorf("parse %s scenario path %q: %w", p.Kind, route, err)
	}
	if parsedRoute.Path == "" {
		parsedRoute.Path = "/"
	}
	return (&url.URL{
		Scheme: base.Scheme, Host: base.Host, Path: parsedRoute.Path,
		RawPath: parsedRoute.RawPath, RawQuery: parsedRoute.RawQuery, Fragment: parsedRoute.Fragment,
	}).String(), nil
}

func normalizedTargetKind(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return TargetKindElectron
	}
	return strings.ToLower(strings.TrimSpace(kind))
}

func admitDesktopRenderer(target *url.URL) bool {
	return target.Scheme == "file" || target.Scheme == "http" || target.Scheme == "https"
}

func admitWebViewRenderer(target *url.URL) bool {
	return target.Scheme == "http" || target.Scheme == "https" || target.Scheme == "file"
}
