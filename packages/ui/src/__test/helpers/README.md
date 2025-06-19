# Storybook Testing Helpers

This directory contains helper utilities for writing reliable and maintainable Storybook stories.

## API Mocking Helpers

### `storybookMocking.ts`

Provides helper functions to construct MSW mock URLs more reliably in Storybook stories.

#### Benefits

- **Type Safety**: Leverages TypeScript to catch endpoint configuration errors
- **Consistency**: Ensures all mock URLs follow the same pattern
- **Maintainability**: Single source of truth for URL construction logic
- **Reliability**: Eliminates string interpolation errors in story files

#### Functions

##### `getMockUrl(endpointConfig)`

For direct endpoint configurations like `endpointsResource.findApiVersion`.

```typescript
// Before
http.get(`${API_URL}/v2${endpointsResource.findApiVersion.endpoint}`, handler)

// After
http.get(getMockUrl(endpointsResource.findApiVersion), handler)
```

##### `getMockUrlNested(endpointConfig)`

For nested endpoint configurations where you need to access a specific sub-endpoint.

```typescript
// Before
http.get(`${API_URL}/v2${endpointsResource.findDataStructureVersion.findOne.endpoint}`, handler)

// After
http.get(getMockUrlNested(endpointsResource.findDataStructureVersion.findOne), handler)
```

##### `getMockEndpoint(endpointOrNested)` (Recommended)

Generic helper that automatically handles both simple and nested endpoint configurations.

```typescript
// Works with direct endpoints
http.get(getMockEndpoint(endpointsResource.findApiVersion), handler)

// Works with nested endpoints  
http.get(getMockEndpoint(endpointsResource.findDataStructureVersion), handler)
```

#### Usage Examples

```typescript
import { endpointsResource } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { getMockUrl, getMockEndpoint } from "../__test/helpers/storybookMocking.js";

export function MyStory() {
    // Story component
}

MyStory.parameters = {
    msw: {
        handlers: [
            // Direct endpoint
            http.get(getMockUrl(endpointsResource.findApiVersion), () => {
                return HttpResponse.json({ data: mockData });
            }),
            
            // Nested endpoint (automatically detects .findOne)
            http.get(getMockEndpoint(endpointsResource.findDataStructureVersion), () => {
                return HttpResponse.json({ data: mockData });
            }),
        ],
    },
};
```

#### Route Path Helpers

##### `getStoryRoutePath(mockData)`

For view page routes.

```typescript
// Before
route: {
    path: `${API_URL}/v2${getObjectUrl(mockApiVersionData)}`,
}

// After
route: {
    path: getStoryRoutePath(mockApiVersionData),
}
```

##### `getStoryRouteEditPath(mockData)`

For edit page routes.

```typescript
// Before
route: {
    path: `${API_URL}/v2${getObjectUrl(mockApiVersionData)}/edit`,
}

// After
route: {
    path: getStoryRouteEditPath(mockApiVersionData),
}
```

##### `getStoryRoutePathWithQuery(mockData, queryParams)`

For routes with query parameters.

```typescript
// Before
route: {
    path: `${API_URL}/v2${getObjectUrl(mockData)}?runId=${generatePK().toString()}`,
}

// After
route: {
    path: getStoryRoutePathWithQuery(mockData, { runId: generatePK().toString() }),
}
```

#### Migration Guide

1. Import the helper functions in your story file:
   ```typescript
   import { getMockEndpoint, getStoryRoutePath, getStoryRouteEditPath } from "../../../__test/helpers/storybookMocking.js";
   ```

2. Replace MSW handler string interpolation with helper calls:
   ```typescript
   // Replace this pattern:
   http.get(`${API_URL}/v2${endpointsResource.findApiVersion.endpoint}`, handler)
   
   // With this:
   http.get(getMockEndpoint(endpointsResource.findApiVersion), handler)
   ```

3. Replace route path string interpolation with helper calls:
   ```typescript
   // Replace this pattern:
   route: {
       path: `${API_URL}/v2${getObjectUrl(mockApiVersionData)}`,
   }
   
   // With this:
   route: {
       path: getStoryRoutePath(mockApiVersionData),
   }
   ```

4. For nested endpoints, the helper automatically detects the structure:
   ```typescript
   // Replace this pattern:
   http.get(`${API_URL}/v2${endpointsResource.findDataStructureVersion.findOne.endpoint}`, handler)
   
   // With this:
   http.get(getMockEndpoint(endpointsResource.findDataStructureVersion), handler)
   ```