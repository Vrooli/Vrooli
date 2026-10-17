from audio_tools.v1.common import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Capability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPABILITY_UNSPECIFIED: _ClassVar[Capability]
    CAPABILITY_STT: _ClassVar[Capability]
    CAPABILITY_TTS: _ClassVar[Capability]
    CAPABILITY_SUMMARIZE: _ClassVar[Capability]
    CAPABILITY_TRANSCODE: _ClassVar[Capability]
CAPABILITY_UNSPECIFIED: Capability
CAPABILITY_STT: Capability
CAPABILITY_TTS: Capability
CAPABILITY_SUMMARIZE: Capability
CAPABILITY_TRANSCODE: Capability

class RunSuiteRequest(_message.Message):
    __slots__ = ("capabilities",)
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedScalarFieldContainer[Capability]
    def __init__(self, capabilities: _Optional[_Iterable[_Union[Capability, str]]] = ...) -> None: ...

class RunSuiteResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: RunSuiteResult
    def __init__(self, run: _Optional[_Union[RunSuiteResult, _Mapping]] = ...) -> None: ...

class GetLastRunRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetLastRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: RunSuiteResult
    def __init__(self, run: _Optional[_Union[RunSuiteResult, _Mapping]] = ...) -> None: ...

class ListFixturesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListFixturesResponse(_message.Message):
    __slots__ = ("fixtures",)
    FIXTURES_FIELD_NUMBER: _ClassVar[int]
    fixtures: _containers.RepeatedCompositeFieldContainer[Fixture]
    def __init__(self, fixtures: _Optional[_Iterable[_Union[Fixture, _Mapping]]] = ...) -> None: ...

class Fixture(_message.Message):
    __slots__ = ("id", "capability", "description", "size_bytes", "content_type")
    ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    id: str
    capability: Capability
    description: str
    size_bytes: int
    content_type: str
    def __init__(self, id: _Optional[str] = ..., capability: _Optional[_Union[Capability, str]] = ..., description: _Optional[str] = ..., size_bytes: _Optional[int] = ..., content_type: _Optional[str] = ...) -> None: ...

class RunSuiteResult(_message.Message):
    __slots__ = ("run_id", "started_at_unix_ms", "finished_at_unix_ms", "steps", "overall")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    OVERALL_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    started_at_unix_ms: int
    finished_at_unix_ms: int
    steps: _containers.RepeatedCompositeFieldContainer[SuiteStepResult]
    overall: SuiteOverall
    def __init__(self, run_id: _Optional[str] = ..., started_at_unix_ms: _Optional[int] = ..., finished_at_unix_ms: _Optional[int] = ..., steps: _Optional[_Iterable[_Union[SuiteStepResult, _Mapping]]] = ..., overall: _Optional[_Union[SuiteOverall, _Mapping]] = ...) -> None: ...

class SuiteOverall(_message.Message):
    __slots__ = ("status", "pass_count", "fail_count", "total_count")
    class Status(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        STATUS_UNSPECIFIED: _ClassVar[SuiteOverall.Status]
        STATUS_NEVER: _ClassVar[SuiteOverall.Status]
        STATUS_PASS: _ClassVar[SuiteOverall.Status]
        STATUS_PARTIAL: _ClassVar[SuiteOverall.Status]
        STATUS_FAIL: _ClassVar[SuiteOverall.Status]
    STATUS_UNSPECIFIED: SuiteOverall.Status
    STATUS_NEVER: SuiteOverall.Status
    STATUS_PASS: SuiteOverall.Status
    STATUS_PARTIAL: SuiteOverall.Status
    STATUS_FAIL: SuiteOverall.Status
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PASS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAIL_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    status: SuiteOverall.Status
    pass_count: int
    fail_count: int
    total_count: int
    def __init__(self, status: _Optional[_Union[SuiteOverall.Status, str]] = ..., pass_count: _Optional[int] = ..., fail_count: _Optional[int] = ..., total_count: _Optional[int] = ...) -> None: ...

class SuiteStepResult(_message.Message):
    __slots__ = ("capability", "ok", "error_code", "error_message", "started_at_unix_ms", "finished_at_unix_ms", "provider_tier", "provider_id", "model_id", "latency_ms", "details")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    capability: Capability
    ok: bool
    error_code: str
    error_message: str
    started_at_unix_ms: int
    finished_at_unix_ms: int
    provider_tier: _common_pb2.ProviderTier
    provider_id: str
    model_id: str
    latency_ms: float
    details: _containers.ScalarMap[str, str]
    def __init__(self, capability: _Optional[_Union[Capability, str]] = ..., ok: _Optional[bool] = ..., error_code: _Optional[str] = ..., error_message: _Optional[str] = ..., started_at_unix_ms: _Optional[int] = ..., finished_at_unix_ms: _Optional[int] = ..., provider_tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ..., details: _Optional[_Mapping[str, str]] = ...) -> None: ...
