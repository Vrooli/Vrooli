# File roles

Tidiness Manager accepts an optional `.vrooli/file-roles.json` in the scanned
scenario. It declares structural facts, not policy; findings remain visible.

```json
{"roles":[{"glob":"src/wiring.py","role":"declarative-wiring"}]}
```

Valid roles are `production`, `test`, `test-support`, `generated`,
`composition-root`, and `declarative-wiring`. Manifest matches are applied
before the built-in filename conventions; generated markers remain recognized.

`DUPLICATED_BOILERPLATE` is informational and zero-debt. `DUPLICATED_CODE`
contributes `duplication_line_debt`: largest physical span times copies minus
one. The two metrics intentionally differ because structural boilerplate is
visible but not refactor debt.
