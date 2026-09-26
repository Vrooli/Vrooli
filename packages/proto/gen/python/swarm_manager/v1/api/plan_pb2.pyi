from swarm_manager.v1.domain import plan_pb2 as _plan_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PlanBoardResponse(_message.Message):
    __slots__ = ("now", "next", "later", "done", "meta")
    NOW_FIELD_NUMBER: _ClassVar[int]
    NEXT_FIELD_NUMBER: _ClassVar[int]
    LATER_FIELD_NUMBER: _ClassVar[int]
    DONE_FIELD_NUMBER: _ClassVar[int]
    META_FIELD_NUMBER: _ClassVar[int]
    now: _plan_pb2.PlanNowSummary
    next: _plan_pb2.PlanColumn
    later: _plan_pb2.PlanColumn
    done: _plan_pb2.PlanColumn
    meta: _plan_pb2.PlanBoardMeta
    def __init__(self, now: _Optional[_Union[_plan_pb2.PlanNowSummary, _Mapping]] = ..., next: _Optional[_Union[_plan_pb2.PlanColumn, _Mapping]] = ..., later: _Optional[_Union[_plan_pb2.PlanColumn, _Mapping]] = ..., done: _Optional[_Union[_plan_pb2.PlanColumn, _Mapping]] = ..., meta: _Optional[_Union[_plan_pb2.PlanBoardMeta, _Mapping]] = ...) -> None: ...
