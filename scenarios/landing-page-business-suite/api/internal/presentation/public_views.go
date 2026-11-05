package presentation

import "errors"

// PublicViews enumerates the canonical, unprivileged route/locale projections
// for publication and crawler consumers. It is not a second membership filter:
// every app route passes through Resolve, including enabled non-members and
// private profiles, whose not-found results are deliberately excluded.
func PublicViews(document Document) ([]ResolveResult, error) {
	if err := Validate(document); err != nil {
		return nil, err
	}
	routes := []string{"/"}
	for _, app := range document.Apps {
		routes = append(routes, "/apps/"+app.Slug)
	}
	views := make([]ResolveResult, 0)
	for _, route := range routes {
		for _, locale := range document.Bundle.Locales {
			view, err := Resolve(document, ResolveRequest{Route: route, Locale: locale})
			if errors.Is(err, ErrNotFound) && route != "/" {
				continue
			}
			if err != nil {
				return nil, err
			}
			views = append(views, view)
		}
	}
	return views, nil
}
