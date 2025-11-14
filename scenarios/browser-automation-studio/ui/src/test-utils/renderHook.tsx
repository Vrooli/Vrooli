// Simplified hook rendering utility for testing custom hooks
import { render } from '@testing-library/react';

interface RenderHookResult<TResult, TProps> {
  result: { current: TResult };
  rerender: (props?: TProps) => void;
  unmount: () => void;
}

/**
 * Simplified renderHook utility for testing React hooks.
 * Renders the hook inside a test component and provides access to the result.
 */
export function renderHook<TResult, TProps = Record<string, never>>(
  hook: (props: TProps) => TResult,
  options?: {
    initialProps?: TProps;
    wrapper?: React.ComponentType<{ children: React.ReactNode }>;
  }
): RenderHookResult<TResult, TProps> {
  const result = { current: undefined as unknown as TResult };

  function TestComponent({ hookProps }: { hookProps: TProps }) {
    result.current = hook(hookProps);
    return null;
  }

  const Wrapper = options?.wrapper;
  const initialProps = options?.initialProps ?? ({} as TProps);

  const renderResult = render(
    Wrapper ? (
      <Wrapper>
        <TestComponent hookProps={initialProps} />
      </Wrapper>
    ) : (
      <TestComponent hookProps={initialProps} />
    )
  );

  return {
    result,
    rerender: (props?: TProps) => {
      const newProps = props ?? initialProps;
      renderResult.rerender(
        Wrapper ? (
          <Wrapper>
            <TestComponent hookProps={newProps} />
          </Wrapper>
        ) : (
          <TestComponent hookProps={newProps} />
        )
      );
    },
    unmount: renderResult.unmount,
  };
}
