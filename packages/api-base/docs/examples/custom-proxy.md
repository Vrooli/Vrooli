# Custom Proxy Pattern Example

Implementing custom proxy patterns beyond the standard `/apps/{name}/proxy`.

## Custom Path Pattern

Use **any** path pattern - not just `/apps/`:

```javascript
import { buildProxyMetadata, injectProxyMetadata } from '@vrooli/api-base/server'

// Custom patterns that work:
const patterns = [
  '/widgets/{name}/proxy',
  '/preview/{name}/proxy',
  '/embed/{name}/proxy',
  '/tools/{name}/proxy',
  '/dashboards/{name}/proxy',
]

// Example: /widgets/analyzer/proxy
const metadata = buildProxyMetadata({
  appId: 'analyzer',
  basePath: '/widgets/analyzer/proxy',
  ports: [{ port: 3000, label: 'ui', slug: 'ui' }],
  primaryPort: { port: 3000, label: 'ui', slug: 'ui' },
})
```

**Client automatically detects** any `/proxy` marker:

```typescript
// URL: https://dashboard.com/widgets/analyzer/proxy/tools
// Pathname: /widgets/analyzer/proxy/tools

resolveApiBase()
// Finds "/proxy" → extracts /widgets/analyzer/proxy
// → "https://dashboard.com/widgets/analyzer/proxy"
```

---

## Custom Global Names

Use custom proxy global names:

**Server**:
```javascript
// Inject with custom global name
const html = '<html>...</html>'
const script = `
<script>
  window.__MY_CUSTOM_PROXY__ = ${JSON.stringify(metadata)};
</script>
`
const modifiedHtml = html.replace('<head>', `<head>${script}`)
```

**Client**:
```typescript
resolveApiBase({
  proxyGlobalNames: ['__MY_CUSTOM_PROXY__']
})
```

---

## See Also

- [Basic Scenario Example](./basic-scenario.md)
- [Embedded Scenario Example](./embedded-scenario.md)
- [Proxy Resolution Concept](../concepts/proxy-resolution.md)
