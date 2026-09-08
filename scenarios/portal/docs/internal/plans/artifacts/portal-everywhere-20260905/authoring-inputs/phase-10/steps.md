1. Retain X11 as a named backend while replacing weak readiness probes.
2. Implement AT-SPI semantic observation and actions.
3. Implement X11 capture and complete input support.
4. Implement portal-based Wayland capture and authorized input.
5. Evaluate libei through governed dependency selection.
6. Declare GNOME and KDE backend capabilities separately.
7. Report missing portal interfaces as unsupported or unavailable.
8. Handle portal session revocation and reconnect.
9. Validate scale, display origin, keyboard layout, and Unicode.
10. Prevent fallback to X11 when the active Wayland session is incompatible.
