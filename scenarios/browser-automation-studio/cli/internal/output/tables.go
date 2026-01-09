package output

import (
	"fmt"
	"io"
	"text/tabwriter"
)

type PlaybookOrderRow struct {
	Order        string `json:"order"`
	Reset        string `json:"reset"`
	Requirements int    `json:"requirements"`
	File         string `json:"file"`
	Description  string `json:"description,omitempty"`
}

func RenderPlaybookOrder(w io.Writer, rows []PlaybookOrderRow) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ORDER\tRESET\tREQ\tPLAYBOOK")
	for _, row := range rows {
		line := fmt.Sprintf("%s\t%s\t%d\t%s", row.Order, row.Reset, row.Requirements, row.File)
		if row.Description != "" {
			line += " - " + row.Description
		}
		fmt.Fprintln(tw, line)
	}
	_ = tw.Flush()
}
