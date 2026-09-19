from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StartReviewRequest(_message.Message):
    __slots__ = ("repository_id", "scenario_name", "checks", "details", "thresholds")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    THRESHOLDS_FIELD_NUMBER: _ClassVar[int]
    repository_id: int
    scenario_name: str
    checks: _containers.RepeatedScalarFieldContainer[str]
    details: int
    thresholds: ReadinessThresholds
    def __init__(self, repository_id: _Optional[int] = ..., scenario_name: _Optional[str] = ..., checks: _Optional[_Iterable[str]] = ..., details: _Optional[int] = ..., thresholds: _Optional[_Union[ReadinessThresholds, _Mapping]] = ...) -> None: ...

class ReadinessThresholds(_message.Message):
    __slots__ = ("code_quality_min_score", "test_min_pass_rate", "max_blocking_violations", "max_warnings", "require_screenshots", "require_tests")
    CODE_QUALITY_MIN_SCORE_FIELD_NUMBER: _ClassVar[int]
    TEST_MIN_PASS_RATE_FIELD_NUMBER: _ClassVar[int]
    MAX_BLOCKING_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    MAX_WARNINGS_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_SCREENSHOTS_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_TESTS_FIELD_NUMBER: _ClassVar[int]
    code_quality_min_score: float
    test_min_pass_rate: float
    max_blocking_violations: int
    max_warnings: int
    require_screenshots: bool
    require_tests: bool
    def __init__(self, code_quality_min_score: _Optional[float] = ..., test_min_pass_rate: _Optional[float] = ..., max_blocking_violations: _Optional[int] = ..., max_warnings: _Optional[int] = ..., require_screenshots: _Optional[bool] = ..., require_tests: _Optional[bool] = ...) -> None: ...

class StartReviewResponse(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...
