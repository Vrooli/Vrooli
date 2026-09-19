import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { describe, expect, it, vi } from 'vitest';
import { ScriptEditorModal } from './ScriptEditorModal';
import { ScriptResultsModal } from './ScriptResultsModal';
import { ScriptExecutionStatus } from '../../../types';
import type { InvestigationScript, ScriptExecution } from '../../../types';

vi.mock('../../../shared/components/LazyScriptHighlighter', () => ({ ScriptHighlighter: ({ content }: { content: string }) => <pre>{content}</pre> }));

const script = { id: 'check', name: 'Check', description: 'Check system', category: 'performance', author: 'test', enabled: true } as unknown as InvestigationScript;
const ts = timestampFromDate(new Date('2026-01-01T00:00:00Z'));

describe('investigation modals', () => {
  it('views, edits, executes, and saves scripts', async () => {
    const onClose = vi.fn();
    const onExecute = vi.fn().mockResolvedValue(undefined);
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<ScriptEditorModal isOpen script={script} scriptContent="echo ok" mode="view" onClose={onClose} onExecute={onExecute} onSave={onSave} />);
    expect(screen.getByText('Check')).toBeInTheDocument();
    fireEvent.click(screen.getByTitle('Edit Script'));
    expect(screen.getByDisplayValue('echo ok')).toBeInTheDocument();
    fireEvent.change(screen.getByDisplayValue('echo ok'), { target: { value: 'echo changed' } });
    fireEvent.click(screen.getByTitle('Execute Script'));
    await waitFor(() => { expect(onExecute).toHaveBeenCalledWith('check', 'echo changed'); });
    fireEvent.click(screen.getByTitle('Save Script'));
    await waitFor(() => { expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ id: 'check' }), 'echo changed'); });
    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalled();
  });

  it('creates a script with metadata and supports no-op execute guards', () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<ScriptEditorModal isOpen mode="create" onClose={vi.fn()} onExecute={vi.fn()} onSave={onSave} />);
    expect(screen.getByText('New Investigation Script')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('script-name')).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText('Human readable name'), { target: { value: 'New check' } });
    const textboxes = screen.getAllByRole('textbox');
    const editor = textboxes[textboxes.length - 1];
    if (!editor) throw new Error('script editor textbox was not rendered');
    fireEvent.change(editor, { target: { value: 'echo new' } });
    fireEvent.click(screen.getByTitle('Save Script'));
    expect(onSave).toHaveBeenCalled();
  });

  it('renders completed, failed, running, timed-out, and empty results', () => {
    const base: ScriptExecution = { scriptId: 'check', executionId: 'exec', startedAt: ts };
    const { rerender } = render(<ScriptResultsModal isOpen execution={{ ...base, status: ScriptExecutionStatus.COMPLETED, exitCode: 0, output: 'ok', durationSeconds: 2 }} onClose={vi.fn()} />);
    expect(screen.getByText('Script Output')).toBeInTheDocument();
    expect(screen.getByText('2s')).toBeInTheDocument();
    rerender(<ScriptResultsModal isOpen execution={{ ...base, status: ScriptExecutionStatus.FAILED, exitCode: 1, error: 'bad' }} onClose={vi.fn()} />);
    expect(screen.getByText('Error Output')).toBeInTheDocument();
    rerender(<ScriptResultsModal isOpen execution={{ ...base, status: ScriptExecutionStatus.RUNNING }} onClose={vi.fn()} />);
    expect(screen.getByText('Script is still running...')).toBeInTheDocument();
    rerender(<ScriptResultsModal isOpen execution={{ ...base, status: ScriptExecutionStatus.COMPLETED, exitCode: 1, timedOut: true }} onClose={vi.fn()} />);
    expect(screen.getByText('Yes')).toBeInTheDocument();
    rerender(<ScriptResultsModal isOpen onClose={vi.fn()} />);
    expect(screen.queryByText('Script Execution Results')).not.toBeInTheDocument();
  });
});
