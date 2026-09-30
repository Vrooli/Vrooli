import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import type { Investigation } from '../../../types';
import { InvestigationsSection } from './InvestigationsSection';

vi.mock('./AutomaticTriggersSection', () => ({ AutomaticTriggersSection: () => <div>automatic trigger controls</div> }));
vi.mock('./InvestigationsPanel', () => ({ InvestigationsPanel: () => <div>embedded reports</div> }));
vi.mock('./InvestigationScriptsPanel', () => ({ InvestigationScriptsPanel: () => <div>embedded scripts</div> }));

describe('InvestigationsSection', () => {
  it('supports agent options, notes, searches, and navigation controls', async () => {
    const onSpawnAgent = vi.fn().mockResolvedValue(undefined);
    const onOpenScriptEditor = vi.fn();
    const investigations = [{ id: 'disk', status: 'completed', findings: 'disk issue' }] as unknown as Investigation[];
    render(<InvestigationsSection investigations={investigations} onOpenScriptEditor={onOpenScriptEditor} onSpawnAgent={onSpawnAgent} agents={[]} isSpawningAgent={false} />);
    expect(screen.getByText('INVESTIGATIONS')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('switch'));
    fireEvent.click(screen.getByTitle('Add note for agent context'));
    fireEvent.change(screen.getByLabelText('Optional note for agent context'), { target: { value: 'inspect disk' } });
    fireEvent.click(screen.getByRole('button', { name: 'Spawn Agent' }));
    await waitFor(() => { expect(onSpawnAgent).toHaveBeenCalledWith({ autoFix: true, note: 'inspect disk' }); });
    fireEvent.change(screen.getByPlaceholderText('Search reports...'), { target: { value: 'disk' } });
    fireEvent.change(screen.getByPlaceholderText('Search scripts...'), { target: { value: 'shell' } });
    fireEvent.click(screen.getByText('Reports'));
    expect(screen.queryByText('embedded reports')).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('Reports'));
    expect(screen.getByText('embedded reports')).toBeInTheDocument();
  });

  it('shows spawn failures and disabled spawning state', () => {
    render(<InvestigationsSection investigations={[]} onOpenScriptEditor={vi.fn()} onSpawnAgent={vi.fn()} agents={[{ id: 'a', status: 'running', startTime: '', autoFix: false }]} isSpawningAgent spawnAgentError="agent service unavailable" />);
    expect(screen.getByText('agent service unavailable')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Spawning...' })).toBeDisabled();
  });
});
