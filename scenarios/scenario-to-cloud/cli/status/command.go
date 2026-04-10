package status

import (
	"github.com/vrooli/cli-core/cliutil"
)

// Run executes the status command.
func Run(client *Client, args []string) error {
	body, _, err := client.Check()
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
