import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ErrorBoundaryTest } from './ErrorBoundaryTest';
import { ScriptHighlighter } from './LazyScriptHighlighter';

describe('development-only and lazy script surfaces', () => {
  afterEach(() => { vi.unstubAllEnvs(); });

  it('keeps the development error demonstrator hidden outside development', () => {
    vi.stubEnv('NODE_ENV', 'production');
    const { container, rerender } = render(<ErrorBoundaryTest />);
    expect(container).toBeEmptyDOMElement();
    vi.stubEnv('NODE_ENV', 'development');
    rerender(<ErrorBoundaryTest />);
    expect(screen.getByText('Error Boundary Test')).toBeInTheDocument();
  });

  it('renders readable script text while the highlighter is loading', async () => {
    render(<ScriptHighlighter content="echo hello" padding="1rem" />);
    expect(screen.getByText('echo hello')).toBeInTheDocument();
    await waitFor(() => { expect(screen.getByText('echo hello')).toBeInTheDocument(); });
  });

  it('can deliberately throw from the development demonstrator', () => {
    vi.stubEnv('NODE_ENV', 'development');
    render(<ErrorBoundaryTest />);
    expect(() => fireEvent.click(screen.getByRole('button', { name: /Throw Error/ }))).toThrow('Test error thrown');
  });
});
