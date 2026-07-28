import { MemoryRouter } from 'react-router-dom';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import AIPromptModal from './AIPromptModal';
import { selectors } from '@/constants/selectors';

vi.mock('@/domains/recording', () => ({
  BrowserUrlBar: () => <input aria-label="Starting URL" />,
}));

describe('AIPromptModal', () => {
  it('renders and invokes the manual-builder path when supplied', () => {
    const onSwitchToManual = vi.fn();
    render(
      <MemoryRouter>
        <AIPromptModal
          folder="/"
          onClose={vi.fn()}
          onSwitchToManual={onSwitchToManual}
        />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByTestId(selectors.ai.modal.switchToManualButton));

    expect(onSwitchToManual).toHaveBeenCalledOnce();
  });

  it('exposes each example prompt through the documented selector contract', () => {
    render(
      <MemoryRouter>
        <AIPromptModal folder="/" onClose={vi.fn()} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId(selectors.ai.modal.examplePrompts.first)).toBeVisible();
  });
});
