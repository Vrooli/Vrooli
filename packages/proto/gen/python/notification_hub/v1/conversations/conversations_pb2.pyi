from notification_hub.v1.shared import types_pb2 as _types_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AskOption(_message.Message):
    __slots__ = ("key", "label")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class ProviderDecisionContext(_message.Message):
    __slots__ = ("schema_version", "canonical_terms_sha256", "scope")
    class Scope(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        SCOPE_UNSPECIFIED: _ClassVar[ProviderDecisionContext.Scope]
        SCOPE_PROVIDER_RULE: _ClassVar[ProviderDecisionContext.Scope]
        SCOPE_PROVIDER_AGREEMENT: _ClassVar[ProviderDecisionContext.Scope]
    SCOPE_UNSPECIFIED: ProviderDecisionContext.Scope
    SCOPE_PROVIDER_RULE: ProviderDecisionContext.Scope
    SCOPE_PROVIDER_AGREEMENT: ProviderDecisionContext.Scope
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_TERMS_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    canonical_terms_sha256: str
    scope: ProviderDecisionContext.Scope
    def __init__(self, schema_version: _Optional[int] = ..., canonical_terms_sha256: _Optional[str] = ..., scope: _Optional[_Union[ProviderDecisionContext.Scope, str]] = ...) -> None: ...

class AskRequest(_message.Message):
    __slots__ = ("question", "allowed_answers", "deadline", "sensitivity_label", "idempotency_key", "options", "recommended", "recommendation_reason", "default_answer", "reversible", "urgency", "context_url", "big_picture", "technical_details", "provider_decision_context", "subject_key", "reassert_within_seconds", "attachments", "questions", "accepts_context_requests", "clear_attachments", "clear_questions", "remove_attachments", "remove_questions", "context_response", "sender", "default_sender", "packet", "template_name", "template_data", "kind", "allow_related")
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_ANSWERS_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    SENSITIVITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_REASON_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ANSWER_FIELD_NUMBER: _ClassVar[int]
    REVERSIBLE_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_URL_FIELD_NUMBER: _ClassVar[int]
    BIG_PICTURE_FIELD_NUMBER: _ClassVar[int]
    TECHNICAL_DETAILS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_DECISION_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_KEY_FIELD_NUMBER: _ClassVar[int]
    REASSERT_WITHIN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTS_CONTEXT_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    CLEAR_ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    CLEAR_QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    REMOVE_ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    REMOVE_QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_RESPONSE_FIELD_NUMBER: _ClassVar[int]
    SENDER_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SENDER_FIELD_NUMBER: _ClassVar[int]
    PACKET_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_NAME_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_DATA_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ALLOW_RELATED_FIELD_NUMBER: _ClassVar[int]
    question: str
    allowed_answers: _containers.RepeatedScalarFieldContainer[str]
    deadline: str
    sensitivity_label: str
    idempotency_key: str
    options: _containers.RepeatedCompositeFieldContainer[AskOption]
    recommended: str
    recommendation_reason: str
    default_answer: str
    reversible: bool
    urgency: str
    context_url: str
    big_picture: str
    technical_details: str
    provider_decision_context: ProviderDecisionContext
    subject_key: str
    reassert_within_seconds: int
    attachments: _containers.RepeatedCompositeFieldContainer[AskAttachment]
    questions: _containers.RepeatedCompositeFieldContainer[AskQuestion]
    accepts_context_requests: bool
    clear_attachments: bool
    clear_questions: bool
    remove_attachments: _containers.RepeatedScalarFieldContainer[str]
    remove_questions: _containers.RepeatedScalarFieldContainer[str]
    context_response: str
    sender: str
    default_sender: str
    packet: str
    template_name: str
    template_data: _struct_pb2.Struct
    kind: str
    allow_related: bool
    def __init__(self, question: _Optional[str] = ..., allowed_answers: _Optional[_Iterable[str]] = ..., deadline: _Optional[str] = ..., sensitivity_label: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., options: _Optional[_Iterable[_Union[AskOption, _Mapping]]] = ..., recommended: _Optional[str] = ..., recommendation_reason: _Optional[str] = ..., default_answer: _Optional[str] = ..., reversible: _Optional[bool] = ..., urgency: _Optional[str] = ..., context_url: _Optional[str] = ..., big_picture: _Optional[str] = ..., technical_details: _Optional[str] = ..., provider_decision_context: _Optional[_Union[ProviderDecisionContext, _Mapping]] = ..., subject_key: _Optional[str] = ..., reassert_within_seconds: _Optional[int] = ..., attachments: _Optional[_Iterable[_Union[AskAttachment, _Mapping]]] = ..., questions: _Optional[_Iterable[_Union[AskQuestion, _Mapping]]] = ..., accepts_context_requests: _Optional[bool] = ..., clear_attachments: _Optional[bool] = ..., clear_questions: _Optional[bool] = ..., remove_attachments: _Optional[_Iterable[str]] = ..., remove_questions: _Optional[_Iterable[str]] = ..., context_response: _Optional[str] = ..., sender: _Optional[str] = ..., default_sender: _Optional[str] = ..., packet: _Optional[str] = ..., template_name: _Optional[str] = ..., template_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., kind: _Optional[str] = ..., allow_related: _Optional[bool] = ...) -> None: ...

class AskResponse(_message.Message):
    __slots__ = ("ask_id", "notification", "outcome", "warnings", "state", "kind")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATION_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    notification: _types_pb2.Notification
    outcome: str
    warnings: _containers.RepeatedCompositeFieldContainer[PacketWarning]
    state: str
    kind: str
    def __init__(self, ask_id: _Optional[str] = ..., notification: _Optional[_Union[_types_pb2.Notification, _Mapping]] = ..., outcome: _Optional[str] = ..., warnings: _Optional[_Iterable[_Union[PacketWarning, _Mapping]]] = ..., state: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class CheckAskRequest(_message.Message):
    __slots__ = ("ask", "file_contents")
    class FileContentsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: bytes
        def __init__(self, key: _Optional[str] = ..., value: _Optional[bytes] = ...) -> None: ...
    ASK_FIELD_NUMBER: _ClassVar[int]
    FILE_CONTENTS_FIELD_NUMBER: _ClassVar[int]
    ask: AskRequest
    file_contents: _containers.ScalarMap[str, bytes]
    def __init__(self, ask: _Optional[_Union[AskRequest, _Mapping]] = ..., file_contents: _Optional[_Mapping[str, bytes]] = ...) -> None: ...

class CheckAskResponse(_message.Message):
    __slots__ = ("warnings", "push_title", "push_body", "revises_ask_id", "sender", "sender_source")
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    PUSH_TITLE_FIELD_NUMBER: _ClassVar[int]
    PUSH_BODY_FIELD_NUMBER: _ClassVar[int]
    REVISES_ASK_ID_FIELD_NUMBER: _ClassVar[int]
    SENDER_FIELD_NUMBER: _ClassVar[int]
    SENDER_SOURCE_FIELD_NUMBER: _ClassVar[int]
    warnings: _containers.RepeatedCompositeFieldContainer[PacketWarning]
    push_title: str
    push_body: str
    revises_ask_id: str
    sender: str
    sender_source: str
    def __init__(self, warnings: _Optional[_Iterable[_Union[PacketWarning, _Mapping]]] = ..., push_title: _Optional[str] = ..., push_body: _Optional[str] = ..., revises_ask_id: _Optional[str] = ..., sender: _Optional[str] = ..., sender_source: _Optional[str] = ...) -> None: ...

class PacketWarning(_message.Message):
    __slots__ = ("code", "field", "message", "severity", "carried_over")
    CODE_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CARRIED_OVER_FIELD_NUMBER: _ClassVar[int]
    code: str
    field: str
    message: str
    severity: str
    carried_over: bool
    def __init__(self, code: _Optional[str] = ..., field: _Optional[str] = ..., message: _Optional[str] = ..., severity: _Optional[str] = ..., carried_over: _Optional[bool] = ...) -> None: ...

class AskAttachment(_message.Message):
    __slots__ = ("name", "label", "kind", "role", "source_path", "source_host", "sha256", "size_bytes", "media_type", "entries", "preview_kind", "source_status", "source_checked_at", "truncated")
    class Kind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        KIND_UNSPECIFIED: _ClassVar[AskAttachment.Kind]
        KIND_FILE: _ClassVar[AskAttachment.Kind]
        KIND_FOLDER: _ClassVar[AskAttachment.Kind]
    KIND_UNSPECIFIED: AskAttachment.Kind
    KIND_FILE: AskAttachment.Kind
    KIND_FOLDER: AskAttachment.Kind
    class Role(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        ROLE_UNSPECIFIED: _ClassVar[AskAttachment.Role]
        ROLE_EVIDENCE: _ClassVar[AskAttachment.Role]
        ROLE_EXPLAINER: _ClassVar[AskAttachment.Role]
    ROLE_UNSPECIFIED: AskAttachment.Role
    ROLE_EVIDENCE: AskAttachment.Role
    ROLE_EXPLAINER: AskAttachment.Role
    NAME_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HOST_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_STATUS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    name: str
    label: str
    kind: AskAttachment.Kind
    role: AskAttachment.Role
    source_path: str
    source_host: str
    sha256: str
    size_bytes: int
    media_type: str
    entries: _containers.RepeatedCompositeFieldContainer[AskAttachmentEntry]
    preview_kind: str
    source_status: str
    source_checked_at: str
    truncated: bool
    def __init__(self, name: _Optional[str] = ..., label: _Optional[str] = ..., kind: _Optional[_Union[AskAttachment.Kind, str]] = ..., role: _Optional[_Union[AskAttachment.Role, str]] = ..., source_path: _Optional[str] = ..., source_host: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., media_type: _Optional[str] = ..., entries: _Optional[_Iterable[_Union[AskAttachmentEntry, _Mapping]]] = ..., preview_kind: _Optional[str] = ..., source_status: _Optional[str] = ..., source_checked_at: _Optional[str] = ..., truncated: _Optional[bool] = ...) -> None: ...

class AskAttachmentEntry(_message.Message):
    __slots__ = ("path", "sha256", "size_bytes", "media_type", "preview_kind")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_KIND_FIELD_NUMBER: _ClassVar[int]
    path: str
    sha256: str
    size_bytes: int
    media_type: str
    preview_kind: str
    def __init__(self, path: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., media_type: _Optional[str] = ..., preview_kind: _Optional[str] = ...) -> None: ...

class AskQuestion(_message.Message):
    __slots__ = ("id", "type", "prompt", "help", "required", "choices", "sections", "recommended", "recommendation_reason", "unanswered_verdict", "sections_from")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    HELP_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    CHOICES_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_REASON_FIELD_NUMBER: _ClassVar[int]
    UNANSWERED_VERDICT_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FROM_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    prompt: str
    help: str
    required: bool
    choices: _containers.RepeatedCompositeFieldContainer[AskQuestionChoice]
    sections: _containers.RepeatedCompositeFieldContainer[AskQuestionSection]
    recommended: AskQuestionResponse
    recommendation_reason: str
    unanswered_verdict: str
    sections_from: str
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., prompt: _Optional[str] = ..., help: _Optional[str] = ..., required: _Optional[bool] = ..., choices: _Optional[_Iterable[_Union[AskQuestionChoice, _Mapping]]] = ..., sections: _Optional[_Iterable[_Union[AskQuestionSection, _Mapping]]] = ..., recommended: _Optional[_Union[AskQuestionResponse, _Mapping]] = ..., recommendation_reason: _Optional[str] = ..., unanswered_verdict: _Optional[str] = ..., sections_from: _Optional[str] = ...) -> None: ...

class AskQuestionChoice(_message.Message):
    __slots__ = ("key", "label", "detail", "attachment")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    detail: str
    attachment: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., detail: _Optional[str] = ..., attachment: _Optional[str] = ...) -> None: ...

class AskQuestionSection(_message.Message):
    __slots__ = ("key", "title", "summary", "attachment", "anchor")
    KEY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_FIELD_NUMBER: _ClassVar[int]
    ANCHOR_FIELD_NUMBER: _ClassVar[int]
    key: str
    title: str
    summary: str
    attachment: str
    anchor: str
    def __init__(self, key: _Optional[str] = ..., title: _Optional[str] = ..., summary: _Optional[str] = ..., attachment: _Optional[str] = ..., anchor: _Optional[str] = ...) -> None: ...

class AskQuestionResponse(_message.Message):
    __slots__ = ("question_id", "selected", "text", "sections")
    QUESTION_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    question_id: str
    selected: _containers.RepeatedScalarFieldContainer[str]
    text: str
    sections: _containers.RepeatedCompositeFieldContainer[AskSectionVerdict]
    def __init__(self, question_id: _Optional[str] = ..., selected: _Optional[_Iterable[str]] = ..., text: _Optional[str] = ..., sections: _Optional[_Iterable[_Union[AskSectionVerdict, _Mapping]]] = ...) -> None: ...

class AskAttachmentNote(_message.Message):
    __slots__ = ("attachment", "path", "note")
    ATTACHMENT_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    attachment: str
    path: str
    note: str
    def __init__(self, attachment: _Optional[str] = ..., path: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class AskSectionVerdict(_message.Message):
    __slots__ = ("section_key", "verdict", "note")
    SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    section_key: str
    verdict: str
    note: str
    def __init__(self, section_key: _Optional[str] = ..., verdict: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class AskContextRequest(_message.Message):
    __slots__ = ("id", "question", "requested_by", "requested_at", "addressed_at", "addressed_revision", "response")
    ID_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    ADDRESSED_AT_FIELD_NUMBER: _ClassVar[int]
    ADDRESSED_REVISION_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    id: str
    question: str
    requested_by: str
    requested_at: str
    addressed_at: str
    addressed_revision: int
    response: str
    def __init__(self, id: _Optional[str] = ..., question: _Optional[str] = ..., requested_by: _Optional[str] = ..., requested_at: _Optional[str] = ..., addressed_at: _Optional[str] = ..., addressed_revision: _Optional[int] = ..., response: _Optional[str] = ...) -> None: ...

class RequestContextRequest(_message.Message):
    __slots__ = ("ask_id", "question")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    question: str
    def __init__(self, ask_id: _Optional[str] = ..., question: _Optional[str] = ...) -> None: ...

class RequestContextResponse(_message.Message):
    __slots__ = ("ask_id", "request")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    request: AskContextRequest
    def __init__(self, ask_id: _Optional[str] = ..., request: _Optional[_Union[AskContextRequest, _Mapping]] = ...) -> None: ...

class StatAttachmentBlobsRequest(_message.Message):
    __slots__ = ("sha256",)
    SHA256_FIELD_NUMBER: _ClassVar[int]
    sha256: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, sha256: _Optional[_Iterable[str]] = ...) -> None: ...

class StatAttachmentBlobsResponse(_message.Message):
    __slots__ = ("missing",)
    MISSING_FIELD_NUMBER: _ClassVar[int]
    missing: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, missing: _Optional[_Iterable[str]] = ...) -> None: ...

class PutAttachmentBlobRequest(_message.Message):
    __slots__ = ("sha256", "media_type", "content")
    SHA256_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    sha256: str
    media_type: str
    content: bytes
    def __init__(self, sha256: _Optional[str] = ..., media_type: _Optional[str] = ..., content: _Optional[bytes] = ...) -> None: ...

class PutAttachmentBlobResponse(_message.Message):
    __slots__ = ("sha256", "size_bytes", "existed")
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    EXISTED_FIELD_NUMBER: _ClassVar[int]
    sha256: str
    size_bytes: int
    existed: bool
    def __init__(self, sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., existed: _Optional[bool] = ...) -> None: ...

class AnswerRootWithPasswordRequest(_message.Message):
    __slots__ = ("ask_id", "revision_count", "answer", "email", "password", "mfa_challenge", "totp_code", "recovery_code")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_COUNT_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    MFA_CHALLENGE_FIELD_NUMBER: _ClassVar[int]
    TOTP_CODE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_CODE_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    revision_count: int
    answer: str
    email: str
    password: str
    mfa_challenge: str
    totp_code: str
    recovery_code: str
    def __init__(self, ask_id: _Optional[str] = ..., revision_count: _Optional[int] = ..., answer: _Optional[str] = ..., email: _Optional[str] = ..., password: _Optional[str] = ..., mfa_challenge: _Optional[str] = ..., totp_code: _Optional[str] = ..., recovery_code: _Optional[str] = ...) -> None: ...

class AnswerRootWithPasswordResponse(_message.Message):
    __slots__ = ("mfa_required", "mfa_challenge", "answered_at")
    MFA_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MFA_CHALLENGE_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    mfa_required: bool
    mfa_challenge: str
    answered_at: str
    def __init__(self, mfa_required: _Optional[bool] = ..., mfa_challenge: _Optional[str] = ..., answered_at: _Optional[str] = ...) -> None: ...

class AnswerRequest(_message.Message):
    __slots__ = ("ask_id", "answer", "note", "responses", "attachment_notes", "changes", "images")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    RESPONSES_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_NOTES_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    IMAGES_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    answer: str
    note: str
    responses: _containers.RepeatedCompositeFieldContainer[AskQuestionResponse]
    attachment_notes: _containers.RepeatedCompositeFieldContainer[AskAttachmentNote]
    changes: _struct_pb2.Struct
    images: _containers.RepeatedCompositeFieldContainer[AnswerImage]
    def __init__(self, ask_id: _Optional[str] = ..., answer: _Optional[str] = ..., note: _Optional[str] = ..., responses: _Optional[_Iterable[_Union[AskQuestionResponse, _Mapping]]] = ..., attachment_notes: _Optional[_Iterable[_Union[AskAttachmentNote, _Mapping]]] = ..., changes: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., images: _Optional[_Iterable[_Union[AnswerImage, _Mapping]]] = ...) -> None: ...

class AnswerResponse(_message.Message):
    __slots__ = ("ask_id", "answer", "answered_at", "late", "changes", "images")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    LATE_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    IMAGES_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    answer: str
    answered_at: str
    late: bool
    changes: _struct_pb2.Struct
    images: _containers.RepeatedCompositeFieldContainer[AnswerImage]
    def __init__(self, ask_id: _Optional[str] = ..., answer: _Optional[str] = ..., answered_at: _Optional[str] = ..., late: _Optional[bool] = ..., changes: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., images: _Optional[_Iterable[_Union[AnswerImage, _Mapping]]] = ...) -> None: ...

class AnswerImage(_message.Message):
    __slots__ = ("name", "sha256", "media_type", "size_bytes")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    name: str
    sha256: str
    media_type: str
    size_bytes: int
    def __init__(self, name: _Optional[str] = ..., sha256: _Optional[str] = ..., media_type: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class MarkReadRequest(_message.Message):
    __slots__ = ("ask_id",)
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    def __init__(self, ask_id: _Optional[str] = ...) -> None: ...

class MarkReadResponse(_message.Message):
    __slots__ = ("ask_id", "state")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    state: str
    def __init__(self, ask_id: _Optional[str] = ..., state: _Optional[str] = ...) -> None: ...

class DismissRequest(_message.Message):
    __slots__ = ("ask_id", "note")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    note: str
    def __init__(self, ask_id: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class DismissResponse(_message.Message):
    __slots__ = ("ask_id", "state")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    state: str
    def __init__(self, ask_id: _Optional[str] = ..., state: _Optional[str] = ...) -> None: ...

class WithdrawRequest(_message.Message):
    __slots__ = ("ask_id", "reason", "subject_key")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_KEY_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    reason: str
    subject_key: str
    def __init__(self, ask_id: _Optional[str] = ..., reason: _Optional[str] = ..., subject_key: _Optional[str] = ...) -> None: ...

class WithdrawResponse(_message.Message):
    __slots__ = ("ask_id", "state")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    state: str
    def __init__(self, ask_id: _Optional[str] = ..., state: _Optional[str] = ...) -> None: ...

class WaitRequest(_message.Message):
    __slots__ = ("ask_id", "deadline", "return_on_context_request", "subject_key")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    RETURN_ON_CONTEXT_REQUEST_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_KEY_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    deadline: str
    return_on_context_request: bool
    subject_key: str
    def __init__(self, ask_id: _Optional[str] = ..., deadline: _Optional[str] = ..., return_on_context_request: _Optional[bool] = ..., subject_key: _Optional[str] = ...) -> None: ...

class WaitResponse(_message.Message):
    __slots__ = ("ask_id", "state", "answer", "reason", "note", "responses", "answer_summary", "context_request", "attachment_notes", "packet", "changes", "images")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    RESPONSES_FIELD_NUMBER: _ClassVar[int]
    ANSWER_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_REQUEST_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_NOTES_FIELD_NUMBER: _ClassVar[int]
    PACKET_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    IMAGES_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    state: str
    answer: str
    reason: str
    note: str
    responses: _containers.RepeatedCompositeFieldContainer[AskQuestionResponse]
    answer_summary: str
    context_request: AskContextRequest
    attachment_notes: _containers.RepeatedCompositeFieldContainer[AskAttachmentNote]
    packet: str
    changes: _struct_pb2.Struct
    images: _containers.RepeatedCompositeFieldContainer[AnswerImage]
    def __init__(self, ask_id: _Optional[str] = ..., state: _Optional[str] = ..., answer: _Optional[str] = ..., reason: _Optional[str] = ..., note: _Optional[str] = ..., responses: _Optional[_Iterable[_Union[AskQuestionResponse, _Mapping]]] = ..., answer_summary: _Optional[str] = ..., context_request: _Optional[_Union[AskContextRequest, _Mapping]] = ..., attachment_notes: _Optional[_Iterable[_Union[AskAttachmentNote, _Mapping]]] = ..., packet: _Optional[str] = ..., changes: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., images: _Optional[_Iterable[_Union[AnswerImage, _Mapping]]] = ...) -> None: ...

class Ask(_message.Message):
    __slots__ = ("id", "notification_id", "question", "options", "recommended", "recommendation_reason", "default_answer", "reversible", "urgency", "context_url", "deadline", "state", "reason", "answer", "answer_label", "note", "answered_by", "answered_at", "late", "default_applied_at", "first_delivered_at", "default_eligible_at", "source", "source_event_type", "requester_ref", "resolved_at", "created_at", "updated_at", "big_picture", "technical_details", "title", "provider_decision_context", "answered_by_identity_kind", "answered_by_identity_source", "subject_key", "last_asserted_at", "reassert_within_seconds", "revision_count", "superseded_by_ask_id", "revisions", "attachments", "questions", "responses", "context_requests", "answer_summary", "attachment_count", "question_count", "open_context_request_count", "attachment_notes", "can_request_context", "correlation", "sender", "closing_soon", "packet", "approval_url", "kind", "answer_changes", "answer_images", "recurrence", "template_name", "template_data")
    class CorrelationEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATION_ID_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_REASON_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ANSWER_FIELD_NUMBER: _ClassVar[int]
    REVERSIBLE_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_URL_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    ANSWER_LABEL_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_BY_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    LATE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    FIRST_DELIVERED_AT_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ELIGIBLE_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUESTER_REF_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    BIG_PICTURE_FIELD_NUMBER: _ClassVar[int]
    TECHNICAL_DETAILS_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_DECISION_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_BY_IDENTITY_KIND_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_BY_IDENTITY_SOURCE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_KEY_FIELD_NUMBER: _ClassVar[int]
    LAST_ASSERTED_AT_FIELD_NUMBER: _ClassVar[int]
    REASSERT_WITHIN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REVISION_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_BY_ASK_ID_FIELD_NUMBER: _ClassVar[int]
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    RESPONSES_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    ANSWER_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    QUESTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    OPEN_CONTEXT_REQUEST_COUNT_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_NOTES_FIELD_NUMBER: _ClassVar[int]
    CAN_REQUEST_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_FIELD_NUMBER: _ClassVar[int]
    SENDER_FIELD_NUMBER: _ClassVar[int]
    CLOSING_SOON_FIELD_NUMBER: _ClassVar[int]
    PACKET_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_URL_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ANSWER_CHANGES_FIELD_NUMBER: _ClassVar[int]
    ANSWER_IMAGES_FIELD_NUMBER: _ClassVar[int]
    RECURRENCE_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_NAME_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_DATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    notification_id: str
    question: str
    options: _containers.RepeatedCompositeFieldContainer[AskOption]
    recommended: str
    recommendation_reason: str
    default_answer: str
    reversible: bool
    urgency: str
    context_url: str
    deadline: str
    state: str
    reason: str
    answer: str
    answer_label: str
    note: str
    answered_by: str
    answered_at: str
    late: bool
    default_applied_at: str
    first_delivered_at: str
    default_eligible_at: str
    source: str
    source_event_type: str
    requester_ref: str
    resolved_at: str
    created_at: str
    updated_at: str
    big_picture: str
    technical_details: str
    title: str
    provider_decision_context: ProviderDecisionContext
    answered_by_identity_kind: str
    answered_by_identity_source: str
    subject_key: str
    last_asserted_at: str
    reassert_within_seconds: int
    revision_count: int
    superseded_by_ask_id: str
    revisions: _containers.RepeatedCompositeFieldContainer[AskRevision]
    attachments: _containers.RepeatedCompositeFieldContainer[AskAttachment]
    questions: _containers.RepeatedCompositeFieldContainer[AskQuestion]
    responses: _containers.RepeatedCompositeFieldContainer[AskQuestionResponse]
    context_requests: _containers.RepeatedCompositeFieldContainer[AskContextRequest]
    answer_summary: str
    attachment_count: int
    question_count: int
    open_context_request_count: int
    attachment_notes: _containers.RepeatedCompositeFieldContainer[AskAttachmentNote]
    can_request_context: bool
    correlation: _containers.ScalarMap[str, str]
    sender: str
    closing_soon: bool
    packet: str
    approval_url: str
    kind: str
    answer_changes: _struct_pb2.Struct
    answer_images: _containers.RepeatedCompositeFieldContainer[AnswerImage]
    recurrence: SubjectRecurrence
    template_name: str
    template_data: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., notification_id: _Optional[str] = ..., question: _Optional[str] = ..., options: _Optional[_Iterable[_Union[AskOption, _Mapping]]] = ..., recommended: _Optional[str] = ..., recommendation_reason: _Optional[str] = ..., default_answer: _Optional[str] = ..., reversible: _Optional[bool] = ..., urgency: _Optional[str] = ..., context_url: _Optional[str] = ..., deadline: _Optional[str] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ..., answer: _Optional[str] = ..., answer_label: _Optional[str] = ..., note: _Optional[str] = ..., answered_by: _Optional[str] = ..., answered_at: _Optional[str] = ..., late: _Optional[bool] = ..., default_applied_at: _Optional[str] = ..., first_delivered_at: _Optional[str] = ..., default_eligible_at: _Optional[str] = ..., source: _Optional[str] = ..., source_event_type: _Optional[str] = ..., requester_ref: _Optional[str] = ..., resolved_at: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., big_picture: _Optional[str] = ..., technical_details: _Optional[str] = ..., title: _Optional[str] = ..., provider_decision_context: _Optional[_Union[ProviderDecisionContext, _Mapping]] = ..., answered_by_identity_kind: _Optional[str] = ..., answered_by_identity_source: _Optional[str] = ..., subject_key: _Optional[str] = ..., last_asserted_at: _Optional[str] = ..., reassert_within_seconds: _Optional[int] = ..., revision_count: _Optional[int] = ..., superseded_by_ask_id: _Optional[str] = ..., revisions: _Optional[_Iterable[_Union[AskRevision, _Mapping]]] = ..., attachments: _Optional[_Iterable[_Union[AskAttachment, _Mapping]]] = ..., questions: _Optional[_Iterable[_Union[AskQuestion, _Mapping]]] = ..., responses: _Optional[_Iterable[_Union[AskQuestionResponse, _Mapping]]] = ..., context_requests: _Optional[_Iterable[_Union[AskContextRequest, _Mapping]]] = ..., answer_summary: _Optional[str] = ..., attachment_count: _Optional[int] = ..., question_count: _Optional[int] = ..., open_context_request_count: _Optional[int] = ..., attachment_notes: _Optional[_Iterable[_Union[AskAttachmentNote, _Mapping]]] = ..., can_request_context: _Optional[bool] = ..., correlation: _Optional[_Mapping[str, str]] = ..., sender: _Optional[str] = ..., closing_soon: _Optional[bool] = ..., packet: _Optional[str] = ..., approval_url: _Optional[str] = ..., kind: _Optional[str] = ..., answer_changes: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., answer_images: _Optional[_Iterable[_Union[AnswerImage, _Mapping]]] = ..., recurrence: _Optional[_Union[SubjectRecurrence, _Mapping]] = ..., template_name: _Optional[str] = ..., template_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class HistoricalAnswer(_message.Message):
    __slots__ = ("answer", "note", "answered_at", "changes")
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    answer: str
    note: str
    answered_at: str
    changes: _struct_pb2.Struct
    def __init__(self, answer: _Optional[str] = ..., note: _Optional[str] = ..., answered_at: _Optional[str] = ..., changes: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class SubjectRecurrence(_message.Message):
    __slots__ = ("count", "dates", "previous_answers")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    DATES_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_ANSWERS_FIELD_NUMBER: _ClassVar[int]
    count: int
    dates: _containers.RepeatedScalarFieldContainer[str]
    previous_answers: _containers.RepeatedCompositeFieldContainer[HistoricalAnswer]
    def __init__(self, count: _Optional[int] = ..., dates: _Optional[_Iterable[str]] = ..., previous_answers: _Optional[_Iterable[_Union[HistoricalAnswer, _Mapping]]] = ...) -> None: ...

class AskRevision(_message.Message):
    __slots__ = ("revision", "title", "question", "recommendation_reason", "context_url", "big_picture", "technical_details", "options", "deadline", "revised_at")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_REASON_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_URL_FIELD_NUMBER: _ClassVar[int]
    BIG_PICTURE_FIELD_NUMBER: _ClassVar[int]
    TECHNICAL_DETAILS_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    REVISED_AT_FIELD_NUMBER: _ClassVar[int]
    revision: int
    title: str
    question: str
    recommendation_reason: str
    context_url: str
    big_picture: str
    technical_details: str
    options: _containers.RepeatedCompositeFieldContainer[AskOption]
    deadline: str
    revised_at: str
    def __init__(self, revision: _Optional[int] = ..., title: _Optional[str] = ..., question: _Optional[str] = ..., recommendation_reason: _Optional[str] = ..., context_url: _Optional[str] = ..., big_picture: _Optional[str] = ..., technical_details: _Optional[str] = ..., options: _Optional[_Iterable[_Union[AskOption, _Mapping]]] = ..., deadline: _Optional[str] = ..., revised_at: _Optional[str] = ...) -> None: ...

class GetAskRequest(_message.Message):
    __slots__ = ("ask_id", "subject_key")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_KEY_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    subject_key: str
    def __init__(self, ask_id: _Optional[str] = ..., subject_key: _Optional[str] = ...) -> None: ...

class GetAskResponse(_message.Message):
    __slots__ = ("ask",)
    ASK_FIELD_NUMBER: _ClassVar[int]
    ask: Ask
    def __init__(self, ask: _Optional[_Union[Ask, _Mapping]] = ...) -> None: ...

class ListAsksRequest(_message.Message):
    __slots__ = ("open_only", "limit", "cursor", "query", "state_filter", "order", "urgencies", "senders")
    class Order(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        ORDER_UNSPECIFIED: _ClassVar[ListAsksRequest.Order]
        ORDER_NEWEST: _ClassVar[ListAsksRequest.Order]
        ORDER_ATTENTION: _ClassVar[ListAsksRequest.Order]
        ORDER_DUE: _ClassVar[ListAsksRequest.Order]
        ORDER_OLDEST: _ClassVar[ListAsksRequest.Order]
    ORDER_UNSPECIFIED: ListAsksRequest.Order
    ORDER_NEWEST: ListAsksRequest.Order
    ORDER_ATTENTION: ListAsksRequest.Order
    ORDER_DUE: ListAsksRequest.Order
    ORDER_OLDEST: ListAsksRequest.Order
    OPEN_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    STATE_FILTER_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    URGENCIES_FIELD_NUMBER: _ClassVar[int]
    SENDERS_FIELD_NUMBER: _ClassVar[int]
    open_only: bool
    limit: int
    cursor: str
    query: str
    state_filter: str
    order: ListAsksRequest.Order
    urgencies: _containers.RepeatedScalarFieldContainer[str]
    senders: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, open_only: _Optional[bool] = ..., limit: _Optional[int] = ..., cursor: _Optional[str] = ..., query: _Optional[str] = ..., state_filter: _Optional[str] = ..., order: _Optional[_Union[ListAsksRequest.Order, str]] = ..., urgencies: _Optional[_Iterable[str]] = ..., senders: _Optional[_Iterable[str]] = ...) -> None: ...

class ListAsksResponse(_message.Message):
    __slots__ = ("asks", "next_cursor", "total_awaiting", "facets", "closing_window_seconds")
    ASKS_FIELD_NUMBER: _ClassVar[int]
    NEXT_CURSOR_FIELD_NUMBER: _ClassVar[int]
    TOTAL_AWAITING_FIELD_NUMBER: _ClassVar[int]
    FACETS_FIELD_NUMBER: _ClassVar[int]
    CLOSING_WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    asks: _containers.RepeatedCompositeFieldContainer[Ask]
    next_cursor: str
    total_awaiting: int
    facets: AskFacets
    closing_window_seconds: int
    def __init__(self, asks: _Optional[_Iterable[_Union[Ask, _Mapping]]] = ..., next_cursor: _Optional[str] = ..., total_awaiting: _Optional[int] = ..., facets: _Optional[_Union[AskFacets, _Mapping]] = ..., closing_window_seconds: _Optional[int] = ...) -> None: ...

class AskFacets(_message.Message):
    __slots__ = ("urgencies", "senders")
    URGENCIES_FIELD_NUMBER: _ClassVar[int]
    SENDERS_FIELD_NUMBER: _ClassVar[int]
    urgencies: _containers.RepeatedCompositeFieldContainer[AskFacetCount]
    senders: _containers.RepeatedCompositeFieldContainer[AskFacetCount]
    def __init__(self, urgencies: _Optional[_Iterable[_Union[AskFacetCount, _Mapping]]] = ..., senders: _Optional[_Iterable[_Union[AskFacetCount, _Mapping]]] = ...) -> None: ...

class AskFacetCount(_message.Message):
    __slots__ = ("value", "count")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    value: str
    count: int
    def __init__(self, value: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class NativeConsent(_message.Message):
    __slots__ = ("authorization_id", "notification_id", "source", "title", "body", "review_url", "state", "review_expires_at", "observed_at", "created_at")
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    REVIEW_URL_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REVIEW_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    authorization_id: str
    notification_id: str
    source: str
    title: str
    body: str
    review_url: str
    state: str
    review_expires_at: str
    observed_at: str
    created_at: str
    def __init__(self, authorization_id: _Optional[str] = ..., notification_id: _Optional[str] = ..., source: _Optional[str] = ..., title: _Optional[str] = ..., body: _Optional[str] = ..., review_url: _Optional[str] = ..., state: _Optional[str] = ..., review_expires_at: _Optional[str] = ..., observed_at: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class ListNativeConsentsRequest(_message.Message):
    __slots__ = ("limit", "cursor", "query")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    limit: int
    cursor: str
    query: str
    def __init__(self, limit: _Optional[int] = ..., cursor: _Optional[str] = ..., query: _Optional[str] = ...) -> None: ...

class ListNativeConsentsResponse(_message.Message):
    __slots__ = ("consents", "next_cursor")
    CONSENTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_CURSOR_FIELD_NUMBER: _ClassVar[int]
    consents: _containers.RepeatedCompositeFieldContainer[NativeConsent]
    next_cursor: str
    def __init__(self, consents: _Optional[_Iterable[_Union[NativeConsent, _Mapping]]] = ..., next_cursor: _Optional[str] = ...) -> None: ...
