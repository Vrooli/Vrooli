import { Building2, ChevronRight } from 'lucide-react';
import type { BusinessAccount } from '../../../shared/api';

interface AccountChooserProps {
  appName: string;
  accounts: BusinessAccount[];
  busyId: string | null;
  onChoose: (accountId: string) => void;
}

export function AccountChooser({ appName, accounts, busyId, onChoose }: AccountChooserProps) {
  return (
    <div className="auth-step" data-testid="desktop-account-selection">
      <header className="auth-head">
        <h1>Choose an account</h1>
        <p>Pick which account {appName} should use on this computer.</p>
      </header>
      <ul className="auth-choices">
        {accounts.map((account) => (
          <li key={account.id}>
            <button
              type="button"
              className="auth-choice"
              disabled={busyId !== null}
              aria-busy={busyId === account.id}
              onClick={() => { onChoose(account.id); }}
            >
              <span className="auth-choice-icon" aria-hidden="true"><Building2 /></span>
              <span className="auth-choice-text">
                <strong>{account.display_name}</strong>
                <small>{account.role.charAt(0).toUpperCase() + account.role.slice(1)}</small>
              </span>
              {busyId === account.id ? <span className="site-spinner" aria-hidden="true" /> : <ChevronRight aria-hidden="true" />}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
