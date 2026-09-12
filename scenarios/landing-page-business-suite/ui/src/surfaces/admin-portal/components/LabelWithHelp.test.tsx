import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from "@vrooli/api-base/testing";
import { LabelWithHelp } from './LabelWithHelp';

describe('LabelWithHelp', () => {
  it('links the label, exposes contextual help, and lets the operator dismiss it', () => {
    renderWithProviders(<LabelWithHelp label="SMTP host" help="Use the hostname supplied by your provider." htmlFor="smtp-host" className="settings-label" />);

    expect(screen.getByText('SMTP host').closest('label')).toHaveAttribute('for', 'smtp-host');
    expect(screen.getByText('SMTP host').parentElement).toHaveClass('settings-label');
    expect(screen.queryByText('Use the hostname supplied by your provider.')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Help for SMTP host' }));
    expect(screen.getByText('Use the hostname supplied by your provider.')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Close help' }));
    expect(screen.queryByText('Use the hostname supplied by your provider.')).not.toBeInTheDocument();
  });

  it('uses the default wrapper class when no optional class is supplied', () => {
    renderWithProviders(<LabelWithHelp label="Support email" help="Replies are sent here." />);
    expect(screen.getByText('Support email').parentElement).toHaveClass('relative');
  });
});
