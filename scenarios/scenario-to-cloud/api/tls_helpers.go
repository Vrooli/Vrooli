package main

import (
	"fmt"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tlsinfo"
)

func buildTLSInfoResponse(domainName string, snapshot tlsinfo.Snapshot, err error) TLSInfoResponse {
	resp := TLSInfoResponse{
		Domain:     domainName,
		Validation: defaultValidation(snapshot.Probe.Validation),
		ALPN:       &snapshot.ALPN,
	}
	if err != nil {
		resp.OK = false
		resp.Valid = false
		resp.Error = fmt.Sprintf("TLS probe failed: %v", err)
		return resp
	}

	resp.Valid = snapshot.Probe.Valid
	resp.Issuer = snapshot.Probe.Issuer
	resp.Subject = snapshot.Probe.Subject
	resp.NotBefore = snapshot.Probe.NotBefore
	resp.NotAfter = snapshot.Probe.NotAfter
	resp.DaysRemaining = snapshot.Probe.DaysRemaining
	resp.SerialNumber = snapshot.Probe.SerialNumber
	resp.SANs = snapshot.Probe.SANs
	if snapshot.Probe.ValidationError != "" {
		resp.Error = snapshot.Probe.ValidationError
	}
	resp.OK = true
	return resp
}

func buildDomainTLSInfo(snapshot tlsinfo.Snapshot, err error) domain.TLSInfo {
	info := domain.TLSInfo{
		Validation: defaultValidation(snapshot.Probe.Validation),
		ALPN:       convertALPN(snapshot.ALPN),
	}
	if err != nil {
		info.Valid = false
		info.Error = fmt.Sprintf("TLS probe failed: %v", err)
		return info
	}

	info.Valid = snapshot.Probe.Valid
	info.Issuer = snapshot.Probe.Issuer
	info.Expires = snapshot.Probe.NotAfter
	info.DaysRemaining = snapshot.Probe.DaysRemaining
	if snapshot.Probe.ValidationError != "" {
		info.Error = snapshot.Probe.ValidationError
	}
	return info
}

func convertALPN(check tlsinfo.ALPNCheck) *domain.ALPNCheck {
	return &domain.ALPNCheck{
		Status:   string(check.Status),
		Message:  check.Message,
		Hint:     check.Hint,
		Protocol: check.Protocol,
		Error:    check.Error,
	}
}

func defaultValidation(value string) string {
	if strings.TrimSpace(value) == "" {
		return "time_only"
	}
	return value
}
