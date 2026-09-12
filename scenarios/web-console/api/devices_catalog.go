package main

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	deviceshandler "web-console/handlers/devices"
	"web-console/session"
)

type deviceRPCService struct{ server *Server }

func (s *Server) mountDevices() {
	deviceshandler.Module(&deviceRPCService{server: s}).Mount(s.router)
}

func (d *deviceRPCService) List(_ context.Context, selfDeviceID string) ([]deviceshandler.Device, error) {
	if d == nil || d.server == nil || d.server.sessions == nil {
		return nil, errors.New("session manager is unavailable")
	}
	rows := d.server.sessions.ConnectedDevices()
	out := make([]deviceshandler.Device, 0, len(rows))
	for _, row := range rows {
		sessions := make([]deviceshandler.SessionAttachment, 0, len(row.Sessions))
		for _, attachment := range row.Sessions {
			sessions = append(sessions, deviceshandler.SessionAttachment{SessionID: attachment.SessionID, SessionName: attachment.SessionName, HoldsLease: attachment.HoldsLease})
		}
		out = append(out, deviceshandler.Device{
			ID: row.DeviceID, Label: row.DeviceLabel, Class: row.DeviceClass,
			ConnectionCount: row.ConnCount, FirstSeenUnix: row.FirstSeenAt.Unix(), Sessions: sessions,
			IsSelf:       strings.TrimSpace(selfDeviceID) != "" && selfDeviceID == row.DeviceID,
			Reconnecting: row.Reconnecting,
		})
	}
	return out, nil
}

func (d *deviceRPCService) Disconnect(_ context.Context, deviceID, connectionID string) (int, error) {
	if d == nil || d.server == nil || d.server.sessions == nil {
		return 0, errors.New("session manager is unavailable")
	}
	deviceID = strings.TrimSpace(deviceID)
	connectionID = strings.TrimSpace(connectionID)
	if deviceID == "" {
		return 0, connect.NewError(connect.CodeInvalidArgument, errors.New("a device id is required"))
	}
	closed := 0
	for _, sess := range d.server.sessions.List() {
		for _, view := range sess.ConnectedDevices() {
			if view.DeviceID != deviceID {
				continue
			}
			for _, conn := range view.Connections {
				if connectionID != "" && conn.ConnID != connectionID {
					continue
				}
				if channel := sess.ClientChannel(conn.ConnID); channel != nil {
					sess.Supersede(channel)
					closed++
				}
			}
		}
	}
	return closed, nil
}

func (d *deviceRPCService) GiveControl(_ context.Context, deviceID, sessionID string) (bool, error) {
	if d == nil || d.server == nil || d.server.sessions == nil {
		return false, errors.New("session manager is unavailable")
	}
	deviceID = strings.TrimSpace(deviceID)
	sessionID = strings.TrimSpace(sessionID)
	if deviceID == "" {
		return false, connect.NewError(connect.CodeInvalidArgument, errors.New("a device id is required"))
	}
	for _, sess := range d.server.sessions.List() {
		if sessionID != "" && sess.ID != sessionID {
			continue
		}
		for _, view := range sess.ConnectedDevices() {
			if view.DeviceID != deviceID {
				continue
			}
			for _, conn := range view.Connections {
				if channel := sess.ClientChannel(conn.ConnID); channel != nil {
					return true, sess.AcquireLease(channel, session.LeaseReasonExplicit)
				}
			}
		}
	}
	return false, nil
}
