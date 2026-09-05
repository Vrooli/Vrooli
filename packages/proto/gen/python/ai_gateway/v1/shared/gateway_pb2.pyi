from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RequestKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REQUEST_KIND_UNSPECIFIED: _ClassVar[RequestKind]
    REQUEST_KIND_TEXT_GENERATION: _ClassVar[RequestKind]
    REQUEST_KIND_TEXT_EMBEDDING: _ClassVar[RequestKind]
    REQUEST_KIND_STRUCTURED_EXTRACTION: _ClassVar[RequestKind]
    REQUEST_KIND_IMAGE_GENERATION: _ClassVar[RequestKind]
    REQUEST_KIND_VIDEO_GENERATION: _ClassVar[RequestKind]

class PrivacyClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRIVACY_CLASS_UNSPECIFIED: _ClassVar[PrivacyClass]
    PRIVACY_CLASS_PUBLIC: _ClassVar[PrivacyClass]
    PRIVACY_CLASS_INTERNAL: _ClassVar[PrivacyClass]
    PRIVACY_CLASS_CONFIDENTIAL: _ClassVar[PrivacyClass]
    PRIVACY_CLASS_SECRET: _ClassVar[PrivacyClass]

class Profile(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROFILE_UNSPECIFIED: _ClassVar[Profile]
    PROFILE_LOCAL_ONLY: _ClassVar[Profile]
    PROFILE_LOCAL_FIRST: _ClassVar[Profile]
    PROFILE_REMOTE_ONLY: _ClassVar[Profile]
    PROFILE_QUALITY_FIRST: _ClassVar[Profile]
    PROFILE_CHEAP_FIRST: _ClassVar[Profile]
    PROFILE_PRIVACY_SENSITIVE: _ClassVar[Profile]

class Modality(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MODALITY_UNSPECIFIED: _ClassVar[Modality]
    MODALITY_TEXT: _ClassVar[Modality]
    MODALITY_IMAGE: _ClassVar[Modality]
    MODALITY_VECTOR: _ClassVar[Modality]
    MODALITY_VIDEO: _ClassVar[Modality]
    MODALITY_AUDIO: _ClassVar[Modality]

class SamplingSupport(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SAMPLING_SUPPORT_UNSPECIFIED: _ClassVar[SamplingSupport]
    SAMPLING_SUPPORT_HONORED: _ClassVar[SamplingSupport]
    SAMPLING_SUPPORT_IGNORED: _ClassVar[SamplingSupport]
    SAMPLING_SUPPORT_REJECTED: _ClassVar[SamplingSupport]
    SAMPLING_SUPPORT_UNKNOWN: _ClassVar[SamplingSupport]

class OutputCapSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OUTPUT_CAP_SOURCE_UNSPECIFIED: _ClassVar[OutputCapSource]
    OUTPUT_CAP_SOURCE_REQUEST: _ClassVar[OutputCapSource]
    OUTPUT_CAP_SOURCE_ROLE_POLICY: _ClassVar[OutputCapSource]
    OUTPUT_CAP_SOURCE_NONE_IMPOSED: _ClassVar[OutputCapSource]
REQUEST_KIND_UNSPECIFIED: RequestKind
REQUEST_KIND_TEXT_GENERATION: RequestKind
REQUEST_KIND_TEXT_EMBEDDING: RequestKind
REQUEST_KIND_STRUCTURED_EXTRACTION: RequestKind
REQUEST_KIND_IMAGE_GENERATION: RequestKind
REQUEST_KIND_VIDEO_GENERATION: RequestKind
PRIVACY_CLASS_UNSPECIFIED: PrivacyClass
PRIVACY_CLASS_PUBLIC: PrivacyClass
PRIVACY_CLASS_INTERNAL: PrivacyClass
PRIVACY_CLASS_CONFIDENTIAL: PrivacyClass
PRIVACY_CLASS_SECRET: PrivacyClass
PROFILE_UNSPECIFIED: Profile
PROFILE_LOCAL_ONLY: Profile
PROFILE_LOCAL_FIRST: Profile
PROFILE_REMOTE_ONLY: Profile
PROFILE_QUALITY_FIRST: Profile
PROFILE_CHEAP_FIRST: Profile
PROFILE_PRIVACY_SENSITIVE: Profile
MODALITY_UNSPECIFIED: Modality
MODALITY_TEXT: Modality
MODALITY_IMAGE: Modality
MODALITY_VECTOR: Modality
MODALITY_VIDEO: Modality
MODALITY_AUDIO: Modality
SAMPLING_SUPPORT_UNSPECIFIED: SamplingSupport
SAMPLING_SUPPORT_HONORED: SamplingSupport
SAMPLING_SUPPORT_IGNORED: SamplingSupport
SAMPLING_SUPPORT_REJECTED: SamplingSupport
SAMPLING_SUPPORT_UNKNOWN: SamplingSupport
OUTPUT_CAP_SOURCE_UNSPECIFIED: OutputCapSource
OUTPUT_CAP_SOURCE_REQUEST: OutputCapSource
OUTPUT_CAP_SOURCE_ROLE_POLICY: OutputCapSource
OUTPUT_CAP_SOURCE_NONE_IMPOSED: OutputCapSource

class SamplingControls(_message.Message):
    __slots__ = ("temperature",)
    TEMPERATURE_FIELD_NUMBER: _ClassVar[int]
    temperature: float
    def __init__(self, temperature: _Optional[float] = ...) -> None: ...

class AppliedSettings(_message.Message):
    __slots__ = ("temperature_sent", "temperature_support", "max_output_tokens_effective", "max_output_tokens_source")
    TEMPERATURE_SENT_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_SUPPORT_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_SOURCE_FIELD_NUMBER: _ClassVar[int]
    temperature_sent: float
    temperature_support: SamplingSupport
    max_output_tokens_effective: int
    max_output_tokens_source: OutputCapSource
    def __init__(self, temperature_sent: _Optional[float] = ..., temperature_support: _Optional[_Union[SamplingSupport, str]] = ..., max_output_tokens_effective: _Optional[int] = ..., max_output_tokens_source: _Optional[_Union[OutputCapSource, str]] = ...) -> None: ...

class Attachment(_message.Message):
    __slots__ = ("modality", "media_type", "width", "height", "bytes", "inline_bytes", "reference")
    MODALITY_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    INLINE_BYTES_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    modality: Modality
    media_type: str
    width: int
    height: int
    bytes: int
    inline_bytes: bytes
    reference: str
    def __init__(self, modality: _Optional[_Union[Modality, str]] = ..., media_type: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., bytes: _Optional[int] = ..., inline_bytes: _Optional[bytes] = ..., reference: _Optional[str] = ...) -> None: ...

class GatewayRequest(_message.Message):
    __slots__ = ("kind", "role", "profile", "privacy_class", "operation", "scenario", "timeout_ms", "max_cost_usd", "max_output_tokens", "request_id", "metadata", "attachments", "sampling")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    KIND_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_COST_USD_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    SAMPLING_FIELD_NUMBER: _ClassVar[int]
    kind: RequestKind
    role: str
    profile: Profile
    privacy_class: PrivacyClass
    operation: str
    scenario: str
    timeout_ms: int
    max_cost_usd: float
    max_output_tokens: int
    request_id: str
    metadata: _containers.ScalarMap[str, str]
    attachments: _containers.RepeatedCompositeFieldContainer[Attachment]
    sampling: SamplingControls
    def __init__(self, kind: _Optional[_Union[RequestKind, str]] = ..., role: _Optional[str] = ..., profile: _Optional[_Union[Profile, str]] = ..., privacy_class: _Optional[_Union[PrivacyClass, str]] = ..., operation: _Optional[str] = ..., scenario: _Optional[str] = ..., timeout_ms: _Optional[int] = ..., max_cost_usd: _Optional[float] = ..., max_output_tokens: _Optional[int] = ..., request_id: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., attachments: _Optional[_Iterable[_Union[Attachment, _Mapping]]] = ..., sampling: _Optional[_Union[SamplingControls, _Mapping]] = ...) -> None: ...

class ValidationIssue(_message.Message):
    __slots__ = ("field", "code", "message")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    field: str
    code: str
    message: str
    def __init__(self, field: _Optional[str] = ..., code: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
