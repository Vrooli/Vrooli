from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EvaluateRequest(_message.Message):
    __slots__ = ("workspace_id", "profile_revision", "recipe_revision", "excluded_groups", "allergies", "appliances", "recipe_groups", "required_appliances", "allergen_evidence", "method_ids", "methods", "cookware", "tools", "device_functions")
    class AllergenEvidenceEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_REVISION_FIELD_NUMBER: _ClassVar[int]
    RECIPE_REVISION_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_GROUPS_FIELD_NUMBER: _ClassVar[int]
    ALLERGIES_FIELD_NUMBER: _ClassVar[int]
    APPLIANCES_FIELD_NUMBER: _ClassVar[int]
    RECIPE_GROUPS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_APPLIANCES_FIELD_NUMBER: _ClassVar[int]
    ALLERGEN_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    METHOD_IDS_FIELD_NUMBER: _ClassVar[int]
    METHODS_FIELD_NUMBER: _ClassVar[int]
    COOKWARE_FIELD_NUMBER: _ClassVar[int]
    TOOLS_FIELD_NUMBER: _ClassVar[int]
    DEVICE_FUNCTIONS_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    profile_revision: int
    recipe_revision: int
    excluded_groups: _containers.RepeatedScalarFieldContainer[str]
    allergies: _containers.RepeatedScalarFieldContainer[str]
    appliances: _containers.RepeatedScalarFieldContainer[str]
    recipe_groups: _containers.RepeatedScalarFieldContainer[str]
    required_appliances: _containers.RepeatedScalarFieldContainer[str]
    allergen_evidence: _containers.ScalarMap[str, str]
    method_ids: _containers.RepeatedScalarFieldContainer[str]
    methods: _containers.RepeatedCompositeFieldContainer[EligibilityMethod]
    cookware: _containers.RepeatedScalarFieldContainer[str]
    tools: _containers.RepeatedScalarFieldContainer[str]
    device_functions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, workspace_id: _Optional[str] = ..., profile_revision: _Optional[int] = ..., recipe_revision: _Optional[int] = ..., excluded_groups: _Optional[_Iterable[str]] = ..., allergies: _Optional[_Iterable[str]] = ..., appliances: _Optional[_Iterable[str]] = ..., recipe_groups: _Optional[_Iterable[str]] = ..., required_appliances: _Optional[_Iterable[str]] = ..., allergen_evidence: _Optional[_Mapping[str, str]] = ..., method_ids: _Optional[_Iterable[str]] = ..., methods: _Optional[_Iterable[_Union[EligibilityMethod, _Mapping]]] = ..., cookware: _Optional[_Iterable[str]] = ..., tools: _Optional[_Iterable[str]] = ..., device_functions: _Optional[_Iterable[str]] = ...) -> None: ...

class EligibilityMethod(_message.Message):
    __slots__ = ("id", "name", "preparation_mode", "required_appliances", "required_cookware", "required_tools", "required_device_functions")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PREPARATION_MODE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_APPLIANCES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_COOKWARE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_DEVICE_FUNCTIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    preparation_mode: str
    required_appliances: _containers.RepeatedScalarFieldContainer[str]
    required_cookware: _containers.RepeatedScalarFieldContainer[str]
    required_tools: _containers.RepeatedScalarFieldContainer[str]
    required_device_functions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., preparation_mode: _Optional[str] = ..., required_appliances: _Optional[_Iterable[str]] = ..., required_cookware: _Optional[_Iterable[str]] = ..., required_tools: _Optional[_Iterable[str]] = ..., required_device_functions: _Optional[_Iterable[str]] = ...) -> None: ...

class EligibilityReason(_message.Message):
    __slots__ = ("code", "rule", "reference", "message", "viable_method_ids", "method_id")
    CODE_FIELD_NUMBER: _ClassVar[int]
    RULE_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    VIABLE_METHOD_IDS_FIELD_NUMBER: _ClassVar[int]
    METHOD_ID_FIELD_NUMBER: _ClassVar[int]
    code: str
    rule: str
    reference: str
    message: str
    viable_method_ids: _containers.RepeatedScalarFieldContainer[str]
    method_id: str
    def __init__(self, code: _Optional[str] = ..., rule: _Optional[str] = ..., reference: _Optional[str] = ..., message: _Optional[str] = ..., viable_method_ids: _Optional[_Iterable[str]] = ..., method_id: _Optional[str] = ...) -> None: ...

class EligibilityMethodAssessment(_message.Message):
    __slots__ = ("method_id", "method_name", "status", "reasons")
    METHOD_ID_FIELD_NUMBER: _ClassVar[int]
    METHOD_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    method_id: str
    method_name: str
    status: str
    reasons: _containers.RepeatedCompositeFieldContainer[EligibilityReason]
    def __init__(self, method_id: _Optional[str] = ..., method_name: _Optional[str] = ..., status: _Optional[str] = ..., reasons: _Optional[_Iterable[_Union[EligibilityReason, _Mapping]]] = ...) -> None: ...

class EvaluateResponse(_message.Message):
    __slots__ = ("status", "reasons", "profile_revision", "recipe_revision", "evaluator_version", "method_assessments", "viable_method_ids")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_REVISION_FIELD_NUMBER: _ClassVar[int]
    RECIPE_REVISION_FIELD_NUMBER: _ClassVar[int]
    EVALUATOR_VERSION_FIELD_NUMBER: _ClassVar[int]
    METHOD_ASSESSMENTS_FIELD_NUMBER: _ClassVar[int]
    VIABLE_METHOD_IDS_FIELD_NUMBER: _ClassVar[int]
    status: str
    reasons: _containers.RepeatedCompositeFieldContainer[EligibilityReason]
    profile_revision: int
    recipe_revision: int
    evaluator_version: str
    method_assessments: _containers.RepeatedCompositeFieldContainer[EligibilityMethodAssessment]
    viable_method_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[str] = ..., reasons: _Optional[_Iterable[_Union[EligibilityReason, _Mapping]]] = ..., profile_revision: _Optional[int] = ..., recipe_revision: _Optional[int] = ..., evaluator_version: _Optional[str] = ..., method_assessments: _Optional[_Iterable[_Union[EligibilityMethodAssessment, _Mapping]]] = ..., viable_method_ids: _Optional[_Iterable[str]] = ...) -> None: ...
