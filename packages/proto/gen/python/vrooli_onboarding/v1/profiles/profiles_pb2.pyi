from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListProfilesRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class EvaluateProfileRequest(_message.Message):
    __slots__ = ("target", "profile_id", "answers", "target_context", "manual_decisions", "preset", "profile_ids")
    class AnswersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    class ManualDecisionsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: bool
        def __init__(self, key: _Optional[str] = ..., value: _Optional[bool] = ...) -> None: ...
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWERS_FIELD_NUMBER: _ClassVar[int]
    TARGET_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    MANUAL_DECISIONS_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    PROFILE_IDS_FIELD_NUMBER: _ClassVar[int]
    target: str
    profile_id: str
    answers: _containers.MessageMap[str, _struct_pb2.Value]
    target_context: _struct_pb2.Struct
    manual_decisions: _containers.ScalarMap[str, bool]
    preset: ProfilePreset
    profile_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[str] = ..., profile_id: _Optional[str] = ..., answers: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., target_context: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., manual_decisions: _Optional[_Mapping[str, bool]] = ..., preset: _Optional[_Union[ProfilePreset, _Mapping]] = ..., profile_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ProfilePreset(_message.Message):
    __slots__ = ("id", "version", "source", "answers")
    class AnswersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ANSWERS_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    source: str
    answers: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., source: _Optional[str] = ..., answers: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...

