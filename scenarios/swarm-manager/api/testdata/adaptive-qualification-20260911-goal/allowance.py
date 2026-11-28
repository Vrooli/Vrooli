"""Disposable implementation fixture for the managed adaptive workflow proof."""


def remaining(limit, spent, reserved):
    return max(0, limit - spent - reserved)
