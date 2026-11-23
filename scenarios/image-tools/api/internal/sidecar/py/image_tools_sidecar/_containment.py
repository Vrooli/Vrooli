"""Model-download containment for every image-tools Python process.

Inference never downloads. Weights are acquired only by the governed install
step, which writes into the managed model root and runs with
IMAGE_TOOLS_SIDECAR_NETWORK=governed-acquisition. Every other process the Go
server starts carries IMAGE_TOOLS_SIDECAR_NETWORK=blocked (see
internal/modelenv/pins.json), and this module then refuses every connection that
leaves the machine. That holds for libraries that are not used yet: a package
that tries to fetch weights (insightface, rembg, torch.hub, timm, an old
huggingface_hub code path, a plain urllib call) fails immediately with the
message below instead of silently filling a cache image-tools does not manage.

The hub cache and home variables are pinned by the Go side; this module is the
backstop for code that ignores them.
"""

from __future__ import annotations

import ipaddress
import os
import socket
import sys

ENV_MODE = "IMAGE_TOOLS_SIDECAR_NETWORK"
BLOCKED = "blocked"

_installed = False


class ModelDownloadBlocked(OSError):
    """A library tried to reach the network during inference."""


def _message(target: str) -> str:
    return (
        f"image-tools blocked a network connection to {target}: inference may not download "
        "model files. Weights are acquired only by the governed install step "
        "(`image-tools models install <id>` or job-time auto-acquire), which writes into the managed "
        "model root. If a library needs a file that is missing, declare it in the model's catalog "
        "entry (source.assets / source.repo / source.hub_companions) so it is acquired and receipted; "
        "if it needs a cache directory, add its variable to internal/modelenv/pins.json."
    )


def _own_names():
    """This machine's own hostname: resolving it reaches no remote host, and
    libraries that size a local server or pick a free port ask for it."""
    global _OWN
    if _OWN is None:
        names = set()
        try:
            name = socket.gethostname()
            names.update({name, name.split(".")[0], socket.getfqdn(name)})
        except OSError:
            pass
        _OWN = names
    return _OWN


_OWN = None


def _is_local_host(host) -> bool:
    if host is None:
        return True
    if isinstance(host, bytes):
        host = host.decode("ascii", "ignore")
    host = str(host)
    if host in ("", "localhost", "localhost.localdomain", "ip6-localhost", "0.0.0.0", "::"):
        return True
    if host in _own_names():
        return True
    try:
        return ipaddress.ip_address(host.split("%", 1)[0]).is_loopback
    except ValueError:
        return False


def _is_local_address(address) -> bool:
    if isinstance(address, (str, bytes)):  # AF_UNIX path
        return True
    if isinstance(address, tuple) and address:
        return _is_local_host(address[0])
    return False


def _refuse(target: str):
    text = _message(target)
    try:
        print(f"image-tools: {text}", file=sys.stderr, flush=True)
    except Exception:  # noqa: BLE001 - never mask the refusal itself
        pass
    raise ModelDownloadBlocked(text)


def install(force: bool = False) -> bool:
    """Install the guard when this process is an inference process. Idempotent."""
    global _installed
    if _installed:
        return True
    if not force and os.environ.get(ENV_MODE) != BLOCKED:
        return False

    real_connect = socket.socket.connect
    real_connect_ex = socket.socket.connect_ex
    real_getaddrinfo = socket.getaddrinfo

    def connect(self, address):
        if not _is_local_address(address):
            _refuse(f"{address[0]}:{address[1]}" if isinstance(address, tuple) and len(address) > 1 else str(address))
        return real_connect(self, address)

    def connect_ex(self, address):
        if not _is_local_address(address):
            _refuse(f"{address[0]}:{address[1]}" if isinstance(address, tuple) and len(address) > 1 else str(address))
        return real_connect_ex(self, address)

    def getaddrinfo(host, port, *args, **kwargs):
        if not _is_local_host(host):
            _refuse(f"{host}:{port}")
        return real_getaddrinfo(host, port, *args, **kwargs)

    socket.socket.connect = connect  # type: ignore[assignment]
    socket.socket.connect_ex = connect_ex  # type: ignore[assignment]
    socket.getaddrinfo = getaddrinfo  # type: ignore[assignment]
    _installed = True
    return True
