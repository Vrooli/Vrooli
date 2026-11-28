import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FitnessTier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FITNESS_TIER_UNSPECIFIED: _ClassVar[FitnessTier]
    FITNESS_TIER_STRONG: _ClassVar[FitnessTier]
    FITNESS_TIER_FAIR: _ClassVar[FitnessTier]
    FITNESS_TIER_WEAK: _ClassVar[FitnessTier]

class ReferenceEligibility(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REFERENCE_ELIGIBILITY_UNSPECIFIED: _ClassVar[ReferenceEligibility]
    REFERENCE_ELIGIBILITY_ELIGIBLE: _ClassVar[ReferenceEligibility]
    REFERENCE_ELIGIBILITY_CANDIDATE: _ClassVar[ReferenceEligibility]
    REFERENCE_ELIGIBILITY_INELIGIBLE: _ClassVar[ReferenceEligibility]
FITNESS_TIER_UNSPECIFIED: FitnessTier
FITNESS_TIER_STRONG: FitnessTier
FITNESS_TIER_FAIR: FitnessTier
FITNESS_TIER_WEAK: FitnessTier
REFERENCE_ELIGIBILITY_UNSPECIFIED: ReferenceEligibility
REFERENCE_ELIGIBILITY_ELIGIBLE: ReferenceEligibility
REFERENCE_ELIGIBILITY_CANDIDATE: ReferenceEligibility
REFERENCE_ELIGIBILITY_INELIGIBLE: ReferenceEligibility

class TemplateFitness(_message.Message):
    __slots__ = ("template", "per_replica_cost", "drift_surface_count", "comment_only_contract_count", "coordinated_edit_count", "tier")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    PER_REPLICA_COST_FIELD_NUMBER: _ClassVar[int]
    DRIFT_SURFACE_COUNT_FIELD_NUMBER: _ClassVar[int]
    COMMENT_ONLY_CONTRACT_COUNT_FIELD_NUMBER: _ClassVar[int]
    COORDINATED_EDIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    template: str
    per_replica_cost: int
    drift_surface_count: int
    comment_only_contract_count: int
    coordinated_edit_count: int
    tier: FitnessTier
    def __init__(self, template: _Optional[str] = ..., per_replica_cost: _Optional[int] = ..., drift_surface_count: _Optional[int] = ..., comment_only_contract_count: _Optional[int] = ..., coordinated_edit_count: _Optional[int] = ..., tier: _Optional[_Union[FitnessTier, str]] = ...) -> None: ...

class ReferenceHealth(_message.Message):
    __slots__ = ("scenario", "stale_from_template", "last_template_sync", "clean_on_all_tools", "stability_days", "breadth", "eligibility")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STALE_FROM_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    LAST_TEMPLATE_SYNC_FIELD_NUMBER: _ClassVar[int]
    CLEAN_ON_ALL_TOOLS_FIELD_NUMBER: _ClassVar[int]
    STABILITY_DAYS_FIELD_NUMBER: _ClassVar[int]
    BREADTH_FIELD_NUMBER: _ClassVar[int]
    ELIGIBILITY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    stale_from_template: bool
    last_template_sync: _timestamp_pb2.Timestamp
    clean_on_all_tools: bool
    stability_days: int
    breadth: int
    eligibility: ReferenceEligibility
    def __init__(self, scenario: _Optional[str] = ..., stale_from_template: _Optional[bool] = ..., last_template_sync: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., clean_on_all_tools: _Optional[bool] = ..., stability_days: _Optional[int] = ..., breadth: _Optional[int] = ..., eligibility: _Optional[_Union[ReferenceEligibility, str]] = ...) -> None: ...

class FitnessTrendPoint(_message.Message):
    __slots__ = ("template", "at", "per_replica_cost", "coordinated_edit_count")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    PER_REPLICA_COST_FIELD_NUMBER: _ClassVar[int]
    COORDINATED_EDIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    template: str
    at: _timestamp_pb2.Timestamp
    per_replica_cost: int
    coordinated_edit_count: int
    def __init__(self, template: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., per_replica_cost: _Optional[int] = ..., coordinated_edit_count: _Optional[int] = ...) -> None: ...

class GetConvergenceStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetConvergenceStatusResponse(_message.Message):
    __slots__ = ("templates", "references")
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[TemplateFitness]
    references: _containers.RepeatedCompositeFieldContainer[ReferenceHealth]
    def __init__(self, templates: _Optional[_Iterable[_Union[TemplateFitness, _Mapping]]] = ..., references: _Optional[_Iterable[_Union[ReferenceHealth, _Mapping]]] = ...) -> None: ...

class GetTemplateFitnessRequest(_message.Message):
    __slots__ = ("template",)
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    template: str
    def __init__(self, template: _Optional[str] = ...) -> None: ...

class GetTemplateFitnessResponse(_message.Message):
    __slots__ = ("templates",)
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[TemplateFitness]
    def __init__(self, templates: _Optional[_Iterable[_Union[TemplateFitness, _Mapping]]] = ...) -> None: ...

class ListReferencesRequest(_message.Message):
    __slots__ = ("eligibility",)
    ELIGIBILITY_FIELD_NUMBER: _ClassVar[int]
    eligibility: ReferenceEligibility
    def __init__(self, eligibility: _Optional[_Union[ReferenceEligibility, str]] = ...) -> None: ...

class ListReferencesResponse(_message.Message):
    __slots__ = ("references",)
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    references: _containers.RepeatedCompositeFieldContainer[ReferenceHealth]
    def __init__(self, references: _Optional[_Iterable[_Union[ReferenceHealth, _Mapping]]] = ...) -> None: ...

class GetConvergenceTrendRequest(_message.Message):
    __slots__ = ("template",)
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    template: str
    def __init__(self, template: _Optional[str] = ...) -> None: ...

class GetConvergenceTrendResponse(_message.Message):
    __slots__ = ("points",)
    POINTS_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[FitnessTrendPoint]
    def __init__(self, points: _Optional[_Iterable[_Union[FitnessTrendPoint, _Mapping]]] = ...) -> None: ...
