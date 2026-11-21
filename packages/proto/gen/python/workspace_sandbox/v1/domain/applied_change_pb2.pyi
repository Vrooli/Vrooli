from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class RunOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_OUTCOME_UNSPECIFIED: _ClassVar[RunOutcome]
    RUN_OUTCOME_SUCCESS: _ClassVar[RunOutcome]
    RUN_OUTCOME_FAILURE: _ClassVar[RunOutcome]
    RUN_OUTCOME_CANCELLED: _ClassVar[RunOutcome]
    RUN_OUTCOME_TIMEOUT: _ClassVar[RunOutcome]

class FileState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FILE_STATE_UNSPECIFIED: _ClassVar[FileState]
    FILE_STATE_APPLIED: _ClassVar[FileState]
    FILE_STATE_PENDING_REVIEW: _ClassVar[FileState]
    FILE_STATE_DENIED: _ClassVar[FileState]
RUN_OUTCOME_UNSPECIFIED: RunOutcome
RUN_OUTCOME_SUCCESS: RunOutcome
RUN_OUTCOME_FAILURE: RunOutcome
RUN_OUTCOME_CANCELLED: RunOutcome
RUN_OUTCOME_TIMEOUT: RunOutcome
FILE_STATE_UNSPECIFIED: FileState
FILE_STATE_APPLIED: FileState
FILE_STATE_PENDING_REVIEW: FileState
FILE_STATE_DENIED: FileState
