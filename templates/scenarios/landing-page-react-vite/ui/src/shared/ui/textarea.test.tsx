import { describe, it, expect, vi } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { createRef } from 'react';
import { renderWithProviders as render } from '../../test-utils';
import { Textarea } from './textarea';

describe('Textarea', () => {
  it('renders and forwards value changes', () => {
    const onChange = vi.fn();
    render(<Textarea placeholder="Notes" onChange={onChange} />);
    const el = screen.getByPlaceholderText('Notes');
    fireEvent.change(el, { target: { value: 'multi\nline' } });
    expect(onChange).toHaveBeenCalled();
    expect(el).toHaveValue('multi\nline');
  });

  it('passes through native attributes such as rows and disabled', () => {
    render(<Textarea aria-label="body" rows={6} disabled />);
    const el = screen.getByLabelText('body');
    expect(el).toHaveAttribute('rows', '6');
    expect(el).toBeDisabled();
  });

  it('merges custom className with base styles and forwards a ref', () => {
    const ref = createRef<HTMLTextAreaElement>();
    render(<Textarea ref={ref} aria-label="ref" className="custom-area" />);
    const el = screen.getByLabelText('ref');
    expect(el).toHaveClass('custom-area');
    expect(el.className).toContain('min-h-[80px]');
    expect(ref.current).toBeInstanceOf(HTMLTextAreaElement);
  });
});
