from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Variable(_message.Message):
    __slots__ = ("name", "placeholder", "occurrences")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PLACEHOLDER_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    name: str
    placeholder: str
    occurrences: int
    def __init__(self, name: _Optional[str] = ..., placeholder: _Optional[str] = ..., occurrences: _Optional[int] = ...) -> None: ...

class Skill(_message.Message):
    __slots__ = ("id", "file", "name", "description", "content", "modes", "tags", "icon", "target_tool_id", "default_scope", "target_dimensions", "programmatic_home", "draft", "folder", "skill_dir", "content_path", "created_at", "updated_at", "revision", "content_hash", "usage_count", "last_used", "effectiveness_rating", "variables")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MODES_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    TARGET_TOOL_ID_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SCOPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    PROGRAMMATIC_HOME_FIELD_NUMBER: _ClassVar[int]
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    SKILL_DIR_FIELD_NUMBER: _ClassVar[int]
    CONTENT_PATH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    USAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVENESS_RATING_FIELD_NUMBER: _ClassVar[int]
    VARIABLES_FIELD_NUMBER: _ClassVar[int]
    id: str
    file: str
    name: str
    description: str
    content: str
    modes: _containers.RepeatedScalarFieldContainer[str]
    tags: _containers.RepeatedScalarFieldContainer[str]
    icon: str
    target_tool_id: str
    default_scope: str
    target_dimensions: _containers.RepeatedScalarFieldContainer[str]
    programmatic_home: str
    draft: bool
    folder: str
    skill_dir: str
    content_path: str
    created_at: str
    updated_at: str
    revision: int
    content_hash: str
    usage_count: int
    last_used: str
    effectiveness_rating: int
    variables: _containers.RepeatedCompositeFieldContainer[Variable]
    def __init__(self, id: _Optional[str] = ..., file: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., content: _Optional[str] = ..., modes: _Optional[_Iterable[str]] = ..., tags: _Optional[_Iterable[str]] = ..., icon: _Optional[str] = ..., target_tool_id: _Optional[str] = ..., default_scope: _Optional[str] = ..., target_dimensions: _Optional[_Iterable[str]] = ..., programmatic_home: _Optional[str] = ..., draft: _Optional[bool] = ..., folder: _Optional[str] = ..., skill_dir: _Optional[str] = ..., content_path: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., revision: _Optional[int] = ..., content_hash: _Optional[str] = ..., usage_count: _Optional[int] = ..., last_used: _Optional[str] = ..., effectiveness_rating: _Optional[int] = ..., variables: _Optional[_Iterable[_Union[Variable, _Mapping]]] = ...) -> None: ...

class ListSkillsRequest(_message.Message):
    __slots__ = ("folder", "tag", "modes", "without_programmatic_home")
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    MODES_FIELD_NUMBER: _ClassVar[int]
    WITHOUT_PROGRAMMATIC_HOME_FIELD_NUMBER: _ClassVar[int]
    folder: str
    tag: str
    modes: _containers.RepeatedScalarFieldContainer[str]
    without_programmatic_home: bool
    def __init__(self, folder: _Optional[str] = ..., tag: _Optional[str] = ..., modes: _Optional[_Iterable[str]] = ..., without_programmatic_home: _Optional[bool] = ...) -> None: ...

class ListSkillsResponse(_message.Message):
    __slots__ = ("skills",)
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    skills: _containers.RepeatedCompositeFieldContainer[Skill]
    def __init__(self, skills: _Optional[_Iterable[_Union[Skill, _Mapping]]] = ...) -> None: ...

class GetSkillRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetSkillResponse(_message.Message):
    __slots__ = ("skill",)
    SKILL_FIELD_NUMBER: _ClassVar[int]
    skill: Skill
    def __init__(self, skill: _Optional[_Union[Skill, _Mapping]] = ...) -> None: ...

