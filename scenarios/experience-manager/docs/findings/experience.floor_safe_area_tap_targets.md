# experience.floor_safe_area_tap_targets

The inherited safe-area-tap-targets floor failed. A mobile interactive control intersects the unsafe bottom device-edge zone, where home indicators and rounded screen edges can make it hard or impossible to tap.

Add environment-aware safe-area padding such as `env(safe-area-inset-bottom)` to fixed bottom controls, or move the target fully into the safe viewport.
