from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListPlansRequest(_message.Message):
    __slots__ = ("status", "include_archived")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    status: _model_pb2.PlanStatus
    include_archived: bool
    def __init__(self, status: _Optional[_Union[_model_pb2.PlanStatus, str]] = ..., include_archived: _Optional[bool] = ...) -> None: ...

class ListPlansResponse(_message.Message):
    __slots__ = ("plans",)
    PLANS_FIELD_NUMBER: _ClassVar[int]
    plans: _containers.RepeatedCompositeFieldContainer[_model_pb2.Plan]
    def __init__(self, plans: _Optional[_Iterable[_Union[_model_pb2.Plan, _Mapping]]] = ...) -> None: ...

class GetPlanRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetPlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class CreatePlanRequest(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class CreatePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class UpdatePlanRequest(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class UpdatePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class ArchivePlanRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ArchivePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class RenderMarkdownRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RenderMarkdownResponse(_message.Message):
    __slots__ = ("markdown",)
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    markdown: str
    def __init__(self, markdown: _Optional[str] = ...) -> None: ...

class AddPhaseRequest(_message.Message):
    __slots__ = ("plan_id", "phase")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase: _model_pb2.Phase
    def __init__(self, plan_id: _Optional[str] = ..., phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ...) -> None: ...

class AddPhaseResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class UpdatePhaseRequest(_message.Message):
    __slots__ = ("plan_id", "phase")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase: _model_pb2.Phase
    def __init__(self, plan_id: _Optional[str] = ..., phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ...) -> None: ...

class UpdatePhaseResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class GetGraphRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class GetGraphResponse(_message.Message):
    __slots__ = ("edges",)
    EDGES_FIELD_NUMBER: _ClassVar[int]
    edges: _containers.RepeatedCompositeFieldContainer[_model_pb2.PlanEdge]
    def __init__(self, edges: _Optional[_Iterable[_Union[_model_pb2.PlanEdge, _Mapping]]] = ...) -> None: ...

class LinkSupersessionRequest(_message.Message):
    __slots__ = ("superseding_plan_id", "superseded_plan_id")
    SUPERSEDING_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    superseding_plan_id: str
    superseded_plan_id: str
    def __init__(self, superseding_plan_id: _Optional[str] = ..., superseded_plan_id: _Optional[str] = ...) -> None: ...

class LinkSupersessionResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class ImportPlanRequest(_message.Message):
    __slots__ = ("source_path", "markdown")
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    source_path: str
    markdown: str
    def __init__(self, source_path: _Optional[str] = ..., markdown: _Optional[str] = ...) -> None: ...

class ImportPlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class MigratePlanRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class MigratePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class ListTemplatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTemplatesResponse(_message.Message):
    __slots__ = ("templates",)
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[PlanTemplate]
    def __init__(self, templates: _Optional[_Iterable[_Union[PlanTemplate, _Mapping]]] = ...) -> None: ...

class PlanTemplate(_message.Message):
    __slots__ = ("id", "name", "description", "surface")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    surface: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., surface: _Optional[str] = ...) -> None: ...

class CreateFromTemplateRequest(_message.Message):
    __slots__ = ("template_id", "title", "slug")
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    template_id: str
    title: str
    slug: str
    def __init__(self, template_id: _Optional[str] = ..., title: _Optional[str] = ..., slug: _Optional[str] = ...) -> None: ...

class CreateFromTemplateResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...