class ReadSkillsRequest(_message.Message):
    __slots__ = ("identifiers", "resolve", "allow_missing", "output", "format", "variables", "with_scope", "scope", "experiment_id", "variant_id", "variant_policy", "source")
    class VariablesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    IDENTIFIERS_FIELD_NUMBER: _ClassVar[int]
    RESOLVE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_MISSING_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    VARIABLES_FIELD_NUMBER: _ClassVar[int]
    WITH_SCOPE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_POLICY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    identifiers: _containers.RepeatedScalarFieldContainer[str]
    resolve: str
    allow_missing: bool
    output: str
    format: str
    variables: _containers.ScalarMap[str, str]
    with_scope: bool
    scope: str
    experiment_id: str
    variant_id: str
    variant_policy: str
    source: str
    def __init__(self, identifiers: _Optional[_Iterable[str]] = ..., resolve: _Optional[str] = ..., allow_missing: _Optional[bool] = ..., output: _Optional[str] = ..., format: _Optional[str] = ..., variables: _Optional[_Mapping[str, str]] = ..., with_scope: _Optional[bool] = ..., scope: _Optional[str] = ..., experiment_id: _Optional[str] = ..., variant_id: _Optional[str] = ..., variant_policy: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class ReadIssue(_message.Message):
    __slots__ = ("identifier", "reason")
    IDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    identifier: str
    reason: str
    def __init__(self, identifier: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ReadCandidate(_message.Message):
    __slots__ = ("id", "name", "file", "folder")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    file: str
    folder: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., file: _Optional[str] = ..., folder: _Optional[str] = ...) -> None: ...

class ReadAmbiguous(_message.Message):
    __slots__ = ("identifier", "candidates")
    IDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    identifier: str
    candidates: _containers.RepeatedCompositeFieldContainer[ReadCandidate]
    def __init__(self, identifier: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[ReadCandidate, _Mapping]]] = ...) -> None: ...

class ReadSkillsResponse(_message.Message):
    __slots__ = ("skills", "combined", "combined_hash", "skill_count", "total_tokens", "format", "missing", "ambiguous", "resolve", "output", "scope_skill", "selected_variant_id", "experiment_id")
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    COMBINED_FIELD_NUMBER: _ClassVar[int]
    COMBINED_HASH_FIELD_NUMBER: _ClassVar[int]
    SKILL_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    AMBIGUOUS_FIELD_NUMBER: _ClassVar[int]
    RESOLVE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_SKILL_FIELD_NUMBER: _ClassVar[int]
    SELECTED_VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    skills: _containers.RepeatedCompositeFieldContainer[Skill]
    combined: str
    combined_hash: str
    skill_count: int
    total_tokens: int
    format: str
    missing: _containers.RepeatedCompositeFieldContainer[ReadIssue]
    ambiguous: _containers.RepeatedCompositeFieldContainer[ReadAmbiguous]
    resolve: str
    output: str
    scope_skill: Skill
    selected_variant_id: str
    experiment_id: str
    def __init__(self, skills: _Optional[_Iterable[_Union[Skill, _Mapping]]] = ..., combined: _Optional[str] = ..., combined_hash: _Optional[str] = ..., skill_count: _Optional[int] = ..., total_tokens: _Optional[int] = ..., format: _Optional[str] = ..., missing: _Optional[_Iterable[_Union[ReadIssue, _Mapping]]] = ..., ambiguous: _Optional[_Iterable[_Union[ReadAmbiguous, _Mapping]]] = ..., resolve: _Optional[str] = ..., output: _Optional[str] = ..., scope_skill: _Optional[_Union[Skill, _Mapping]] = ..., selected_variant_id: _Optional[str] = ..., experiment_id: _Optional[str] = ...) -> None: ...

