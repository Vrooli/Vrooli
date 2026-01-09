import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePromptDialog } from './usePromptDialog';

describe('usePromptDialog', () => {
  it('initializes with null dialog state', () => {
    const { result } = renderHook(() => usePromptDialog());
    expect(result.current.dialogState).toBeNull();
  });

  it('opens dialog with prompt() and resolves value on submit()', async () => {
    const { result } = renderHook(() => usePromptDialog());

    let promptPromise: Promise<string | null>;
    act(() => {
      promptPromise = result.current.prompt({
        title: 'Rename',
        label: 'New name',
        defaultValue: 'original',
        placeholder: 'Enter name...',
      });
    });

    // Dialog should be open
    expect(result.current.dialogState).not.toBeNull();
    expect(result.current.dialogState?.title).toBe('Rename');
    expect(result.current.dialogState?.label).toBe('New name');
    expect(result.current.dialogState?.value).toBe('original');
    expect(result.current.dialogState?.placeholder).toBe('Enter name...');
    expect(result.current.dialogState?.isOpen).toBe(true);

    // Update value
    act(() => {
      result.current.setValue('updated');
    });
    expect(result.current.dialogState?.value).toBe('updated');

    // Submit
    act(() => {
      result.current.submit();
    });

    expect(result.current.dialogState).toBeNull();
    const value = await promptPromise!;
    expect(value).toBe('updated');
  });

  it('resolves null on close(null)', async () => {
    const { result } = renderHook(() => usePromptDialog());

    let promptPromise: Promise<string | null>;
    act(() => {
      promptPromise = result.current.prompt({
        title: 'Input',
        label: 'Value',
      });
    });

    act(() => {
      result.current.close(null);
    });

    const value = await promptPromise!;
    expect(value).toBeNull();
  });

  it('validates input before submit', async () => {
    const { result } = renderHook(() => usePromptDialog());

    act(() => {
      result.current.prompt(
        {
          title: 'Name',
          label: 'Enter name',
          defaultValue: '',
        },
        {
          validate: (value) => (value.trim() ? null : 'Name is required'),
        }
      );
    });

    // Try to submit empty value
    act(() => {
      result.current.submit();
    });

    // Dialog should still be open with error
    expect(result.current.dialogState?.isOpen).toBe(true);
    expect(result.current.dialogState?.error).toBe('Name is required');

    // Update with valid value
    act(() => {
      result.current.setValue('valid name');
    });
    expect(result.current.dialogState?.error).toBeNull();

    // Submit should work now
    act(() => {
      result.current.submit();
    });
    expect(result.current.dialogState).toBeNull();
  });

  it('normalizes input on submit', async () => {
    const { result } = renderHook(() => usePromptDialog());

    let promptPromise: Promise<string | null>;
    act(() => {
      promptPromise = result.current.prompt(
        {
          title: 'Name',
          label: 'Enter name',
          defaultValue: '  spaces around  ',
        },
        {
          normalize: (value) => value.trim().toLowerCase(),
        }
      );
    });

    act(() => {
      result.current.submit();
    });

    const value = await promptPromise!;
    expect(value).toBe('spaces around');
  });

  it('clears error when value changes', async () => {
    const { result } = renderHook(() => usePromptDialog());

    act(() => {
      result.current.prompt(
        {
          title: 'Name',
          label: 'Enter name',
          defaultValue: '',
        },
        {
          validate: (value) => (value ? null : 'Required'),
        }
      );
    });

    // Trigger validation error
    act(() => {
      result.current.submit();
    });
    expect(result.current.dialogState?.error).toBe('Required');

    // Changing value should clear error
    act(() => {
      result.current.setValue('something');
    });
    expect(result.current.dialogState?.error).toBeNull();
  });

  it('cancels previous dialog when opening a new one', async () => {
    const { result } = renderHook(() => usePromptDialog());

    let firstPromise: Promise<string | null>;
    let secondPromise: Promise<string | null>;

    act(() => {
      firstPromise = result.current.prompt({ title: 'First', label: 'First' });
    });

    act(() => {
      secondPromise = result.current.prompt({ title: 'Second', label: 'Second' });
    });

    // First should be cancelled
    const firstResult = await firstPromise!;
    expect(firstResult).toBeNull();

    // Second should be active
    expect(result.current.dialogState?.title).toBe('Second');

    act(() => {
      result.current.setValue('result');
      result.current.submit();
    });

    const secondResult = await secondPromise!;
    expect(secondResult).toBe('result');
  });
});
