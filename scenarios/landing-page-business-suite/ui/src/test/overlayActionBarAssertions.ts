import { screen } from '@testing-library/react';
import { expect } from 'vitest';

export function expectActionBarOrder(primaryId: string, secondaryId: string): void {
  const bar = screen.getByRole('group', { name: 'Actions' });
  const buttons = Array.from(bar.querySelectorAll<HTMLButtonElement>('button[data-action-id]'));
  expect(buttons.map((button) => button.dataset.actionId)).toEqual([primaryId, secondaryId]);
  expect(buttons.map((button) => button.dataset.primary)).toEqual(['true', 'false']);
}
