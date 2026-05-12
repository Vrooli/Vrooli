package components

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"react-component-library/internal/components"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
)

// domainToProto converts an internal components.Component into the wire
// shape the components proto declares. Lives in the handler package by
// intent — the conversion is mechanical and only used at the transport
// edge; pulling it into a separate adapters package would create a
// one-import wrapper for no gain.
func domainToProto(c components.Component) *componentsv1.Component {
	tags := append([]string(nil), c.Tags...)
	if tags == nil {
		tags = []string{}
	}
	headers := make(map[string]string, len(c.Headers))
	for k, v := range c.Headers {
		headers[k] = v
	}
	return &componentsv1.Component{
		Id:          c.ID,
		LibraryId:   c.LibraryID,
		DisplayName: c.DisplayName,
		Description: c.Description,
		SourcePath:  c.SourcePath,
		Version:     c.Version,
		Tags:        tags,
		IndexedAt:   timestamppb.New(c.IndexedAt.UTC()),
		UpdatedAt:   timestamppb.New(c.UpdatedAt.UTC()),
		Headers:     headers,
	}
}
