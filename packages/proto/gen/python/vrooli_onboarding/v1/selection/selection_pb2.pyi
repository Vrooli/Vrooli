from setup.v1 import selection_pb2 as _selection_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListScenariosRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class GetCoreSetRequest(_message.Message):
    __slots__ = ("target", "seed")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    target: str
    seed: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[str] = ..., seed: _Optional[_Iterable[str]] = ...) -> None: ...

class GetRecommendationRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class AcceptRecommendationRequest(_message.Message):
    __slots__ = ("target", "selection", "profile")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    target: str
    selection: _selection_pb2.Selection
    profile: str
    def __init__(self, target: _Optional[str] = ..., selection: _Optional[_Union[_selection_pb2.Selection, _Mapping]] = ..., profile: _Optional[str] = ...) -> None: ...

class GetClosureRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class GetUnionRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class CreateHandoffRequest(_message.Message):
    __slots__ = ("target", "machine_id", "node_id", "node_kind", "desired_selection")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_KIND_FIELD_NUMBER: _ClassVar[int]
    DESIRED_SELECTION_FIELD_NUMBER: _ClassVar[int]
    target: str
    machine_id: str
    node_id: str
    node_kind: str
    desired_selection: _selection_pb2.Selection
    def __init__(self, target: _Optional[str] = ..., machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., node_kind: _Optional[str] = ..., desired_selection: _Optional[_Union[_selection_pb2.Selection, _Mapping]] = ...) -> None: ...

class Scenario(_message.Message):
    __slots__ = ("name", "description", "system_required", "enabled", "auto_restart", "resources")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    AUTO_RESTART_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    system_required: bool
    enabled: bool
    auto_restart: bool
    resources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., system_required: _Optional[bool] = ..., enabled: _Optional[bool] = ..., auto_restart: _Optional[bool] = ..., resources: _Optional[_Iterable[str]] = ...) -> None: ...

