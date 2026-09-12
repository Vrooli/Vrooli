package channels

import "fmt"

func ValidateOutbound(d Descriptor, out Outbound) error {
	if d.Limits.MaxTextBytes > 0 && int64(len(out.Text)) > d.Limits.MaxTextBytes {
		return fmt.Errorf("channel %s rejects text: limit maxTextBytes=%d", d.ID, d.Limits.MaxTextBytes)
	}
	for _, media := range out.Media {
		if d.Limits.MaxMediaBytes > 0 && media.Size > d.Limits.MaxMediaBytes {
			return fmt.Errorf("channel %s rejects media %q: limit maxMediaBytes=%d", d.ID, media.Name, d.Limits.MaxMediaBytes)
		}
	}
	return nil
}
