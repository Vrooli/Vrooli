# Debug Component

A debugging utility for React components that provides a visual way to track state changes and debug information without cluttering the console.

## Features

- Non-intrusive visual feedback in the bottom right corner of the application
- Traces identified by unique IDs with distinct colors
- Visual indication (brightening effect) when data is updated
- Detailed information available on hover
- Only visible in development mode
- Optimized for minimal performance impact

## Usage

### Basic Usage

1. Import the debug store in any component where you want to track debug data:

```tsx
import { useDebugStore } from "../../stores/debugStore.js";

function MyComponent() {
  const addData = useDebugStore((state) => state.addData);
  
  const handleClick = () => {
    // Send debug data to the store
    addData("myComponent-click", { 
      timestamp: new Date().toISOString(),
      message: "Button clicked", 
    });
    
    // Rest of your click handler code
  };
  
  return <button onClick={handleClick}>Click Me</button>;
}
```

2. The `DebugComponent` is automatically rendered in the app in development mode only.

### Tracking Multiple Events

Use consistent trace IDs for the same type of event:

```tsx
// In a form component
function handleSubmit(formData) {
  addData("userForm-submit", { 
    valid: isValid,
    values: formData,
    timestamp: new Date().toISOString()
  });
  
  // Submit the form...
}

function handleValidation(errors) {
  addData("userForm-validation", { 
    errors,
    timestamp: new Date().toISOString()
  });
  
  // Handle validation...
}
```

### Tracing API Calls

```tsx
async function fetchUserData(userId) {
  addData("api-userFetch-request", { 
    userId,
    timestamp: new Date().toISOString()
  });
  
  try {
    const response = await fetch(`/api/users/${userId}`);
    const data = await response.json();
    
    addData("api-userFetch-response", { 
      status: response.status,
      data,
      timestamp: new Date().toISOString()
    });
    
    return data;
  } catch (error) {
    addData("api-userFetch-error", { 
      error: error.message,
      timestamp: new Date().toISOString()
    });
    
    throw error;
  }
}
```

## Best Practices

1. **Use meaningful trace IDs**: Create a consistent naming convention for trace IDs, such as `componentName-action` or `category-subcategory-action`.

2. **Include timestamps**: Always include a timestamp in your debug data to track when events occur.

3. **Limit data size**: While the debug component can handle large objects, it's best to keep the data concise for better readability.

4. **Conditionally enable tracing**: For performance-critical sections, consider enabling tracing conditionally:

```tsx
function processLargeDataSet(data) {
  // Only trace in development
  if (process.env.NODE_ENV === "development") {
    addData("dataCruncher-start", { 
      dataSize: data.length,
      timestamp: new Date().toISOString()
    });
  }
  
  // Process data...
  
  if (process.env.NODE_ENV === "development") {
    addData("dataCruncher-end", { 
      processingTime: Date.now() - startTime,
      timestamp: new Date().toISOString()
    });
  }
}
```

## Implementation Details

- Uses Zustand for state management
- Automatically excluded from production builds
- Generates unique colors for each trace based on the trace ID
- Visual feedback (brightening) when data is updated 