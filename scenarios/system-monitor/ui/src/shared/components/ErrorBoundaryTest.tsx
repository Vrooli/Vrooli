import { useState } from 'react';
import { Bug } from 'lucide-react';

/** Development-only harness for manually exercising the error boundary. */
export const ErrorBoundaryTest = () => {
  const [shouldThrow, setShouldThrow] = useState(false);

  if (shouldThrow) {
    throw new Error('Test error thrown by ErrorBoundaryTest component for demonstration purposes');
  }

  if (process.env.NODE_ENV !== 'development') return null;

  return (
    <div data-sm-style="sm-style-5b492864e3">
      <h4 data-sm-style="sm-style-87192fae5467">
        <Bug size={16} />
        Error Boundary Test
      </h4>
      <p data-sm-style="sm-style-d3e418452c">
        Click to test error boundary functionality (development only)
      </p>
      <button type="button" onClick={() => { setShouldThrow(true); }} data-sm-style="sm-style-c15c78d6dc">
        <Bug size={12} />
        Throw Error
      </button>
    </div>
  );
};
