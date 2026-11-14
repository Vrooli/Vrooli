// Package peerrecord owns peer identity records and their persistence boundary.
// A record is addressed by canonical instance slug: live uses the bare scenario
// name, and non-live instances use "scenario@variant". The record payload keeps
// the bare scenario and normalized variant separately and readers validate both.
package peerrecord
