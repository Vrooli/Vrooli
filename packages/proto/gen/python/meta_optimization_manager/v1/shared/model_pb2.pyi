from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class Projection(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROJECTION_UNSPECIFIED: _ClassVar[Projection]
    PROJECTION_ANSWER: _ClassVar[Projection]
    PROJECTION_VALIDATE: _ClassVar[Projection]
    PROJECTION_GUIDE: _ClassVar[Projection]

class DenominatorConfidence(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DENOMINATOR_CONFIDENCE_UNSPECIFIED: _ClassVar[DenominatorConfidence]
    DENOMINATOR_CONFIDENCE_AUTHORITATIVE: _ClassVar[DenominatorConfidence]
    DENOMINATOR_CONFIDENCE_PARTIAL: _ClassVar[DenominatorConfidence]
    DENOMINATOR_CONFIDENCE_SKETCH: _ClassVar[DenominatorConfidence]

class CellStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CELL_STATUS_UNSPECIFIED: _ClassVar[CellStatus]
    CELL_STATUS_NOW: _ClassVar[CellStatus]
    CELL_STATUS_IN_REACH: _ClassVar[CellStatus]
    CELL_STATUS_MISSING: _ClassVar[CellStatus]

class Severity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SEVERITY_UNSPECIFIED: _ClassVar[Severity]
    SEVERITY_INFO: _ClassVar[Severity]
    SEVERITY_WARN: _ClassVar[Severity]
    SEVERITY_ERROR: _ClassVar[Severity]
PROJECTION_UNSPECIFIED: Projection
PROJECTION_ANSWER: Projection
PROJECTION_VALIDATE: Projection
PROJECTION_GUIDE: Projection
DENOMINATOR_CONFIDENCE_UNSPECIFIED: DenominatorConfidence
DENOMINATOR_CONFIDENCE_AUTHORITATIVE: DenominatorConfidence
DENOMINATOR_CONFIDENCE_PARTIAL: DenominatorConfidence
DENOMINATOR_CONFIDENCE_SKETCH: DenominatorConfidence
CELL_STATUS_UNSPECIFIED: CellStatus
CELL_STATUS_NOW: CellStatus
CELL_STATUS_IN_REACH: CellStatus
CELL_STATUS_MISSING: CellStatus
SEVERITY_UNSPECIFIED: Severity
SEVERITY_INFO: Severity
SEVERITY_WARN: Severity
SEVERITY_ERROR: Severity
