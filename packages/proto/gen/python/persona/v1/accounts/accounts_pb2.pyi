import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AccountLink(_message.Message):
    __slots__ = ("id", "persona_id", "site", "login_seam", "recovery_path", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    SITE_FIELD_NUMBER: _ClassVar[int]
    LOGIN_SEAM_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_PATH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    persona_id: str
    site: str
    login_seam: str
    recovery_path: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., persona_id: _Optional[str] = ..., site: _Optional[str] = ..., login_seam: _Optional[str] = ..., recovery_path: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Address(_message.Message):
    __slots__ = ("id", "persona_id", "label", "line1", "line2", "city", "region", "postal_code", "country", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    LINE1_FIELD_NUMBER: _ClassVar[int]
    LINE2_FIELD_NUMBER: _ClassVar[int]
    CITY_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    POSTAL_CODE_FIELD_NUMBER: _ClassVar[int]
    COUNTRY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    persona_id: str
    label: str
    line1: str
    line2: str
    city: str
    region: str
    postal_code: str
    country: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., persona_id: _Optional[str] = ..., label: _Optional[str] = ..., line1: _Optional[str] = ..., line2: _Optional[str] = ..., city: _Optional[str] = ..., region: _Optional[str] = ..., postal_code: _Optional[str] = ..., country: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Obligation(_message.Message):
    __slots__ = ("id", "persona_id", "account_link_id", "description", "renewal_at", "cancel_path", "cancelled", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_LINK_ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RENEWAL_AT_FIELD_NUMBER: _ClassVar[int]
    CANCEL_PATH_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    persona_id: str
    account_link_id: str
    description: str
    renewal_at: _timestamp_pb2.Timestamp
    cancel_path: str
    cancelled: bool
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., persona_id: _Optional[str] = ..., account_link_id: _Optional[str] = ..., description: _Optional[str] = ..., renewal_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cancel_path: _Optional[str] = ..., cancelled: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class LinkAccountRequest(_message.Message):
    __slots__ = ("persona_id", "site", "login_seam", "recovery_path")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    SITE_FIELD_NUMBER: _ClassVar[int]
    LOGIN_SEAM_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_PATH_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    site: str
    login_seam: str
    recovery_path: str
    def __init__(self, persona_id: _Optional[str] = ..., site: _Optional[str] = ..., login_seam: _Optional[str] = ..., recovery_path: _Optional[str] = ...) -> None: ...

class LinkAccountResponse(_message.Message):
    __slots__ = ("account",)
    ACCOUNT_FIELD_NUMBER: _ClassVar[int]
    account: AccountLink
    def __init__(self, account: _Optional[_Union[AccountLink, _Mapping]] = ...) -> None: ...

class ListAccountsRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class ListAccountsResponse(_message.Message):
    __slots__ = ("accounts",)
    ACCOUNTS_FIELD_NUMBER: _ClassVar[int]
    accounts: _containers.RepeatedCompositeFieldContainer[AccountLink]
    def __init__(self, accounts: _Optional[_Iterable[_Union[AccountLink, _Mapping]]] = ...) -> None: ...

class AddAddressRequest(_message.Message):
    __slots__ = ("persona_id", "address")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    address: Address
    def __init__(self, persona_id: _Optional[str] = ..., address: _Optional[_Union[Address, _Mapping]] = ...) -> None: ...

class AddAddressResponse(_message.Message):
    __slots__ = ("address",)
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    address: Address
    def __init__(self, address: _Optional[_Union[Address, _Mapping]] = ...) -> None: ...

class ListAddressesRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class ListAddressesResponse(_message.Message):
    __slots__ = ("addresses",)
    ADDRESSES_FIELD_NUMBER: _ClassVar[int]
    addresses: _containers.RepeatedCompositeFieldContainer[Address]
    def __init__(self, addresses: _Optional[_Iterable[_Union[Address, _Mapping]]] = ...) -> None: ...

class AddObligationRequest(_message.Message):
    __slots__ = ("persona_id", "account_link_id", "description", "renewal_at", "cancel_path")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_LINK_ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RENEWAL_AT_FIELD_NUMBER: _ClassVar[int]
    CANCEL_PATH_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    account_link_id: str
    description: str
    renewal_at: _timestamp_pb2.Timestamp
    cancel_path: str
    def __init__(self, persona_id: _Optional[str] = ..., account_link_id: _Optional[str] = ..., description: _Optional[str] = ..., renewal_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cancel_path: _Optional[str] = ...) -> None: ...

class AddObligationResponse(_message.Message):
    __slots__ = ("obligation",)
    OBLIGATION_FIELD_NUMBER: _ClassVar[int]
    obligation: Obligation
    def __init__(self, obligation: _Optional[_Union[Obligation, _Mapping]] = ...) -> None: ...

class ListObligationsRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class ListObligationsResponse(_message.Message):
    __slots__ = ("obligations",)
    OBLIGATIONS_FIELD_NUMBER: _ClassVar[int]
    obligations: _containers.RepeatedCompositeFieldContainer[Obligation]
    def __init__(self, obligations: _Optional[_Iterable[_Union[Obligation, _Mapping]]] = ...) -> None: ...

class CancelObligationRequest(_message.Message):
    __slots__ = ("obligation_id",)
    OBLIGATION_ID_FIELD_NUMBER: _ClassVar[int]
    obligation_id: str
    def __init__(self, obligation_id: _Optional[str] = ...) -> None: ...

class CancelObligationResponse(_message.Message):
    __slots__ = ("obligation",)
    OBLIGATION_FIELD_NUMBER: _ClassVar[int]
    obligation: Obligation
    def __init__(self, obligation: _Optional[_Union[Obligation, _Mapping]] = ...) -> None: ...

class ReleaseAddressRequest(_message.Message):
    __slots__ = ("persona_id", "address_id", "target_kind", "target_id")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    address_id: str
    target_kind: str
    target_id: str
    def __init__(self, persona_id: _Optional[str] = ..., address_id: _Optional[str] = ..., target_kind: _Optional[str] = ..., target_id: _Optional[str] = ...) -> None: ...

class ReleaseAddressResponse(_message.Message):
    __slots__ = ("address", "target_kind", "target_id")
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    address: Address
    target_kind: str
    target_id: str
    def __init__(self, address: _Optional[_Union[Address, _Mapping]] = ..., target_kind: _Optional[str] = ..., target_id: _Optional[str] = ...) -> None: ...
