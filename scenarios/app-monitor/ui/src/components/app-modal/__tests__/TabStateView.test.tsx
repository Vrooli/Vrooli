import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TabStateView from '../TabStateView';

describe('TabStateView', () => {
  it('renders children when no guard state is active', () => {
    render(
      <TabStateView>
        <div>child content</div>
      </TabStateView>,
    );
    expect(screen.getByText('child content')).toBeInTheDocument();
  });

  it('shows loading state', () => {
    render(
      <TabStateView loading loadingMessage="Fetching data...">
        <div>child content</div>
      </TabStateView>,
    );
    expect(screen.getByText('Fetching data...')).toBeInTheDocument();
    expect(screen.queryByText('child content')).not.toBeInTheDocument();
  });

  it('shows error state with retry', async () => {
    const onRetry = vi.fn();
    const user = userEvent.setup();

    render(
      <TabStateView error="Something broke" onRetry={onRetry}>
        <div>child content</div>
      </TabStateView>,
    );

    expect(screen.getByText('Something broke')).toBeInTheDocument();
    expect(screen.queryByText('child content')).not.toBeInTheDocument();

    await user.click(screen.getByText('Retry'));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it('shows empty state', () => {
    render(
      <TabStateView empty emptyMessage="Nothing here">
        <div>child content</div>
      </TabStateView>,
    );
    expect(screen.getByText('Nothing here')).toBeInTheDocument();
    expect(screen.queryByText('child content')).not.toBeInTheDocument();
  });

  it('prioritizes loading over error', () => {
    render(
      <TabStateView loading error="Error" loadingMessage="Loading...">
        <div>child content</div>
      </TabStateView>,
    );
    expect(screen.getByText('Loading...')).toBeInTheDocument();
    expect(screen.queryByText('Error')).not.toBeInTheDocument();
  });
});
