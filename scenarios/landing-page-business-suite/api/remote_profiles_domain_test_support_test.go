package main

import "landing-page-business-suite-api/internal/administration"

type (
	RemoteProfile                = administration.RemoteProfile
	RemoteProfileCreateRequest   = administration.RemoteProfileCreateRequest
	RemoteProfileUpdateRequest   = administration.RemoteProfileUpdateRequest
	RemoteProfileLoginRequest    = administration.RemoteProfileLoginRequest
	RemoteProfileProxyRequest    = administration.RemoteProfileProxyRequest
	RemoteProxyResponse          = administration.RemoteProxyResponse
	IncomingRemoteProfileSession = administration.IncomingRemoteProfileSession
	RemoteProfileSessionLinks    = administration.RemoteProfileSessionLinks
	RemoteProfileError           = administration.RemoteProfileError
)

var (
	ErrRemoteProfileNotFound             = administration.ErrRemoteProfileNotFound
	ErrRemoteProfileTagExists            = administration.ErrRemoteProfileTagExists
	ErrRemoteProfileInvalid              = administration.ErrRemoteProfileInvalid
	ErrRemoteProfileSessionMissing       = administration.ErrRemoteProfileSessionMissing
	ErrRemoteProfileDisallowedPath       = administration.ErrRemoteProfileDisallowedPath
	ErrRemoteProfileServiceSecretMissing = administration.ErrRemoteProfileServiceSecretMissing
	remoteProfileCookieName              = administration.RemoteProfileCookieName
	remoteProfileStatusUnknown           = administration.RemoteProfileStatusUnknown
	remoteProfileStatusActive            = administration.RemoteProfileStatusActive
	remoteProfileStatusExpired           = administration.RemoteProfileStatusExpired
	remoteProfileStatusError             = administration.RemoteProfileStatusError
	normalizeRemoteProfileTag            = administration.NormalizeRemoteProfileTag
	normalizeRemoteProfileLabel          = administration.NormalizeRemoteProfileLabel
	normalizeRemoteProfileAPIBase        = administration.NormalizeRemoteProfileAPIBase
	normalizeRemoteProxyPath             = administration.NormalizeRemoteProxyPath
	readLimitedBody                      = administration.ReadLimitedBody
	classifyRemoteError                  = administration.ClassifyRemoteError
	extractRemoteErrorMessage            = administration.ExtractRemoteErrorMessage
	mapRemoteStatus                      = administration.MapRemoteStatus
	isAllowedRemoteProxyPath             = administration.IsAllowedRemoteProxyPath
)
