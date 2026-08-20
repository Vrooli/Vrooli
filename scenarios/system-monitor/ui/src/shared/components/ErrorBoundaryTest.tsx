import { useState } from 'react';
import { Bug } from 'lucide-react';

/**
 * Test component to demonstrate Error Boundary functionality
 * This should only be used in development mode
 */
export const ErrorBoundaryTest = () => {
  const [shouldThrow, setShouldThrow] = useState(false);

  // This will throw an error when shouldThrow is true
  if (shouldThrow) {
    throw new Error('Test error thrown by ErrorBoundaryTest component for demonstration purposes');
  }

  // Only show in development mode
  if (process.env.NODE_ENV !== 'development') {
    return null;
  }

  return (
    <div data-sm-style="sm-style-5b492864e3">
      <h4 data-sm-style="sm-style-87192e5467">
        <Bug size={16} />
        Error Boundary Test
      </h4>
      
      <p data-sm-style="sm-style-d3e418452c">
        Click to test error boundary functionality (development only)
      </p>
      
      <button
        onClick={() => { setShouldThrow(true); }}
        data-sm-style="sm-style-c15c78d6dc"
      >
        <Bug size={12} />
        Throw Error
      </button>
    </div>
  );
};