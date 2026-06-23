from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PolicyChange(_message.Message):
    __slots__ = ("id", "target", "action", "status", "effects", "rollback_supported")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    target: str
    action: str
    status: str
    effects: _containers.RepeatedScalarFieldContainer[str]
    rollback_supported: bool
    def __init__(self, id: _Optional[str] = ..., target: _Optional[str] = ..., action: _Optional[str] = ..., status: _Optional[str] = ..., effects: _Optional[_Iterable[str]] = ..., rollback_supported: _Optional[bool] = ...) -> None: ...

class PreviewPolicyChangeRequest(_message.Message):
    __slots__ = ("target", "action", "values")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    target: str
    action: str
    values: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[str] = ..., action: _Optional[str] = ..., values: _Optional[_Iterable[str]] = ...) -> None: ...

class PreviewPolicyChangeResponse(_message.Message):
    __slots__ = ("preview",)
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    preview: PolicyChange
    def __init__(self, preview: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class ApplyPolicyChangeRequest(_message.Message):
    __slots__ = ("preview_id", "approved")
    PREVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    APPROVED_FIELD_NUMBER: _ClassVar[int]
    preview_id: str
    approved: bool
    def __init__(self, preview_id: _Optional[str] = ..., approved: _Optional[bool] = ...) -> None: ...

class ApplyPolicyChangeResponse(_message.Message):
    __slots__ = ("change",)
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    change: PolicyChange
    def __init__(self, change: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class RollbackPolicyChangeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RollbackPolicyChangeResponse(_message.Message):
    __slots__ = ("change",)
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    change: PolicyChange
    def __init__(self, change: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class PauseFilteringRequest(_message.Message):
    __slots__ = ("target", "duration")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    target: str
    duration: str
    def __init__(self, target: _Optional[str] = ..., duration: _Optional[str] = ...) -> None: ...

class PauseFilteringResponse(_message.Message):
    __slots__ = ("change",)
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    change: PolicyChange
    def __init__(self, change: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class ResumeFilteringRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class ResumeFilteringResponse(_message.Message):
    __slots__ = ("change",)
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    change: PolicyChange
    def __init__(self, change: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...
