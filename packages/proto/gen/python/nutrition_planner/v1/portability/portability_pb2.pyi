from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ExportRecipesRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ExportRecipesResponse(_message.Message):
    __slots__ = ("format", "schema_version", "content_json", "omissions")
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_JSON_FIELD_NUMBER: _ClassVar[int]
    OMISSIONS_FIELD_NUMBER: _ClassVar[int]
    format: str
    schema_version: int
    content_json: str
    omissions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, format: _Optional[str] = ..., schema_version: _Optional[int] = ..., content_json: _Optional[str] = ..., omissions: _Optional[_Iterable[str]] = ...) -> None: ...

class ExportWorkspaceRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ExportWorkspaceResponse(_message.Message):
    __slots__ = ("format", "schema_version", "content_json", "omissions", "plan_revision")
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_JSON_FIELD_NUMBER: _ClassVar[int]
    OMISSIONS_FIELD_NUMBER: _ClassVar[int]
    PLAN_REVISION_FIELD_NUMBER: _ClassVar[int]
    format: str
    schema_version: int
    content_json: str
    omissions: _containers.RepeatedScalarFieldContainer[str]
    plan_revision: int
    def __init__(self, format: _Optional[str] = ..., schema_version: _Optional[int] = ..., content_json: _Optional[str] = ..., omissions: _Optional[_Iterable[str]] = ..., plan_revision: _Optional[int] = ...) -> None: ...

class PreviewWorkspaceImportRequest(_message.Message):
    __slots__ = ("workspace_id", "content_json")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_JSON_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    content_json: str
    def __init__(self, workspace_id: _Optional[str] = ..., content_json: _Optional[str] = ...) -> None: ...

class PreviewWorkspaceImportResponse(_message.Message):
    __slots__ = ("valid", "format", "schema_version", "record_count", "record_kinds", "omissions", "errors")
    VALID_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RECORD_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECORD_KINDS_FIELD_NUMBER: _ClassVar[int]
    OMISSIONS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    format: str
    schema_version: int
    record_count: int
    record_kinds: _containers.RepeatedScalarFieldContainer[str]
    omissions: _containers.RepeatedScalarFieldContainer[str]
    errors: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, valid: _Optional[bool] = ..., format: _Optional[str] = ..., schema_version: _Optional[int] = ..., record_count: _Optional[int] = ..., record_kinds: _Optional[_Iterable[str]] = ..., omissions: _Optional[_Iterable[str]] = ..., errors: _Optional[_Iterable[str]] = ...) -> None: ...

class ApplyWorkspaceImportRequest(_message.Message):
    __slots__ = ("workspace_id", "expected_workspace_revision", "content_json", "idempotency_key")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_WORKSPACE_REVISION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_JSON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    expected_workspace_revision: int
    content_json: str
    idempotency_key: str
    def __init__(self, workspace_id: _Optional[str] = ..., expected_workspace_revision: _Optional[int] = ..., content_json: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ApplyWorkspaceImportResponse(_message.Message):
    __slots__ = ("workspace_revision", "recipes_applied", "checkpoint_id")
    WORKSPACE_REVISION_FIELD_NUMBER: _ClassVar[int]
    RECIPES_APPLIED_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_revision: int
    recipes_applied: int
    checkpoint_id: str
    def __init__(self, workspace_revision: _Optional[int] = ..., recipes_applied: _Optional[int] = ..., checkpoint_id: _Optional[str] = ...) -> None: ...

class PreviewRecipesImportRequest(_message.Message):
    __slots__ = ("workspace_id", "content_json")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_JSON_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    content_json: str
    def __init__(self, workspace_id: _Optional[str] = ..., content_json: _Optional[str] = ...) -> None: ...

class PreviewRecipesImportResponse(_message.Message):
    __slots__ = ("valid", "format", "schema_version", "recipe_count", "duplicate_count", "conflict_count", "errors")
    VALID_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RECIPE_COUNT_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CONFLICT_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    format: str
    schema_version: int
    recipe_count: int
    duplicate_count: int
    conflict_count: int
    errors: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, valid: _Optional[bool] = ..., format: _Optional[str] = ..., schema_version: _Optional[int] = ..., recipe_count: _Optional[int] = ..., duplicate_count: _Optional[int] = ..., conflict_count: _Optional[int] = ..., errors: _Optional[_Iterable[str]] = ...) -> None: ...

class ApplyRecipesImportRequest(_message.Message):
    __slots__ = ("workspace_id", "expected_workspace_revision", "content_json", "idempotency_key", "conflict_policy")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_WORKSPACE_REVISION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_JSON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    CONFLICT_POLICY_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    expected_workspace_revision: int
    content_json: str
    idempotency_key: str
    conflict_policy: str
    def __init__(self, workspace_id: _Optional[str] = ..., expected_workspace_revision: _Optional[int] = ..., content_json: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., conflict_policy: _Optional[str] = ...) -> None: ...

class ApplyRecipesImportResponse(_message.Message):
    __slots__ = ("workspace_revision", "recipes_applied", "recipes_skipped", "remapped_ids")
    WORKSPACE_REVISION_FIELD_NUMBER: _ClassVar[int]
    RECIPES_APPLIED_FIELD_NUMBER: _ClassVar[int]
    RECIPES_SKIPPED_FIELD_NUMBER: _ClassVar[int]
    REMAPPED_IDS_FIELD_NUMBER: _ClassVar[int]
    workspace_revision: int
    recipes_applied: int
    recipes_skipped: int
    remapped_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, workspace_revision: _Optional[int] = ..., recipes_applied: _Optional[int] = ..., recipes_skipped: _Optional[int] = ..., remapped_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ExportGroceriesCSVRequest(_message.Message):
    __slots__ = ("workspace_id", "expected_revision")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    expected_revision: int
    def __init__(self, workspace_id: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class ExportGroceriesCSVResponse(_message.Message):
    __slots__ = ("filename", "content_csv", "revision")
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_CSV_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    filename: str
    content_csv: str
    revision: int
    def __init__(self, filename: _Optional[str] = ..., content_csv: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class ExportRecipePDFRequest(_message.Message):
    __slots__ = ("workspace_id", "recipe_id", "page_size")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    recipe_id: str
    page_size: str
    def __init__(self, workspace_id: _Optional[str] = ..., recipe_id: _Optional[str] = ..., page_size: _Optional[str] = ...) -> None: ...

class ExportWeeklyPDFRequest(_message.Message):
    __slots__ = ("workspace_id", "expected_revision", "page_size")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    expected_revision: int
    page_size: str
    def __init__(self, workspace_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., page_size: _Optional[str] = ...) -> None: ...

class ExportPDFResponse(_message.Message):
    __slots__ = ("filename", "content", "revision")
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    filename: str
    content: bytes
    revision: int
    def __init__(self, filename: _Optional[str] = ..., content: _Optional[bytes] = ..., revision: _Optional[int] = ...) -> None: ...
