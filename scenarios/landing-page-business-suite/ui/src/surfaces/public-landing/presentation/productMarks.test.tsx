// provider-free-exception: ProductMark is an inert SVG primitive consuming a finite kind prop, with no context or provider behavior.
import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render } from '@testing-library/react';
import { getProductMarkPath, productMarkDrawing, productMarkPaths } from './productMarks.js';
import { BrandLogo, ProductMark } from './primitives';
import type { Mark } from './resources';
afterEach(cleanup);
describe('shared Node/browser product mark grammar', () => {
  it.each<Mark>(['letter-a', 'landscape', 'suite', 'play'])('uses the shared path and geometry for %s', kind => {
    const { container } = render(<ProductMark kind={kind} />);
    expect(container.querySelector('svg')).toHaveAttribute('viewBox', productMarkDrawing.viewBox);
    expect(container.querySelector('svg')).toHaveAttribute('fill', 'none');
    expect(container.querySelector('path')).toHaveAttribute('d', productMarkPaths[kind]);
    expect(container.querySelector('path')).toHaveAttribute('stroke-width', '2.3');
    expect(container.querySelector('path')).toHaveAttribute('stroke-linecap', 'round');
    expect(container.querySelector('path')).toHaveAttribute('stroke-linejoin', 'round');
  });
  it.each(['unknown', 'constructor', '__proto__', '', null, 1])('does not supply a fallback mark for %s', value => {
    expect(getProductMarkPath(value)).toBeUndefined();
  });
  it('keeps the shared catalog and geometry immutable', () => {
    expect(Object.isFrozen(productMarkPaths)).toBe(true); expect(Object.isFrozen(productMarkDrawing)).toBe(true);
  });
});
describe('configured brand logo images', () => {
  it('renders a same-origin logo image instead of the SVG mark', () => {
    const { container } = render(<BrandLogo kind="letter-a" logo="/public/apps/aquila.png" alt="Aquila" />);
    const img = container.querySelector('img');
    expect(img).toHaveAttribute('src', '/public/apps/aquila.png');
    expect(img).toHaveAttribute('alt', 'Aquila');
    expect(container.querySelector('svg')).toBeNull();
  });
  it('falls back to the SVG mark when no logo is configured', () => {
    const { container } = render(<BrandLogo kind="suite" />);
    expect(container.querySelector('path')).toHaveAttribute('d', productMarkPaths.suite);
    expect(container.querySelector('img')).toBeNull();
  });
  it.each(['https://evil.example/x.png', '//evil.example/x.png', 'javascript:alert(1)'])(
    'rejects the unsafe logo reference %s',
    logo => {
      expect(() => render(<BrandLogo kind="suite" logo={logo} />)).toThrow(/Unsafe product logo/);
    },
  );
});
