"""The builtin surface a submitted program may reference.

This list is the single authority for two consumers: the kernel, which installs
it as ``__builtins__`` for every program, and the static analyzer, which must
never flag a name the kernel would have resolved. Keeping one list is the whole
point — a name present in one and absent from the other becomes a refusal of a
correct program.

Excluded on purpose: ``eval``, ``exec``, ``compile``, ``globals``, ``locals``,
``vars``, ``input``, ``breakpoint``, ``help``, ``exit``, ``quit``, and
``memoryview``. The posture is trusted-local-agent rather than adversarial
containment, so this is a guardrail against accidents, not a security boundary;
``__import__`` is allowed because programs legitimately need the standard
library.
"""
from __future__ import annotations

import builtins

_CALLABLES = (
    "__build_class__", "__import__", "abs", "all", "any", "ascii", "bin", "bool", "bytearray",
    "bytes", "callable", "chr", "classmethod", "complex", "delattr", "dict", "dir", "divmod",
    "enumerate", "filter", "float", "format", "frozenset", "getattr", "hasattr", "hash", "hex",
    "id", "int", "isinstance", "issubclass", "iter", "len", "list", "map", "max", "min", "next",
    "object", "oct", "open", "ord", "pow", "print", "property", "range", "repr", "reversed", "round",
    "set", "setattr", "slice", "sorted", "staticmethod", "str", "sum", "super", "tuple", "type",
    "zip",
)

# Every exception a program can reasonably catch or raise. Their absence was the
# defect that refused `except KeyError:` — an ordinary, correct program.
_EXCEPTIONS = (
    "ArithmeticError", "AssertionError", "AttributeError", "BaseException", "BlockingIOError",
    "BrokenPipeError", "BufferError", "BytesWarning", "ConnectionError", "ConnectionAbortedError",
    "ConnectionRefusedError", "ConnectionResetError", "DeprecationWarning", "EOFError",
    "Exception", "FileExistsError", "FileNotFoundError", "FloatingPointError", "GeneratorExit",
    "ImportError", "IndentationError", "IndexError", "InterruptedError", "IsADirectoryError",
    "KeyError", "KeyboardInterrupt", "LookupError", "MemoryError", "ModuleNotFoundError",
    "NameError", "NotADirectoryError", "NotImplementedError", "OSError", "OverflowError",
    "PermissionError", "ProcessLookupError", "RecursionError", "ReferenceError", "RuntimeError",
    "RuntimeWarning", "StopAsyncIteration", "StopIteration", "SyntaxError", "SystemError",
    "TabError", "TimeoutError", "TypeError", "UnboundLocalError", "UnicodeDecodeError",
    "UnicodeEncodeError", "UnicodeError", "UserWarning", "ValueError", "Warning",
    "ZeroDivisionError",
)

_CONSTANTS = ("Ellipsis", "NotImplemented", "__name__", "__spec__")

SAFE_BUILTIN_NAMES = tuple(sorted(set(_CALLABLES + _EXCEPTIONS + _CONSTANTS)))

SAFE_BUILTINS = {name: getattr(builtins, name) for name in SAFE_BUILTIN_NAMES if hasattr(builtins, name)}