class ProfileOption(_message.Message):
    __slots__ = ("id", "label_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    label_key: str
    def __init__(self, id: _Optional[str] = ..., label_key: _Optional[str] = ...) -> None: ...

class ProfileQuestion(_message.Message):
    __slots__ = ("id", "type", "prompt_key", "required", "options", "min_selections", "max_selections", "visible", "default_value")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PROMPT_KEY_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    MIN_SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    MAX_SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    VISIBLE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_VALUE_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    prompt_key: str
    required: bool
    options: _containers.RepeatedCompositeFieldContainer[ProfileOption]
    min_selections: int
    max_selections: int
    visible: bool
    default_value: _struct_pb2.Value
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., prompt_key: _Optional[str] = ..., required: _Optional[bool] = ..., options: _Optional[_Iterable[_Union[ProfileOption, _Mapping]]] = ..., min_selections: _Optional[int] = ..., max_selections: _Optional[int] = ..., visible: _Optional[bool] = ..., default_value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class ProfileRecommendation(_message.Message):
    __slots__ = ("capability_ref", "scenario_refs", "reason_key", "rule_id", "key", "selected", "required")
    CAPABILITY_REF_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_REFS_FIELD_NUMBER: _ClassVar[int]
    REASON_KEY_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    capability_ref: str
    scenario_refs: _containers.RepeatedScalarFieldContainer[str]
    reason_key: str
    rule_id: str
    key: str
    selected: bool
    required: bool
    def __init__(self, capability_ref: _Optional[str] = ..., scenario_refs: _Optional[_Iterable[str]] = ..., reason_key: _Optional[str] = ..., rule_id: _Optional[str] = ..., key: _Optional[str] = ..., selected: _Optional[bool] = ..., required: _Optional[bool] = ...) -> None: ...

class ProfileExplanation(_message.Message):
    __slots__ = ("rule_id", "capability_ref", "scenario_refs", "reason_key", "selected")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_REF_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_REFS_FIELD_NUMBER: _ClassVar[int]
    REASON_KEY_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    capability_ref: str
    scenario_refs: _containers.RepeatedScalarFieldContainer[str]
    reason_key: str
    selected: bool
    def __init__(self, rule_id: _Optional[str] = ..., capability_ref: _Optional[str] = ..., scenario_refs: _Optional[_Iterable[str]] = ..., reason_key: _Optional[str] = ..., selected: _Optional[bool] = ...) -> None: ...

class ProfileValidationIssue(_message.Message):
    __slots__ = ("field", "code", "message")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    field: str
    code: str
    message: str
    def __init__(self, field: _Optional[str] = ..., code: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ProfileConflict(_message.Message):
    __slots__ = ("code", "capability_ref", "recommendation_keys", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_REF_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_KEYS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    capability_ref: str
    recommendation_keys: _containers.RepeatedScalarFieldContainer[str]
    message: str
    def __init__(self, code: _Optional[str] = ..., capability_ref: _Optional[str] = ..., recommendation_keys: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ...) -> None: ...

class ProfileOutstanding(_message.Message):
    __slots__ = ("field", "code", "capability_ref", "message")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_REF_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    field: str
    code: str
    capability_ref: str
    message: str
    def __init__(self, field: _Optional[str] = ..., code: _Optional[str] = ..., capability_ref: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class Profile(_message.Message):
    __slots__ = ("id", "version", "title_key", "description_key", "owner", "provenance_source", "provenance_revision", "schema_version", "compatible_catalog_major", "default", "manual_selection_available")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    TITLE_KEY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_KEY_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    COMPATIBLE_CATALOG_MAJOR_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    MANUAL_SELECTION_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    title_key: str
    description_key: str
    owner: str
    provenance_source: str
    provenance_revision: str
    schema_version: str
    compatible_catalog_major: int
    default: bool
    manual_selection_available: bool
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., title_key: _Optional[str] = ..., description_key: _Optional[str] = ..., owner: _Optional[str] = ..., provenance_source: _Optional[str] = ..., provenance_revision: _Optional[str] = ..., schema_version: _Optional[str] = ..., compatible_catalog_major: _Optional[int] = ..., default: _Optional[bool] = ..., manual_selection_available: _Optional[bool] = ...) -> None: ...

class ListProfilesResponse(_message.Message):
    __slots__ = ("profiles",)
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[Profile]
    def __init__(self, profiles: _Optional[_Iterable[_Union[Profile, _Mapping]]] = ...) -> None: ...

class EvaluateProfileResponse(_message.Message):
    __slots__ = ("profile", "questions", "recommendations", "scenarios", "resources", "issues", "valid", "explanations", "digest", "catalog_revision", "conflicts", "outstanding", "preset", "profiles")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    VALID_FIELD_NUMBER: _ClassVar[int]
    EXPLANATIONS_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    CATALOG_REVISION_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    OUTSTANDING_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    questions: _containers.RepeatedCompositeFieldContainer[ProfileQuestion]
    recommendations: _containers.RepeatedCompositeFieldContainer[ProfileRecommendation]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    resources: _containers.RepeatedScalarFieldContainer[str]
    issues: _containers.RepeatedCompositeFieldContainer[ProfileValidationIssue]
    valid: bool
    explanations: _containers.RepeatedCompositeFieldContainer[ProfileExplanation]
    digest: str
    catalog_revision: str
    conflicts: _containers.RepeatedCompositeFieldContainer[ProfileConflict]
    outstanding: _containers.RepeatedCompositeFieldContainer[ProfileOutstanding]
    preset: ProfilePreset
    profiles: _containers.RepeatedCompositeFieldContainer[Profile]
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ..., questions: _Optional[_Iterable[_Union[ProfileQuestion, _Mapping]]] = ..., recommendations: _Optional[_Iterable[_Union[ProfileRecommendation, _Mapping]]] = ..., scenarios: _Optional[_Iterable[str]] = ..., resources: _Optional[_Iterable[str]] = ..., issues: _Optional[_Iterable[_Union[ProfileValidationIssue, _Mapping]]] = ..., valid: _Optional[bool] = ..., explanations: _Optional[_Iterable[_Union[ProfileExplanation, _Mapping]]] = ..., digest: _Optional[str] = ..., catalog_revision: _Optional[str] = ..., conflicts: _Optional[_Iterable[_Union[ProfileConflict, _Mapping]]] = ..., outstanding: _Optional[_Iterable[_Union[ProfileOutstanding, _Mapping]]] = ..., preset: _Optional[_Union[ProfilePreset, _Mapping]] = ..., profiles: _Optional[_Iterable[_Union[Profile, _Mapping]]] = ...) -> None: ...
