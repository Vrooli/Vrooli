from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class PrivacyClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRIVACY_CLASS_UNSPECIFIED: _ClassVar[PrivacyClass]
    PRIVACY_CLASS_PUBLIC: _ClassVar[PrivacyClass]
    PRIVACY_CLASS_INTERNAL: _ClassVar[PrivacyClass]
    PRIVACY_CLASS_CONFIDENTIAL: _ClassVar[PrivacyClass]
    PRIVACY_CLASS_SECRET: _ClassVar[PrivacyClass]

class AnchorKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ANCHOR_KIND_UNSPECIFIED: _ClassVar[AnchorKind]
    ANCHOR_KIND_LOGICAL: _ClassVar[AnchorKind]
    ANCHOR_KIND_GEOMETRIC: _ClassVar[AnchorKind]
    ANCHOR_KIND_TABULAR: _ClassVar[AnchorKind]

class Tier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TIER_UNSPECIFIED: _ClassVar[Tier]
    TIER_ONE: _ClassVar[Tier]
    TIER_TWO: _ClassVar[Tier]
    TIER_THREE: _ClassVar[Tier]

class TerminalState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TERMINAL_STATE_UNSPECIFIED: _ClassVar[TerminalState]
    TERMINAL_STATE_PARSED: _ClassVar[TerminalState]
    TERMINAL_STATE_NO_HANDLER_FOR_FORMAT: _ClassVar[TerminalState]
    TERMINAL_STATE_HANDLER_UNAVAILABLE: _ClassVar[TerminalState]
    TERMINAL_STATE_HANDLER_FAILED: _ClassVar[TerminalState]
    TERMINAL_STATE_BLOCKED_BY_POLICY: _ClassVar[TerminalState]
    TERMINAL_STATE_UNSUPPORTED_VARIANT: _ClassVar[TerminalState]
PRIVACY_CLASS_UNSPECIFIED: PrivacyClass
PRIVACY_CLASS_PUBLIC: PrivacyClass
PRIVACY_CLASS_INTERNAL: PrivacyClass
PRIVACY_CLASS_CONFIDENTIAL: PrivacyClass
PRIVACY_CLASS_SECRET: PrivacyClass
ANCHOR_KIND_UNSPECIFIED: AnchorKind
ANCHOR_KIND_LOGICAL: AnchorKind
ANCHOR_KIND_GEOMETRIC: AnchorKind
ANCHOR_KIND_TABULAR: AnchorKind
TIER_UNSPECIFIED: Tier
TIER_ONE: Tier
TIER_TWO: Tier
TIER_THREE: Tier
TERMINAL_STATE_UNSPECIFIED: TerminalState
TERMINAL_STATE_PARSED: TerminalState
TERMINAL_STATE_NO_HANDLER_FOR_FORMAT: TerminalState
TERMINAL_STATE_HANDLER_UNAVAILABLE: TerminalState
TERMINAL_STATE_HANDLER_FAILED: TerminalState
TERMINAL_STATE_BLOCKED_BY_POLICY: TerminalState
TERMINAL_STATE_UNSUPPORTED_VARIANT: TerminalState
