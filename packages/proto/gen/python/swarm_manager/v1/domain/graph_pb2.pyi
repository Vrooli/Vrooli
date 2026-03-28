from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GraphPosition(_message.Message):
    __slots__ = ("x", "y")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...

class GraphInitiativeRollup(_message.Message):
    __slots__ = ("total", "completed", "in_progress", "failed", "pending")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    IN_PROGRESS_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    total: int
    completed: int
    in_progress: int
    failed: int
    pending: int
    def __init__(self, total: _Optional[int] = ..., completed: _Optional[int] = ..., in_progress: _Optional[int] = ..., failed: _Optional[int] = ..., pending: _Optional[int] = ...) -> None: ...

class GraphBacklogNodeData(_message.Message):
    __slots__ = ("kind", "name", "title", "status", "priority")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    title: str
    status: str
    priority: int
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ...) -> None: ...

class GraphInitiativeNodeData(_message.Message):
    __slots__ = ("name", "title", "status", "rollup")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ROLLUP_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    status: str
    rollup: GraphInitiativeRollup
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., rollup: _Optional[_Union[GraphInitiativeRollup, _Mapping]] = ...) -> None: ...

class GraphCaptureNodeData(_message.Message):
    __slots__ = ("id", "text", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    text: str
    status: str
    def __init__(self, id: _Optional[str] = ..., text: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GraphScenarioNodeData(_message.Message):
    __slots__ = ("name", "status")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GraphExecutionNodeData(_message.Message):
    __slots__ = ("execution_id", "backlog_kind", "backlog_name", "status", "mode", "run_id")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_KIND_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    backlog_kind: str
    backlog_name: str
    status: str
    mode: str
    run_id: str
    def __init__(self, execution_id: _Optional[str] = ..., backlog_kind: _Optional[str] = ..., backlog_name: _Optional[str] = ..., status: _Optional[str] = ..., mode: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class GraphRunNodeData(_message.Message):
    __slots__ = ("run_id", "task_id", "status")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    task_id: str
    status: str
    def __init__(self, run_id: _Optional[str] = ..., task_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GraphNodeData(_message.Message):
    __slots__ = ("backlog", "initiative", "capture", "scenario", "execution", "run")
    BACKLOG_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    backlog: GraphBacklogNodeData
    initiative: GraphInitiativeNodeData
    capture: GraphCaptureNodeData
    scenario: GraphScenarioNodeData
    execution: GraphExecutionNodeData
    run: GraphRunNodeData
    def __init__(self, backlog: _Optional[_Union[GraphBacklogNodeData, _Mapping]] = ..., initiative: _Optional[_Union[GraphInitiativeNodeData, _Mapping]] = ..., capture: _Optional[_Union[GraphCaptureNodeData, _Mapping]] = ..., scenario: _Optional[_Union[GraphScenarioNodeData, _Mapping]] = ..., execution: _Optional[_Union[GraphExecutionNodeData, _Mapping]] = ..., run: _Optional[_Union[GraphRunNodeData, _Mapping]] = ...) -> None: ...

class GraphNode(_message.Message):
    __slots__ = ("id", "type", "data", "position")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    data: GraphNodeData
    position: GraphPosition
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., data: _Optional[_Union[GraphNodeData, _Mapping]] = ..., position: _Optional[_Union[GraphPosition, _Mapping]] = ...) -> None: ...

class GraphEdge(_message.Message):
    __slots__ = ("id", "source", "target", "type")
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    id: str
    source: str
    target: str
    type: str
    def __init__(self, id: _Optional[str] = ..., source: _Optional[str] = ..., target: _Optional[str] = ..., type: _Optional[str] = ...) -> None: ...

class GraphMeta(_message.Message):
    __slots__ = ("lens", "node_count", "edge_count", "generated_at", "agent_manager_available")
    LENS_FIELD_NUMBER: _ClassVar[int]
    NODE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EDGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    AGENT_MANAGER_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    lens: str
    node_count: int
    edge_count: int
    generated_at: str
    agent_manager_available: bool
    def __init__(self, lens: _Optional[str] = ..., node_count: _Optional[int] = ..., edge_count: _Optional[int] = ..., generated_at: _Optional[str] = ..., agent_manager_available: _Optional[bool] = ...) -> None: ...
