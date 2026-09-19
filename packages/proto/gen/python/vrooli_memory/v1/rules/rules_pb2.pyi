from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Rule(_message.Message):
    __slots__ = ("id", "scope", "priority", "facet_id", "source_runtime", "kind", "source_path_glob", "body_pattern", "enabled")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RUNTIME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_GLOB_FIELD_NUMBER: _ClassVar[int]
    BODY_PATTERN_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    id: str
    scope: str
    priority: int
    facet_id: str
    source_runtime: str
    kind: str
    source_path_glob: str
    body_pattern: str
    enabled: bool
    def __init__(self, id: _Optional[str] = ..., scope: _Optional[str] = ..., priority: _Optional[int] = ..., facet_id: _Optional[str] = ..., source_runtime: _Optional[str] = ..., kind: _Optional[str] = ..., source_path_glob: _Optional[str] = ..., body_pattern: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class ListRulesRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class ListRulesResponse(_message.Message):
    __slots__ = ("rules",)
    RULES_FIELD_NUMBER: _ClassVar[int]
    rules: _containers.RepeatedCompositeFieldContainer[Rule]
    def __init__(self, rules: _Optional[_Iterable[_Union[Rule, _Mapping]]] = ...) -> None: ...

class CreateRuleRequest(_message.Message):
    __slots__ = ("rule",)
    RULE_FIELD_NUMBER: _ClassVar[int]
    rule: Rule
    def __init__(self, rule: _Optional[_Union[Rule, _Mapping]] = ...) -> None: ...

class CreateRuleResponse(_message.Message):
    __slots__ = ("rule",)
    RULE_FIELD_NUMBER: _ClassVar[int]
    rule: Rule
    def __init__(self, rule: _Optional[_Union[Rule, _Mapping]] = ...) -> None: ...

class DryRunRuleRequest(_message.Message):
    __slots__ = ("rule_id", "scope")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    scope: str
    def __init__(self, rule_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class DryRunRuleResponse(_message.Message):
    __slots__ = ("rule_id", "corpus_fingerprint", "match_count", "samples")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    CORPUS_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    MATCH_COUNT_FIELD_NUMBER: _ClassVar[int]
    SAMPLES_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    corpus_fingerprint: str
    match_count: int
    samples: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, rule_id: _Optional[str] = ..., corpus_fingerprint: _Optional[str] = ..., match_count: _Optional[int] = ..., samples: _Optional[_Iterable[str]] = ...) -> None: ...

class EnableRuleRequest(_message.Message):
    __slots__ = ("rule_id", "scope")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    scope: str
    def __init__(self, rule_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class EnableRuleResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RevertRuleRequest(_message.Message):
    __slots__ = ("rule_id", "scope")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    scope: str
    def __init__(self, rule_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class RevertRuleResponse(_message.Message):
    __slots__ = ("restored_count",)
    RESTORED_COUNT_FIELD_NUMBER: _ClassVar[int]
    restored_count: int
    def __init__(self, restored_count: _Optional[int] = ...) -> None: ...

class RefacetCorpusRequest(_message.Message):
    __slots__ = ("scope", "after_entry_id", "limit")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    AFTER_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scope: str
    after_entry_id: str
    limit: int
    def __init__(self, scope: _Optional[str] = ..., after_entry_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class RefacetCorpusResponse(_message.Message):
    __slots__ = ("total", "assigned", "rule_assigned", "classified", "failed", "next_entry_id", "complete")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    ASSIGNED_FIELD_NUMBER: _ClassVar[int]
    RULE_ASSIGNED_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    NEXT_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    total: int
    assigned: int
    rule_assigned: int
    classified: int
    failed: int
    next_entry_id: str
    complete: bool
    def __init__(self, total: _Optional[int] = ..., assigned: _Optional[int] = ..., rule_assigned: _Optional[int] = ..., classified: _Optional[int] = ..., failed: _Optional[int] = ..., next_entry_id: _Optional[str] = ..., complete: _Optional[bool] = ...) -> None: ...

class MeasureDistributionRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class MeasureDistributionResponse(_message.Message):
    __slots__ = ("scope", "total", "rule_matched", "classifier_tail", "rule_coverage", "classifier_tail_by_facet", "ceiling_percent", "max_tail_facet", "max_tail_percent", "within_ceiling")
    class RuleCoverageEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class ClassifierTailByFacetEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    RULE_MATCHED_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIER_TAIL_FIELD_NUMBER: _ClassVar[int]
    RULE_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIER_TAIL_BY_FACET_FIELD_NUMBER: _ClassVar[int]
    CEILING_PERCENT_FIELD_NUMBER: _ClassVar[int]
    MAX_TAIL_FACET_FIELD_NUMBER: _ClassVar[int]
    MAX_TAIL_PERCENT_FIELD_NUMBER: _ClassVar[int]
    WITHIN_CEILING_FIELD_NUMBER: _ClassVar[int]
    scope: str
    total: int
    rule_matched: int
    classifier_tail: int
    rule_coverage: _containers.ScalarMap[str, int]
    classifier_tail_by_facet: _containers.ScalarMap[str, int]
    ceiling_percent: float
    max_tail_facet: str
    max_tail_percent: float
    within_ceiling: bool
    def __init__(self, scope: _Optional[str] = ..., total: _Optional[int] = ..., rule_matched: _Optional[int] = ..., classifier_tail: _Optional[int] = ..., rule_coverage: _Optional[_Mapping[str, int]] = ..., classifier_tail_by_facet: _Optional[_Mapping[str, int]] = ..., ceiling_percent: _Optional[float] = ..., max_tail_facet: _Optional[str] = ..., max_tail_percent: _Optional[float] = ..., within_ceiling: _Optional[bool] = ...) -> None: ...
