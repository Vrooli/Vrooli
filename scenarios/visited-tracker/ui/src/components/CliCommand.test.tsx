import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CliCommand } from './CliCommand';
import { FilePathWithCopy } from './FilePathWithCopy';

afterEach(() => vi.unstubAllGlobals());

describe('copy controls', () => {
  it('copies the complete command and announces success', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });
    const command = 'vrooli scenario status visited-tracker';
    render(<CliCommand command={command} description="Inspect runtime" />);
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Copy command to clipboard' })));
    expect(writeText).toHaveBeenCalledWith(command);
    expect(screen.getByRole('button', { name: 'Copied to clipboard!' })).toBeInTheDocument();
  });

  it('gives a manual fallback when clipboard access fails', async () => {
    vi.stubGlobal('navigator', {});
    render(<CliCommand command="visited-tracker campaigns list" />);
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Copy command to clipboard' })));
    expect(screen.getByRole('status')).toHaveTextContent('Select the text to copy it manually.');
    expect(screen.queryByRole('button', { name: 'Copied to clipboard!' })).not.toBeInTheDocument();
  });

  it('copies a file path without activating its containing row', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    const openRow = vi.fn();
    vi.stubGlobal('navigator', { clipboard: { writeText } });
    render(<div onClick={openRow}><FilePathWithCopy path="src/a file.ts" /></div>);
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Copy path' })));
    expect(writeText).toHaveBeenCalledWith('src/a file.ts');
    expect(openRow).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Copied!' })).toBeInTheDocument();
  });
});
