# Typed inference JSON Schema subset

The ai-gateway `InferenceService` accepts a deliberately small JSON Schema
subset. The gate rejects a request before provider execution when the schema
contains a construct outside this list. A rejected schema is never silently
degraded to unconstrained generation.

Supported constructs:

- `type` with one of `object`, `array`, `string`, `number`, `integer`, `boolean`, or `null`.
- `enum` and `const`.
- `required`, `properties`, and `items` for object and array structure.
- `pattern` for regular-expression constrained strings.
- `minimum` and `maximum` for numeric values.
- `$id`, `$schema`, `title`, and `description` as metadata only.

`description` is not an instruction channel. Ollama does not reliably carry
schema descriptions into the prompt, so caller intent must be supplied through
the request's first-class `instruction` field.

The following constructs are intentionally unsupported and are named in the
typed error returned by the gate:

- Composition and conditional logic: `allOf`, `anyOf`, `oneOf`, `not`, `if`, `then`, and `else`.
- Object policy and dependent constraints: `additionalProperties`, `patternProperties`, `propertyNames`, `dependentRequired`, and `dependentSchemas`.
- Size and string constraints: `minLength`, `maxLength`, `minItems`, `maxItems`, `uniqueItems`, and `format`.
- Numeric constraints beyond inclusive bounds: `exclusiveMinimum`, `exclusiveMaximum`, and `multipleOf`.
- External or recursive schemas: `$ref`, `$defs`, and `definitions`.

Provider output is advisory. ai-gateway parses the returned JSON and validates
it locally against the submitted schema. Only a locally valid value is marked
`validated: true`; malformed or schema-violating output returns a typed
validation failure with usage and provider provenance preserved.

## `locate.visual`

The gateway-owned `locate.visual` role uses the following request schema shape:

```json
{
  "type": "object",
  "required": ["found", "bounds", "confidence"],
  "properties": {
    "found": {"type": "boolean"},
    "bounds": {"type": "array", "items": {"type": "number"}},
    "confidence": {"type": "number", "minimum": 0, "maximum": 1}
  }
}
```

The gateway additionally requires exactly four bounds and normalizes the
provider response to `[left, top, right, bottom]` floats in `[0,1]` using the
resolved model's declared `coordinate_convention`. A caller supplies image
width and height on the attachment; coordinate conversion is rejected when
dimensions are missing or the converted bounds leave the image.
