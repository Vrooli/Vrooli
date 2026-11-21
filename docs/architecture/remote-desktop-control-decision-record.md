# Remote Desktop Control Decision Record

This record preserves the design conversation that led to the remote desktop
implementation plan. It is an input artifact for future agents; it is not a
replacement for the canonical architecture or owner PRDs.

**Captured:** 2026-09-18  
**Target:** minimouse, Bridge node `25c7e426-c76c-421a-8351-aaf964589802`  
**Operator:** matthalloran8  
**Canonical design:** [`remote-desktop-control.md`](remote-desktop-control.md)

## Starting problem

The original question was why minimouse did not have an active GUI session and
whether the Bridge work was intended to make the GUI reachable. The control
plane successfully onboarded minimouse, refreshed the current node-agent, and
proved host inventory and remote screen-sharing service presence. The live
inventory still reported no active Aqua session.

The first graphical-session safeguard accepted a successful automatic-login
command without validating the macOS postcondition. That was corrected to fail
closed. The corrected run observed:

```json
{
  "active_session_user": "",
  "session_type": "",
  "display_attached": true,
  "auto_login_user": "matthalloran8",
  "macos_auto_login_status": "disabled"
}
```

Therefore the blocker was not Bridge delivery, pairing, or node reachability.
It was the absence of a usable authenticated desktop session, compounded by
macOS security boundaries such as FileVault and privacy permissions.

## Original ownership discussion

The conversation considered four possible owners:

| Candidate | Discussion outcome |
|---|---|
| `vrooli-bridge` | Owns identity, pairing, trust, reach, transport, revocation, and audit; should not own every desktop protocol or native OS API. |
| `device-control` | Already owns device operation, host-desktop strategy, desktop sessions, leases, input, observation, and receipts. Selected owner of desktop semantics. |
| `web-console` | Owns the operator workspace and terminal experience; selected owner of the desktop pane, not the native backend. |
| New `remote-desktop` scenario | Rejected for the first release because it would duplicate Device Control’s existing desktop authority. It may become appropriate later only if streaming becomes an independent product with multiple non-Device-Control consumers. |

The repository confirms that Device Control already declares portable desktop
user-session control and destination authority in its PRD and requirements.
Bridge already contains planned typed desktop transport requirements. The design
was therefore corrected from “new remote-desktop scenario” to “Device Control
desktop capability transported through Bridge and rendered by Web Console.”

## Operator decisions

The operator selected or accepted the following:

1. **Ownership:** Device Control owns desktop sessions and input.
2. **Transport:** custom WebRTC rather than VNC/RFB or RDP-first.
3. **Interaction:** both raw human input and semantic Device Control actions.
4. **Media routing:** professional long-term hybrid—direct WebRTC when possible,
   TURN relay fallback, with Bridge authorizing/signaling the session.
5. **Concurrency:** one controller plus read-only viewers.
6. **Target rollout:** minimouse/macOS first, then Linux and Windows.
7. **Clipboard:** included in the first release.
8. **Operator UI:** a Desktop session/pane in Web Console.
9. **Companion packaging:** Device Control-owned companion managed through Bridge
   onboarding; this remains the recommended choice pending final plan execution.
10. **Codec:** VP8 first, with later platform-specific codec optimization.
11. **Capture:** native platform capture, ScreenCaptureKit on macOS.
12. **WebRTC implementation:** Pion in the Go companion.
13. **Display scope:** one selected display first.
14. **macOS permissions:** one-time local approval for minimouse, with managed
    fleet policy support later; Bridge must report permission gaps honestly.

## Goals

- Operate minimouse through Bridge/Web Console without routine direct SSH or
  physical remote login.
- Provide a real screen stream and actual keyboard/mouse input.
- Keep trust/reach, desktop operation, transport, and UI ownership distinct.
- Preserve Device Control’s leases, epochs, geometry revisions, receipts, and
  fail-closed behavior.
- Support explicit text clipboard transfer without ambient synchronization.
- Make no-session, locked-session, FileVault-preboot, permission-denied, and
  helper-crash states visible and actionable.
- Validate the complete route on real minimouse hardware before claiming support.

## Non-goals for the first release

- Universal preboot/FileVault unlock through the ordinary node agent.
- Arbitrary remote shell disguised as desktop control.
- Multi-controller input arbitration.
- File transfer or binary clipboard synchronization.
- Multi-display composition.
- Always-on screen recording.
- Replacing Device Control’s semantic flow engine with raw video input.
- Making Web Console the owner of native desktop adapters.

## Research anchors

The design was checked against the following primary or owner documents:

- [Apple ScreenCaptureKit](https://developer.apple.com/videos/play/wwdc2022/10156/)
- [Apple `CGEventPost`](https://developer.apple.com/documentation/coregraphics/cgevent/post%28tap%3A%29?language=objc)
- [Apple Privacy Preferences Policy Control](https://developer.apple.com/documentation/devicemanagement/privacypreferencespolicycontrol/services-data.dictionary?changes=_2.)
- [Apple deployment update for macOS 26.2/27 Accessibility policy](https://support.apple.com/en-au/guide/deployment/depd567c9ffa/web)
- [W3C WebRTC specification](https://www.w3.org/TR/webrtc/)
- [WebRTC data channels](https://webrtc.org/getting-started/data-channels)
- [WebRTC connectivity and TURN relay candidates](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Connectivity)
- [Bridge PRD](../../scenarios/vrooli-bridge/PRD.md)
- [Device Control PRD](../../scenarios/device-control/PRD.md)
- [Device Control desktop proto](../../packages/proto/schemas/device-control/v1/desktop/desktop.proto)
- [Scenario-to-desktop interactive desktop guide](../../scenarios/scenario-to-desktop/docs/guides/interactive-desktop.md)

## Existing evidence artifacts

The prior minimouse setup and fail-closed safeguard work are recorded in Vrooli
Memory work record `073938dc-f924-4ecd-ae36-24b48b97ed30` (the exact record ID
should be rechecked before plan execution if the memory store has normalized it).
The active related implementation plan is:

`/home/matthalloran8/.vrooli/plans/cross-machine-desktop-validation-route-mac-and-windows.md`

The large related Device Control/Portal plan is:

`/home/matthalloran8/.vrooli/plans/portal-everywhere-native-companion-portable-desktop-control.md`

Those plans are related context, not proof that this design is implemented.
Their phase evidence and current source must be reviewed before execution.
