package composition

import "music-tools/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "composition-submit", Path: "/vrooli.music_tools.v1.composition.CompositionService/Submit", Method: "POST", Summary: "Submit a composition batch", Category: "composition"},
	{ID: "composition-list-takes", Path: "/vrooli.music_tools.v1.composition.CompositionService/ListTakes", Method: "POST", Summary: "List peer takes", Category: "composition"},
}
