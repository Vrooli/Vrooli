# Brand design tokens

`BrandsService/GetTokens` is the stable read surface for downstream composition.
It accepts a brand id and returns named color tokens in deterministic order:
`$brand.primary`, `$brand.secondary`, `$brand.accent`, `$brand.background`,
`$brand.surface`, `$brand.text`, and `$brand.error`.

```sh
curl "$API_BASE/vrooli.brand_manager.v1.brands.BrandsService/GetTokens" \
  -H 'Content-Type: application/json' \
  -d '{"brand_id":"abc123"}'
```

The response is a versioned projection of the persisted brand colors. Consumers
must resolve the values at composition time; they must not read Brand Manager's
database or maintain a second palette authority.
