# Troubleshooting

If the UI shows no scenarios, verify repository manifests are available to the
scenario process. If credentials are unconfigured, provision the exact declared
logical identity through onboarding or `vrooli credentials provision` on stdin.
If the native authority is unsupported, use a supported target; plaintext files
and Vault fallback reads are not supported.
