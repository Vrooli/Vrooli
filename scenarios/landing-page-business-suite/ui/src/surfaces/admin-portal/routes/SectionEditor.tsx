import { Navigate, useParams } from 'react-router-dom';
import { presentationEditorPath } from '../config/navigation.utils';

/** Authenticated legacy bookmarks retain their variant, not a legacy section lookup. */
export function SectionEditor() {
  const { variantSlug } = useParams();
  return <Navigate replace to={presentationEditorPath(variantSlug)} />;
}
