import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import VariableSuggestionList from './VariableSuggestionList';
import type { WorkflowVariableInfo } from '@hooks/useWorkflowVariables';

describe('VariableSuggestionList', () => {
  const baseVariable: WorkflowVariableInfo = {
    name: 'token',
    sourceNodeId: 'node-a',
    sourceType: 'setVariable',
    sourceLabel: 'Set Variable name • Set Variable',
  };

  it('renders hint when no variables are present', () => {
    render(<VariableSuggestionList variables={[]} emptyHint="define one" />);
    expect(screen.getByTestId('variable-suggestions-empty')).toHaveTextContent('define one');
  });

  it('renders variable chips and notifies selection', () => {
    const selectSpy = vi.fn();
    render(
      <VariableSuggestionList
        variables={[baseVariable, { ...baseVariable, name: 'alias', sourceNodeId: 'node-b' }]}
        onSelect={selectSpy}
      />
    );

    const chip = screen.getByText('token');
    expect(chip).toHaveAttribute('title', baseVariable.sourceLabel);
    fireEvent.click(chip);
    expect(selectSpy).toHaveBeenCalledWith('token');

    const chips = screen.getAllByRole('button');
    expect(chips).toHaveLength(2);
  });
});
