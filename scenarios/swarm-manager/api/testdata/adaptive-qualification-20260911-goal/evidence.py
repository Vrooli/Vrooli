"""Disposable evidence fixture; absent required outcomes must stay pending."""


def pending(required, observed):
    return [name for name in required if not observed.get(name, False)]
