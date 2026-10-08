import { Component, type ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.unmock('./useLandingVariant');

import { useLandingVariant } from './useLandingVariant';

// provider-free-exception: this contract intentionally mounts the hook without
// LandingVariantProvider to verify the actionable missing-provider error.

class HookErrorBoundary extends Component<{ children: ReactNode }, { message: string }> {
  state = { message: '' };

  static getDerivedStateFromError(error: Error) {
    return { message: error.message };
  }

  render() {
    if (this.state.message) {
      return <p role="alert">{this.state.message}</p>;
    }
    return this.props.children;
  }
}

function LandingVariantConsumer() {
  useLandingVariant();
  return null;
}

describe('useLandingVariant', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('fails fast when used outside the landing variant provider', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    render(
      <HookErrorBoundary>
        <LandingVariantConsumer />
      </HookErrorBoundary>,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('useLandingVariant must be used within a LandingVariantProvider');
  });
});
