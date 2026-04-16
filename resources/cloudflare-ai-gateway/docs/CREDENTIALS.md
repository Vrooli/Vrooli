# Credentials

Reference the secret source for `cloudflare-ai-gateway` here. Do not commit real secrets.

Keep `resource.json` as the declarative credential contract.

The native Go credential resolver now:

- prefers `resource-vault content get` for `resources/cloudflare/account_id` and `resources/cloudflare/api_token`
- falls back to `CLOUDFLARE_ACCOUNT_ID` and `CLOUDFLARE_API_TOKEN`
- keeps token handling redacted for operator-facing status and debug output

If Cloudflare-specific validation or translation grows further, extend `cli/internal/auth` rather than reintroducing shell helpers.