class ListScenariosResponse(_message.Message):
    __slots__ = ("scenarios", "count")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[Scenario]
    count: int
    def __init__(self, scenarios: _Optional[_Iterable[_Union[Scenario, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class SupervisionAttributionStep(_message.Message):
    __slots__ = ("name", "kind", "declared_by", "supervision_intent", "source")
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DECLARED_BY_FIELD_NUMBER: _ClassVar[int]
    SUPERVISION_INTENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    name: str
    kind: str
    declared_by: str
    supervision_intent: str
    source: str
    def __init__(self, name: _Optional[str] = ..., kind: _Optional[str] = ..., declared_by: _Optional[str] = ..., supervision_intent: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class SupervisionMember(_message.Message):
    __slots__ = ("name", "kind", "supervision_intent", "attribution_chain")
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUPERVISION_INTENT_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_CHAIN_FIELD_NUMBER: _ClassVar[int]
    name: str
    kind: str
    supervision_intent: str
    attribution_chain: _containers.RepeatedCompositeFieldContainer[SupervisionAttributionStep]
    def __init__(self, name: _Optional[str] = ..., kind: _Optional[str] = ..., supervision_intent: _Optional[str] = ..., attribution_chain: _Optional[_Iterable[_Union[SupervisionAttributionStep, _Mapping]]] = ...) -> None: ...

class GetCoreSetResponse(_message.Message):
    __slots__ = ("available", "seed", "trusted_base", "members", "member_counts", "load_errors", "error")
    class MemberCountsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class LoadErrorsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    TRUSTED_BASE_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    MEMBER_COUNTS_FIELD_NUMBER: _ClassVar[int]
    LOAD_ERRORS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    available: bool
    seed: _containers.RepeatedScalarFieldContainer[str]
    trusted_base: _containers.RepeatedScalarFieldContainer[str]
    members: _containers.RepeatedCompositeFieldContainer[SupervisionMember]
    member_counts: _containers.ScalarMap[str, int]
    load_errors: _containers.ScalarMap[str, str]
    error: str
    def __init__(self, available: _Optional[bool] = ..., seed: _Optional[_Iterable[str]] = ..., trusted_base: _Optional[_Iterable[str]] = ..., members: _Optional[_Iterable[_Union[SupervisionMember, _Mapping]]] = ..., member_counts: _Optional[_Mapping[str, int]] = ..., load_errors: _Optional[_Mapping[str, str]] = ..., error: _Optional[str] = ...) -> None: ...

class GetRecommendationResponse(_message.Message):
    __slots__ = ("profile", "scenarios", "resources", "explanation")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    profile: str
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    resources: _containers.RepeatedScalarFieldContainer[str]
    explanation: str
    def __init__(self, profile: _Optional[str] = ..., scenarios: _Optional[_Iterable[str]] = ..., resources: _Optional[_Iterable[str]] = ..., explanation: _Optional[str] = ...) -> None: ...

class AcceptRecommendationResponse(_message.Message):
    __slots__ = ("selection", "first_unsatisfied_step")
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    FIRST_UNSATISFIED_STEP_FIELD_NUMBER: _ClassVar[int]
    selection: _selection_pb2.Selection
    first_unsatisfied_step: int
    def __init__(self, selection: _Optional[_Union[_selection_pb2.Selection, _Mapping]] = ..., first_unsatisfied_step: _Optional[int] = ...) -> None: ...

class ClosureProvenance(_message.Message):
    __slots__ = ("kind",)
    KIND_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    kind: str
    def __init__(self, kind: _Optional[str] = ..., **kwargs) -> None: ...

class ClosureMember(_message.Message):
    __slots__ = ("name", "provenance", "required", "direct", "state", "reason", "policy")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    DIRECT_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    name: str
    provenance: _containers.RepeatedCompositeFieldContainer[ClosureProvenance]
    required: bool
    direct: bool
    state: str
    reason: str
    policy: str
    def __init__(self, name: _Optional[str] = ..., provenance: _Optional[_Iterable[_Union[ClosureProvenance, _Mapping]]] = ..., required: _Optional[bool] = ..., direct: _Optional[bool] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ..., policy: _Optional[str] = ...) -> None: ...

class GetClosureResponse(_message.Message):
    __slots__ = ("scenarios", "resources")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[ClosureMember]
    resources: _containers.RepeatedCompositeFieldContainer[ClosureMember]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[ClosureMember, _Mapping]]] = ..., resources: _Optional[_Iterable[_Union[ClosureMember, _Mapping]]] = ...) -> None: ...

class GetUnionResponse(_message.Message):
    __slots__ = ("scenarios", "resources", "host_tools", "safeguards", "catalog_paths", "resource_models", "required_resources", "optional_resources", "standalone_resources")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    HOST_TOOLS_FIELD_NUMBER: _ClassVar[int]
    SAFEGUARDS_FIELD_NUMBER: _ClassVar[int]
    CATALOG_PATHS_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_MODELS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    STANDALONE_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[ClosureMember]
    resources: _containers.RepeatedCompositeFieldContainer[ClosureMember]
    host_tools: _containers.RepeatedScalarFieldContainer[str]
    safeguards: _containers.RepeatedScalarFieldContainer[str]
    catalog_paths: _containers.RepeatedScalarFieldContainer[str]
    resource_models: _containers.RepeatedCompositeFieldContainer[Resource]
    required_resources: _containers.RepeatedCompositeFieldContainer[Resource]
    optional_resources: _containers.RepeatedCompositeFieldContainer[Resource]
    standalone_resources: _containers.RepeatedCompositeFieldContainer[Resource]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[ClosureMember, _Mapping]]] = ..., resources: _Optional[_Iterable[_Union[ClosureMember, _Mapping]]] = ..., host_tools: _Optional[_Iterable[str]] = ..., safeguards: _Optional[_Iterable[str]] = ..., catalog_paths: _Optional[_Iterable[str]] = ..., resource_models: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., required_resources: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., optional_resources: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., standalone_resources: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ...) -> None: ...

class Resource(_message.Message):
    __slots__ = ("name", "display_name", "description", "category", "enabled", "installed")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    category: str
    enabled: bool
    installed: bool
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., category: _Optional[str] = ..., enabled: _Optional[bool] = ..., installed: _Optional[bool] = ...) -> None: ...

class CreateHandoffResponse(_message.Message):
    __slots__ = ("selection",)
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    selection: _selection_pb2.Selection
    def __init__(self, selection: _Optional[_Union[_selection_pb2.Selection, _Mapping]] = ...) -> None: ...
