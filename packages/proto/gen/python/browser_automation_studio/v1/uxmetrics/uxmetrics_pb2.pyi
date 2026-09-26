from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetExecutionMetricsRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetExecutionMetricsResponse(_message.Message):
    __slots__ = ("metrics",)
    METRICS_FIELD_NUMBER: _ClassVar[int]
    metrics: _struct_pb2.Struct
    def __init__(self, metrics: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetStepMetricsRequest(_message.Message):
    __slots__ = ("execution_id", "step_index")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    step_index: int
    def __init__(self, execution_id: _Optional[str] = ..., step_index: _Optional[int] = ...) -> None: ...

class GetStepMetricsResponse(_message.Message):
    __slots__ = ("metrics",)
    METRICS_FIELD_NUMBER: _ClassVar[int]
    metrics: _struct_pb2.Struct
    def __init__(self, metrics: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ComputeExecutionMetricsRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class ComputeExecutionMetricsResponse(_message.Message):
    __slots__ = ("metrics",)
    METRICS_FIELD_NUMBER: _ClassVar[int]
    metrics: _struct_pb2.Struct
    def __init__(self, metrics: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetWorkflowAggregateRequest(_message.Message):
    __slots__ = ("workflow_id", "limit")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    limit: int
    def __init__(self, workflow_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetWorkflowAggregateResponse(_message.Message):
    __slots__ = ("aggregate",)
    AGGREGATE_FIELD_NUMBER: _ClassVar[int]
    aggregate: _struct_pb2.Struct
    def __init__(self, aggregate: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
