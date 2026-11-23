"""Interpreter-start hook: puts the model-download guard in front of every Python
process image-tools launches (sidecar ops, iopaint, probes), including
console-script entry points that never import image_tools_sidecar themselves.

Python imports sitecustomize automatically when this directory is on PYTHONPATH.
The guard is inert unless the process carries IMAGE_TOOLS_SIDECAR_NETWORK=blocked.
"""

try:
    from image_tools_sidecar import _containment

    _containment.install()
except ImportError:  # sidecar not importable from this interpreter: nothing to guard
    pass
