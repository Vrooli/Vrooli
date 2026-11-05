import { cleanup, screen } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { MemoryRouter, Route, Routes, useLocation, useParams } from 'react-router-dom';
import { afterEach, describe, expect, it } from 'vitest';
import { SectionEditor } from './SectionEditor';
afterEach(cleanup);
function Destination() {
  const { variantSlug } = useParams(); const location = useLocation();
  return <p>{location.pathname}:{variantSlug || 'choose'}</p>;
}
describe('legacy section bookmark migration', () => {
  it.each(['42', 'hero', 'new'])('retains the explicit variant for legacy section %s', async section => {
    render(<MemoryRouter basename="/proxy" initialEntries={['/proxy/admin/customization/variants/second/sections/' + section]}><Routes>
      <Route path="/admin/customization/variants/:variantSlug/sections/:sectionId" element={<SectionEditor />} />
      <Route path="/admin/presentation/:variantSlug" element={<Destination />} />
    </Routes></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByText('/admin/presentation/second:second')).toBeInTheDocument();
  });
  it('requires selection when no legacy variant is present', async () => {
    render(<MemoryRouter initialEntries={['/legacy']}><Routes><Route path="/legacy" element={<SectionEditor />} /><Route path="/admin/presentation" element={<Destination />} /></Routes></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByText('/admin/presentation:choose')).toBeInTheDocument();
  });
});
