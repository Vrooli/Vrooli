import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils';
import { SpatialGroup } from './SpatialGroup';

function makeController() {
  return {
    registerGroup: vi.fn(() => vi.fn()),
    pushScope: vi.fn(),
    popScope: vi.fn(),
  };
}

describe('SpatialGroup', () => {
  let controller: ReturnType<typeof makeController>;
  let controllerRef: { current: typeof controller };

  beforeEach(() => {
    controller = makeController();
    controllerRef = { current: controller };
  });

  it('registers a focus group for spatial mode and unregisters on unmount', () => {
    const unregister = vi.fn();
    controller.registerGroup.mockReturnValue(unregister);

    const { unmount } = render(
      <SpatialGroup mode="spatial" controllerRef={controllerRef as never} options={{ wrap: true }}>
        <button>One</button>
      </SpatialGroup>,
    );

    expect(controller.registerGroup).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      'spatial',
      { wrap: true },
    );
    expect(screen.getByText('One')).toBeInTheDocument();

    unmount();
    expect(unregister).toHaveBeenCalledTimes(1);
  });

  it('pushes and pops a modal scope for modal mode', () => {
    const { unmount } = render(
      <SpatialGroup mode="modal" controllerRef={controllerRef as never}>
        <button>Confirm</button>
      </SpatialGroup>,
    );

    expect(controller.pushScope).toHaveBeenCalledWith(expect.any(HTMLElement));
    expect(controller.popScope).not.toHaveBeenCalled();

    unmount();
    expect(controller.popScope).toHaveBeenCalledTimes(1);
  });

  it('is inert when the controller ref is not yet populated', () => {
    render(
      <SpatialGroup mode="grid" controllerRef={{ current: null } as never}>
        <span>child</span>
      </SpatialGroup>,
    );
    expect(controller.registerGroup).not.toHaveBeenCalled();
    expect(screen.getByText('child')).toBeInTheDocument();
  });

  it('applies a custom className to the wrapper', () => {
    render(
      <SpatialGroup mode="grid" controllerRef={controllerRef as never} className="grp">
        <span data-testid="c">child</span>
      </SpatialGroup>,
    );
    expect(screen.getByTestId('c').parentElement).toHaveClass('grp');
    expect(controller.registerGroup).toHaveBeenCalledWith(expect.any(HTMLElement), 'grid', undefined);
  });
});
