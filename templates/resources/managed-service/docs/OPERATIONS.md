# Operations

Install and start this resource through the Vrooli control plane:

```bash
vrooli resource install {{RESOURCE_NAME}}
vrooli resource start {{RESOURCE_NAME}}
vrooli resource status {{RESOURCE_NAME}}
vrooli resource logs {{RESOURCE_NAME}}
```

The server artifact must be supplied by the authenticated Vrooli release. Attach-only and remote providers never grant local lifecycle authority.
