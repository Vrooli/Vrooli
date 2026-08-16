import { MemoryRouter } from 'react-router-dom';
import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import AIPromptModal from './AIPromptModal';
import { selectors } from '@/constants/selectors';
import { renderWithProviders } from '@/test-utils';

vi.mock('@/domains/recording', () => ({
  BrowserUrlBar: () => <input aria-label="Starting URL" />,
}));

describe('AIPromptModal', () => {
  it('renders and invokes the manual-builder path when supplied', () => {
    const onSwitchToManual = vi.fn();
    renderWithProviders(
      <MemoryRouter>
        <AIPromptModal
          folder="/"
          onClose={vi.fn()}
          onSwitchToManual={onSwitchToManual}
        />
      </MemoryRouter>,
      { withoutRouter: true },
    );

    fireEvent.click(screen.getByTestId(selectors.ai.modal.switchToManualButton));

    expect(onSwitchToManual).toHaveBeenCalledOnce();
  });

  it('exposes each example prompt through the documented selector contract', () => {
    renderWithProviders(
      <MemoryRouter>
        <AIPromptModal folder="/" onClose={vi.fn()} />
      </MemoryRouter>,
      { withoutRouter: true },
    );

    expect(screen.getByTestId(selectors.ai.modal.examplePrompts.first)).toBeVisible();
  });
});
