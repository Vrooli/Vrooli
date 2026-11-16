from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StageGenerationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class CatchUpGenerationRequest(_message.Message):
    __slots__ = ("operation_id", "generation_id")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    GENERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    generation_id: str
    def __init__(self, operation_id: _Optional[str] = ..., generation_id: _Optional[str] = ...) -> None: ...

class ValidateGenerationRequest(_message.Message):
    __slots__ = ("generation_id",)
    GENERATION_ID_FIELD_NUMBER: _ClassVar[int]
    generation_id: str
    def __init__(self, generation_id: _Optional[str] = ...) -> None: ...

class PromoteGenerationRequest(_message.Message):
    __slots__ = ("operation_id", "generation_id")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    GENERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    generation_id: str
    def __init__(self, operation_id: _Optional[str] = ..., generation_id: _Optional[str] = ...) -> None: ...

class RollbackGenerationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class GetGenerationOperationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class ResumeGenerationOperationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class GetActiveGenerationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GenerationControlResponse(_message.Message):
    __slots__ = ("generation", "operation", "validation", "active_found")
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FOUND_FIELD_NUMBER: _ClassVar[int]
    generation: SQLiteGeneration
    operation: GenerationOperation
    validation: GenerationValidation
    active_found: bool
    def __init__(self, generation: _Optional[_Union[SQLiteGeneration, _Mapping]] = ..., operation: _Optional[_Union[GenerationOperation, _Mapping]] = ..., validation: _Optional[_Union[GenerationValidation, _Mapping]] = ..., active_found: _Optional[bool] = ...) -> None: ...

class SQLiteGeneration(_message.Message):
    __slots__ = ("generation_id", "recipe_id", "dimensions", "state", "snapshot_id", "snapshot_watermark", "catchup_watermark", "snapshot_cursor", "predecessor_id", "rollback_expires_at", "protected", "version")
    GENERATION_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_WATERMARK_FIELD_NUMBER: _ClassVar[int]
    CATCHUP_WATERMARK_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_CURSOR_FIELD_NUMBER: _ClassVar[int]
    PREDECESSOR_ID_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    PROTECTED_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    generation_id: str
    recipe_id: str
    dimensions: int
    state: str
    snapshot_id: str
    snapshot_watermark: int
    catchup_watermark: int
    snapshot_cursor: int
    predecessor_id: str
    rollback_expires_at: str
    protected: bool
    version: int
    def __init__(self, generation_id: _Optional[str] = ..., recipe_id: _Optional[str] = ..., dimensions: _Optional[int] = ..., state: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., snapshot_watermark: _Optional[int] = ..., catchup_watermark: _Optional[int] = ..., snapshot_cursor: _Optional[int] = ..., predecessor_id: _Optional[str] = ..., rollback_expires_at: _Optional[str] = ..., protected: _Optional[bool] = ..., version: _Optional[int] = ...) -> None: ...

class GenerationOperation(_message.Message):
    __slots__ = ("operation_id", "operation", "generation_id", "recipe_id", "state", "receipt_id", "peak_bytes", "started_at", "completed_at", "error")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    GENERATION_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    PEAK_BYTES_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    operation: str
    generation_id: str
    recipe_id: str
    state: str
    receipt_id: str
    peak_bytes: int
    started_at: str
    completed_at: str
    error: str
    def __init__(self, operation_id: _Optional[str] = ..., operation: _Optional[str] = ..., generation_id: _Optional[str] = ..., recipe_id: _Optional[str] = ..., state: _Optional[str] = ..., receipt_id: _Optional[str] = ..., peak_bytes: _Optional[int] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class GenerationValidation(_message.Message):
    __slots__ = ("generation_id", "state", "ready_for_promotion", "expected_points", "actual_points", "missing_points", "orphaned_points", "invalid_points", "pending_changes", "catchup_watermark", "current_watermark", "recipe_id", "dimensions")
    GENERATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    READY_FOR_PROMOTION_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_POINTS_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_POINTS_FIELD_NUMBER: _ClassVar[int]
    MISSING_POINTS_FIELD_NUMBER: _ClassVar[int]
    ORPHANED_POINTS_FIELD_NUMBER: _ClassVar[int]
    INVALID_POINTS_FIELD_NUMBER: _ClassVar[int]
    PENDING_CHANGES_FIELD_NUMBER: _ClassVar[int]
    CATCHUP_WATERMARK_FIELD_NUMBER: _ClassVar[int]
    CURRENT_WATERMARK_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    generation_id: str
    state: str
    ready_for_promotion: bool
    expected_points: int
    actual_points: int
    missing_points: int
    orphaned_points: int
    invalid_points: int
    pending_changes: int
    catchup_watermark: int
    current_watermark: int
    recipe_id: str
    dimensions: int
    def __init__(self, generation_id: _Optional[str] = ..., state: _Optional[str] = ..., ready_for_promotion: _Optional[bool] = ..., expected_points: _Optional[int] = ..., actual_points: _Optional[int] = ..., missing_points: _Optional[int] = ..., orphaned_points: _Optional[int] = ..., invalid_points: _Optional[int] = ..., pending_changes: _Optional[int] = ..., catchup_watermark: _Optional[int] = ..., current_watermark: _Optional[int] = ..., recipe_id: _Optional[str] = ..., dimensions: _Optional[int] = ...) -> None: ...
