"""Disposable evidence fixture; absent required outcomes must stay pending."""


def pending(required, observed):
    return [name for name in required if name in observed and not observed[name]]
