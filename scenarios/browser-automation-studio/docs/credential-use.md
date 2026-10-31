# Credential-use execution boundary

Browser Automation Studio has a separate protected execution policy for
password-manager login tasks. The policy binds an exact HTTP origin and page
document, and it allows only the operations listed by the authority. Ordinary
workflow actions continue to support broad automation features, but a
credential-use request is rejected if it contains arbitrary evaluation,
extraction, cookie/storage access, or other page-state reads.

Credential input is supplied by the trusted executor boundary. It is not a BAS
workflow variable, initial store value, screenshot artifact, or action result.
The policy returns an explicit unsupported exposure error for an unrestricted
session request. A page can observe fields it receives, so this contract
describes constrained agent access rather than model isolation or a guarantee
about every account action a site may perform.
