from flow_verifier.v1.runs import runs_pb2 as _runs_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class VerificationMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VERIFICATION_MODE_UNSPECIFIED: _ClassVar[VerificationMode]
    VERIFICATION_MODE_CHECK: _ClassVar[VerificationMode]
    VERIFICATION_MODE_GENERATE: _ClassVar[VerificationMode]
VERIFICATION_MODE_UNSPECIFIED: VerificationMode
VERIFICATION_MODE_CHECK: VerificationMode
VERIFICATION_MODE_GENERATE: VerificationMode

class StartVerificationRequest(_message.Message):
    __slots__ = ("root", "flow_id", "mode")
    ROOT_FIELD_NUMBER: _ClassVar[int]
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    root: str
    flow_id: str
    mode: VerificationMode
    def __init__(self, root: _Optional[str] = ..., flow_id: _Optional[str] = ..., mode: _Optional[_Union[VerificationMode, str]] = ...) -> None: ...

class StartVerificationResponse(_message.Message):
    __slots__ = ("status", "error_message", "runs")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    status: str
    error_message: str
    runs: _containers.RepeatedCompositeFieldContainer[_runs_pb2.Run]
    def __init__(self, status: _Optional[str] = ..., error_message: _Optional[str] = ..., runs: _Optional[_Iterable[_Union[_runs_pb2.Run, _Mapping]]] = ...) -> None: ...

class GetVerificationRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetVerificationResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: _runs_pb2.Run
    def __init__(self, run: _Optional[_Union[_runs_pb2.Run, _Mapping]] = ...) -> None: ...