class CreateSkillRequest(_message.Message):
    __slots__ = ("id", "name", "description", "content", "modes", "tags", "icon", "target_tool_id", "default_scope", "target_dimensions", "programmatic_home", "draft", "folder")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MODES_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    TARGET_TOOL_ID_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SCOPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    PROGRAMMATIC_HOME_FIELD_NUMBER: _ClassVar[int]
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    content: str
    modes: _containers.RepeatedScalarFieldContainer[str]
    tags: _containers.RepeatedScalarFieldContainer[str]
    icon: str
    target_tool_id: str
    default_scope: str
    target_dimensions: _containers.RepeatedScalarFieldContainer[str]
    programmatic_home: str
    draft: bool
    folder: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., content: _Optional[str] = ..., modes: _Optional[_Iterable[str]] = ..., tags: _Optional[_Iterable[str]] = ..., icon: _Optional[str] = ..., target_tool_id: _Optional[str] = ..., default_scope: _Optional[str] = ..., target_dimensions: _Optional[_Iterable[str]] = ..., programmatic_home: _Optional[str] = ..., draft: _Optional[bool] = ..., folder: _Optional[str] = ...) -> None: ...

class CreateSkillResponse(_message.Message):
    __slots__ = ("skill",)
    SKILL_FIELD_NUMBER: _ClassVar[int]
    skill: Skill
    def __init__(self, skill: _Optional[_Union[Skill, _Mapping]] = ...) -> None: ...

class UpdateSkillRequest(_message.Message):
    __slots__ = ("id", "file", "name", "description", "content", "modes", "replace_modes", "tags", "replace_tags", "icon", "target_tool_id", "default_scope", "target_dimensions", "replace_target_dimensions", "programmatic_home", "clear_programmatic_home", "draft", "folder")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MODES_FIELD_NUMBER: _ClassVar[int]
    REPLACE_MODES_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    REPLACE_TAGS_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    TARGET_TOOL_ID_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SCOPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    REPLACE_TARGET_DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    PROGRAMMATIC_HOME_FIELD_NUMBER: _ClassVar[int]
    CLEAR_PROGRAMMATIC_HOME_FIELD_NUMBER: _ClassVar[int]
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    id: str
    file: str
    name: str
    description: str
    content: str
    modes: _containers.RepeatedScalarFieldContainer[str]
    replace_modes: bool
    tags: _containers.RepeatedScalarFieldContainer[str]
    replace_tags: bool
    icon: str
    target_tool_id: str
    default_scope: str
    target_dimensions: _containers.RepeatedScalarFieldContainer[str]
    replace_target_dimensions: bool
    programmatic_home: str
    clear_programmatic_home: bool
    draft: bool
    folder: str
    def __init__(self, id: _Optional[str] = ..., file: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., content: _Optional[str] = ..., modes: _Optional[_Iterable[str]] = ..., replace_modes: _Optional[bool] = ..., tags: _Optional[_Iterable[str]] = ..., replace_tags: _Optional[bool] = ..., icon: _Optional[str] = ..., target_tool_id: _Optional[str] = ..., default_scope: _Optional[str] = ..., target_dimensions: _Optional[_Iterable[str]] = ..., replace_target_dimensions: _Optional[bool] = ..., programmatic_home: _Optional[str] = ..., clear_programmatic_home: _Optional[bool] = ..., draft: _Optional[bool] = ..., folder: _Optional[str] = ...) -> None: ...

class UpdateSkillResponse(_message.Message):
    __slots__ = ("skill",)
    SKILL_FIELD_NUMBER: _ClassVar[int]
    skill: Skill
    def __init__(self, skill: _Optional[_Union[Skill, _Mapping]] = ...) -> None: ...

class DeleteSkillRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteSkillResponse(_message.Message):
    __slots__ = ("id", "deleted")
    ID_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    id: str
    deleted: bool
    def __init__(self, id: _Optional[str] = ..., deleted: _Optional[bool] = ...) -> None: ...

class SyncSkillsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SyncSkillsResponse(_message.Message):
    __slots__ = ("skills", "last_updated", "hash")
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATED_FIELD_NUMBER: _ClassVar[int]
    HASH_FIELD_NUMBER: _ClassVar[int]
    skills: _containers.RepeatedCompositeFieldContainer[Skill]
    last_updated: str
    hash: str
    def __init__(self, skills: _Optional[_Iterable[_Union[Skill, _Mapping]]] = ..., last_updated: _Optional[str] = ..., hash: _Optional[str] = ...) -> None: ...

