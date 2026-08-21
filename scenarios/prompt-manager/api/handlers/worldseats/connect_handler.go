package worldseats

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"connectrpc.com/connect"
	worldseatsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/worldseats"
	worldseatsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/worldseats/worldseats_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/worldseats"
)

type connectHandler struct {
	worldseatsconnect.UnimplementedWorldSeatsServiceHandler
	get http.HandlerFunc
	put http.HandlerFunc
}

func NewConnectMount(configDir string) (string, http.Handler) {
	return worldseatsconnect.NewWorldSeatsServiceHandler(&connectHandler{get: domain.HandleGet(configDir), put: domain.HandlePut(configDir)})
}

func (h *connectHandler) GetWorldSeats(ctx context.Context, req *connect.Request[worldseatsv1.GetWorldSeatsRequest]) (*connect.Response[worldseatsv1.WorldSeats], error) {
	result, err := transportbridge.Invoke(ctx, req.Header(), h.get, http.MethodGet, "/world-seats", nil, nil)
	if err != nil {
		return nil, err
	}
	out, err := decodeConfig(result.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SetWorldSeats(ctx context.Context, req *connect.Request[worldseatsv1.SetWorldSeatsRequest]) (*connect.Response[worldseatsv1.WorldSeats], error) {
	body := encodeConfig(req.Msg.GetSeats())
	result, err := transportbridge.Invoke(ctx, req.Header(), h.put, http.MethodPut, "/world-seats", body, nil)
	if err != nil {
		return nil, err
	}
	out, err := decodeConfig(result.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func encodeConfig(seats *worldseatsv1.WorldSeats) map[string]any {
	result := map[string]any{}
	for _, group := range seats.GetGroups() {
		items := make([]map[string]any, 0, len(group.GetSeats()))
		for _, seat := range group.GetSeats() {
			position := seat.GetPosition()
			items = append(items, map[string]any{"position": []float64{position.GetX(), position.GetY(), position.GetZ()}, "rotation": seat.GetRotation()})
		}
		result[group.GetFurnitureType()] = items
	}
	return result
}

func decodeConfig(raw []byte) (*worldseatsv1.WorldSeats, error) {
	var config map[string][]struct {
		Position [3]float64 `json:"position"`
		Rotation float64    `json:"rotation"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := &worldseatsv1.WorldSeats{}
	for _, key := range keys {
		group := &worldseatsv1.SeatGroup{FurnitureType: key}
		for _, seat := range config[key] {
			group.Seats = append(group.Seats, &worldseatsv1.Seat{Position: &worldseatsv1.Position{X: seat.Position[0], Y: seat.Position[1], Z: seat.Position[2]}, Rotation: seat.Rotation})
		}
		out.Groups = append(out.Groups, group)
	}
	return out, nil
}
