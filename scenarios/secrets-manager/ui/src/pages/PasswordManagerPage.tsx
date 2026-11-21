import { PasswordManagerApp, type PasswordManagerPageId } from "../PasswordManagerApp";

export function PasswordManagerPage({ page }: { page: PasswordManagerPageId }) {
  return <PasswordManagerApp initialPage={page} />;
}