class RateSkillRequest(_message.Message):
    __slots__ = ("id", "rating", "notes")
    ID_FIELD_NUMBER: _ClassVar[int]
    RATING_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    rating: int
    notes: str
    def __init__(self, id: _Optional[str] = ..., rating: _Optional[int] = ..., notes: _Optional[str] = ...) -> None: ...

class RateSkillResponse(_message.Message):
    __slots__ = ("id", "rating", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    RATING_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    rating: int
    status: str
    def __init__(self, id: _Optional[str] = ..., rating: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...

class RecordSkillUsageRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RecordSkillUsageResponse(_message.Message):
    __slots__ = ("id", "status", "usage_count", "last_used")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    USAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    usage_count: int
    last_used: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., usage_count: _Optional[int] = ..., last_used: _Optional[str] = ...) -> None: ...

class SkillVersion(_message.Message):
    __slots__ = ("version", "content", "name", "updated_at", "created_by")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    version: int
    content: str
    name: str
    updated_at: str
    created_by: str
    def __init__(self, version: _Optional[int] = ..., content: _Optional[str] = ..., name: _Optional[str] = ..., updated_at: _Optional[str] = ..., created_by: _Optional[str] = ...) -> None: ...

class ListSkillVersionsRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ListSkillVersionsResponse(_message.Message):
    __slots__ = ("skill_id", "current", "versions")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    current: int
    versions: _containers.RepeatedCompositeFieldContainer[SkillVersion]
    def __init__(self, skill_id: _Optional[str] = ..., current: _Optional[int] = ..., versions: _Optional[_Iterable[_Union[SkillVersion, _Mapping]]] = ...) -> None: ...

class RevertSkillRequest(_message.Message):
    __slots__ = ("id", "version")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: int
    def __init__(self, id: _Optional[str] = ..., version: _Optional[int] = ...) -> None: ...

class RevertSkillResponse(_message.Message):
    __slots__ = ("skill_id", "reverted_to", "new_version", "restored_at")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    REVERTED_TO_FIELD_NUMBER: _ClassVar[int]
    NEW_VERSION_FIELD_NUMBER: _ClassVar[int]
    RESTORED_AT_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    reverted_to: int
    new_version: int
    restored_at: str
    def __init__(self, skill_id: _Optional[str] = ..., reverted_to: _Optional[int] = ..., new_version: _Optional[int] = ..., restored_at: _Optional[str] = ...) -> None: ...

class SkillVariant(_message.Message):
    __slots__ = ("id", "skill_id", "name", "description", "content", "created_at", "updated_at", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    skill_id: str
    name: str
    description: str
    content: str
    created_at: str
    updated_at: str
    revision: int
    def __init__(self, id: _Optional[str] = ..., skill_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., content: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class ListSkillVariantsRequest(_message.Message):
    __slots__ = ("skill_id",)
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    def __init__(self, skill_id: _Optional[str] = ...) -> None: ...

class ListSkillVariantsResponse(_message.Message):
    __slots__ = ("variants",)
    VARIANTS_FIELD_NUMBER: _ClassVar[int]
    variants: _containers.RepeatedCompositeFieldContainer[SkillVariant]
    def __init__(self, variants: _Optional[_Iterable[_Union[SkillVariant, _Mapping]]] = ...) -> None: ...

class GetSkillVariantRequest(_message.Message):
    __slots__ = ("skill_id", "variant_id")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    variant_id: str
    def __init__(self, skill_id: _Optional[str] = ..., variant_id: _Optional[str] = ...) -> None: ...

class GetSkillVariantResponse(_message.Message):
    __slots__ = ("variant",)
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    variant: SkillVariant
    def __init__(self, variant: _Optional[_Union[SkillVariant, _Mapping]] = ...) -> None: ...

class CreateSkillVariantRequest(_message.Message):
    __slots__ = ("skill_id", "id", "name", "description", "content")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    id: str
    name: str
    description: str
    content: str
    def __init__(self, skill_id: _Optional[str] = ..., id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class CreateSkillVariantResponse(_message.Message):
    __slots__ = ("variant",)
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    variant: SkillVariant
    def __init__(self, variant: _Optional[_Union[SkillVariant, _Mapping]] = ...) -> None: ...

class UpdateSkillVariantRequest(_message.Message):
    __slots__ = ("skill_id", "variant_id", "name", "description", "content")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    variant_id: str
    name: str
    description: str
    content: str
    def __init__(self, skill_id: _Optional[str] = ..., variant_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class UpdateSkillVariantResponse(_message.Message):
    __slots__ = ("variant",)
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    variant: SkillVariant
    def __init__(self, variant: _Optional[_Union[SkillVariant, _Mapping]] = ...) -> None: ...

class DeleteSkillVariantRequest(_message.Message):
    __slots__ = ("skill_id", "variant_id")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    variant_id: str
    def __init__(self, skill_id: _Optional[str] = ..., variant_id: _Optional[str] = ...) -> None: ...

class DeleteSkillVariantResponse(_message.Message):
    __slots__ = ("skill_id", "variant_id", "deleted")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    variant_id: str
    deleted: bool
    def __init__(self, skill_id: _Optional[str] = ..., variant_id: _Optional[str] = ..., deleted: _Optional[bool] = ...) -> None: ...

class ImportSkillRequest(_message.Message):
    __slots__ = ("source_dir", "source_url", "commit", "license", "checksum", "imported_by", "upstream_version", "id")
    SOURCE_DIR_FIELD_NUMBER: _ClassVar[int]
    SOURCE_URL_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    IMPORTED_BY_FIELD_NUMBER: _ClassVar[int]
    UPSTREAM_VERSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    source_dir: str
    source_url: str
    commit: str
    license: str
    checksum: str
    imported_by: str
    upstream_version: str
    id: str
    def __init__(self, source_dir: _Optional[str] = ..., source_url: _Optional[str] = ..., commit: _Optional[str] = ..., license: _Optional[str] = ..., checksum: _Optional[str] = ..., imported_by: _Optional[str] = ..., upstream_version: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class ImportSkillResponse(_message.Message):
    __slots__ = ("id", "pack", "status", "checksum", "review_verdict", "imported_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PACK_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    REVIEW_VERDICT_FIELD_NUMBER: _ClassVar[int]
    IMPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    pack: str
    status: str
    checksum: str
    review_verdict: str
    imported_at: str
    def __init__(self, id: _Optional[str] = ..., pack: _Optional[str] = ..., status: _Optional[str] = ..., checksum: _Optional[str] = ..., review_verdict: _Optional[str] = ..., imported_at: _Optional[str] = ...) -> None: ...

class ReviewImportedSkillRequest(_message.Message):
    __slots__ = ("id", "reviewer", "verdict")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVIEWER_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    id: str
    reviewer: str
    verdict: str
    def __init__(self, id: _Optional[str] = ..., reviewer: _Optional[str] = ..., verdict: _Optional[str] = ...) -> None: ...

class ReviewImportedSkillResponse(_message.Message):
    __slots__ = ("id", "status", "verdict", "reviewer", "reviewed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REVIEWER_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    verdict: str
    reviewer: str
    reviewed_at: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., verdict: _Optional[str] = ..., reviewer: _Optional[str] = ..., reviewed_at: _Optional[str] = ...) -> None: ...

class ReportImportedSkillStalenessRequest(_message.Message):
    __slots__ = ("id", "upstream_version")
    ID_FIELD_NUMBER: _ClassVar[int]
    UPSTREAM_VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    upstream_version: str
    def __init__(self, id: _Optional[str] = ..., upstream_version: _Optional[str] = ...) -> None: ...

class ReportImportedSkillStalenessResponse(_message.Message):
    __slots__ = ("id", "recorded_version", "current_version", "stale")
    ID_FIELD_NUMBER: _ClassVar[int]
    RECORDED_VERSION_FIELD_NUMBER: _ClassVar[int]
    CURRENT_VERSION_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    id: str
    recorded_version: str
    current_version: str
    stale: bool
    def __init__(self, id: _Optional[str] = ..., recorded_version: _Optional[str] = ..., current_version: _Optional[str] = ..., stale: _Optional[bool] = ...) -> None: ...
