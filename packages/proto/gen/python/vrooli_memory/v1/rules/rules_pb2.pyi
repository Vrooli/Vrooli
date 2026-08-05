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
    __slots__ = ("rule_id",)
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    def __init__(self, rule_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("rule_id",)
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    def __init__(self, rule_id: _Optional[str] = ...) -> None: ...

class EnableRuleResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RevertRuleRequest(_message.Message):
    __slots__ = ("rule_id",)
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    def __init__(self, rule_id: _Optional[str] = ...) -> None: ...

class RevertRuleResponse(_message.Message):
    __slots__ = ("restored_count",)
    RESTORED_COUNT_FIELD_NUMBER: _ClassVar[int]
    restored_count: int
    def __init__(self, restored_count: _Optional[int] = ...) -> None: ...

class RefacetCorpusRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class RefacetCorpusResponse(_message.Message):
    __slots__ = ("total", "assigned", "rule_assigned", "classified", "failed")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    ASSIGNED_FIELD_NUMBER: _ClassVar[int]
    RULE_ASSIGNED_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    total: int
    assigned: int
    rule_assigned: int
    classified: int
    failed: int
    def __init__(self, total: _Optional[int] = ..., assigned: _Optional[int] = ..., rule_assigned: _Optional[int] = ..., classified: _Optional[int] = ..., failed: _Optional[int] = ...) -> None: ...
