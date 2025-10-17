(function (global) {
  const STYLE_ID = 'auth-user-card-styles';
  const CARD_CSS = `
    .user-summary[data-auth-user-card] {
      margin-left: auto;
      display: flex;
      align-items: center;
      gap: 18px;
      padding: 18px 22px;
      background: var(--surface-subtle, #f1f5f9);
      border: 1px solid var(--border, #e2e8f0);
      border-radius: 20px;
      transition: background 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
    }

    .user-summary[data-auth-user-card]:hover {
      background: rgba(15, 23, 42, 0.02);
      box-shadow: 0 12px 28px rgba(15, 23, 42, 0.12);
    }

    .user-summary__meta {
      display: flex;
      flex-direction: column;
      gap: 6px;
      text-align: right;
      min-width: 0;
    }

    .user-summary__name {
      font-size: 16px;
      font-weight: 600;
      color: var(--text-strong, #0f172a);
      max-width: 220px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .user-summary__roles {
      font-size: 13px;
      color: var(--muted-accent, #94a3b8);
      max-width: 220px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .user-summary__actions {
      display: flex;
      justify-content: flex-end;
      gap: 10px;
      flex-wrap: wrap;
      margin-top: 4px;
    }

    .user-summary__action {
      border: none;
      border-radius: 999px;
      padding: 8px 16px;
      font-size: 13px;
      font-weight: 600;
      cursor: pointer;
      transition: transform 0.2s ease, background 0.2s ease, box-shadow 0.2s ease;
      display: inline-flex;
      align-items: center;
      gap: 6px;
    }

    .user-summary__action:focus-visible {
      outline: 2px solid var(--primary, #2563eb);
      outline-offset: 2px;
    }

    .user-summary__action--profile {
      color: var(--primary, #2563eb);
      background: rgba(37, 99, 235, 0.12);
    }

    .user-summary__action--profile:hover {
      background: rgba(37, 99, 235, 0.18);
      transform: translateY(-1px);
    }

    .user-summary__action--admin {
      background: var(--primary, #2563eb);
      color: #ffffff;
      box-shadow: 0 10px 24px rgba(37, 99, 235, 0.26);
    }

    .user-summary__action--admin:hover {
      transform: translateY(-1px);
      box-shadow: 0 16px 30px rgba(37, 99, 235, 0.28);
    }

    .user-summary__avatar {
      width: 56px;
      height: 56px;
      border-radius: 18px;
      display: grid;
      place-items: center;
      font-size: 20px;
      font-weight: 600;
      color: #ffffff;
      background: linear-gradient(135deg, #2563eb 0%, #7c3aed 100%);
      border: none;
      cursor: pointer;
      transition: transform 0.2s ease, box-shadow 0.2s ease;
    }

    .user-summary__avatar:hover {
      transform: translateY(-1px);
      box-shadow: 0 16px 38px rgba(37, 99, 235, 0.28);
    }

    .user-summary__avatar:focus-visible {
      outline: 2px solid #ffffff;
      outline-offset: 3px;
      box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.6);
    }

    .user-summary__avatar--disabled {
      cursor: default;
      opacity: 0.65;
      box-shadow: none !important;
      transform: none !important;
    }

    @media (max-width: 640px) {
      .user-summary[data-auth-user-card] {
        width: 100%;
        justify-content: space-between;
        padding: 16px 20px;
      }

      .user-summary__meta {
        text-align: left;
      }

      .user-summary__name,
      .user-summary__roles {
        max-width: 180px;
      }
    }

    @media (prefers-reduced-motion: reduce) {
      .user-summary[data-auth-user-card],
      .user-summary__action,
      .user-summary__avatar {
        transition: none !important;
      }
    }
  `;

  function injectStyles() {
    if (document.getElementById(STYLE_ID)) {
      return;
    }
    const styleTag = document.createElement('style');
    styleTag.id = STYLE_ID;
    styleTag.textContent = CARD_CSS;
    document.head.appendChild(styleTag);
  }

  function normalizeRoles(roles) {
    if (!roles || (Array.isArray(roles) && roles.length === 0)) {
      return 'No assigned roles';
    }

    if (!Array.isArray(roles)) {
      return String(roles);
    }

    const uniqueRoles = Array.from(
      new Set(
        roles
          .filter(Boolean)
          .map((role) => String(role).trim())
          .filter((role) => role.length > 0)
      )
    );

    if (uniqueRoles.length === 0) {
      return 'No assigned roles';
    }

    return uniqueRoles
      .map((role) => role.charAt(0).toUpperCase() + role.slice(1))
      .join(' • ');
  }

  function computeAvatarText(name, email) {
    const source = name || email || '';
    if (!source) {
      return '?';
    }

    const trimmed = source.trim();
    if (!trimmed) {
      return '?';
    }

    const tokens = trimmed.split(/\s|@|\./).filter(Boolean);
    if (tokens.length === 0) {
      return trimmed.slice(0, 2).toUpperCase();
    }

    const initials = tokens.slice(0, 2).map((token) => token[0].toUpperCase());
    return initials.join('');
  }

  function mount(container, options = {}) {
    if (!container) {
      throw new Error('AuthUserCard.mount requires a valid container element');
    }

    injectStyles();

    const root = document.createElement('div');
    root.className = 'user-summary';
    root.setAttribute('data-auth-user-card', 'true');
    root.innerHTML = `
      <div class="user-summary__meta">
        <span class="user-summary__name" data-element="name">Loading…</span>
        <span class="user-summary__roles" data-element="roles">Verifying access…</span>
        <div class="user-summary__actions">
          <button type="button" class="user-summary__action user-summary__action--profile" data-element="profileButton">
            Manage Profile
          </button>
          <button type="button" class="user-summary__action user-summary__action--admin" data-element="adminButton">
            Admin Console
          </button>
        </div>
      </div>
      <button type="button" class="user-summary__avatar" data-element="avatar" aria-label="Manage profile">
        <span data-element="avatarText">?</span>
      </button>
    `;

    container.innerHTML = '';
    container.appendChild(root);

    const nameElement = root.querySelector('[data-element="name"]');
    const rolesElement = root.querySelector('[data-element="roles"]');
    const profileButton = root.querySelector('[data-element="profileButton"]');
    const adminButton = root.querySelector('[data-element="adminButton"]');
    const avatarButton = root.querySelector('[data-element="avatar"]');
    const avatarText = root.querySelector('[data-element="avatarText"]');

    const labels = options.labels || {};
    profileButton.textContent = labels.profile ? String(labels.profile) : 'Manage Profile';
    adminButton.textContent = labels.admin ? String(labels.admin) : 'Admin Console';

    let profileHandler = typeof options.onProfile === 'function' ? options.onProfile : null;
    let adminHandler = typeof options.onAdmin === 'function' ? options.onAdmin : null;

    function handleProfile(event) {
      if (profileHandler) {
        event.preventDefault();
        profileHandler();
      }
    }

    function handleAvatarKey(event) {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        handleProfile(event);
      }
    }

    profileButton.addEventListener('click', handleProfile);
    avatarButton.addEventListener('click', handleProfile);
    avatarButton.addEventListener('keydown', handleAvatarKey);

    adminButton.addEventListener('click', (event) => {
      if (!adminHandler) {
        return;
      }
      event.preventDefault();
      adminHandler();
    });

    function setProfileHandler(handler) {
      profileHandler = typeof handler === 'function' ? handler : null;
      const isActive = Boolean(profileHandler);
      profileButton.hidden = !isActive;
      avatarButton.setAttribute('aria-disabled', String(!isActive));
      avatarButton.tabIndex = isActive ? 0 : -1;
      if (!isActive) {
        avatarButton.classList.add('user-summary__avatar--disabled');
      } else {
        avatarButton.classList.remove('user-summary__avatar--disabled');
      }
    }

    function setAdminHandler(handler) {
      adminHandler = typeof handler === 'function' ? handler : null;
      adminButton.disabled = !adminHandler;
    }

    function setAdminVisible(isVisible) {
      adminButton.hidden = !isVisible;
    }

    function update(user = {}) {
      const email = user.email || user.userEmail || '';
      const displayName = user.displayName || user.name || user.username || email || 'Authenticated User';
      nameElement.textContent = displayName;
      nameElement.title = email && displayName !== email ? `${displayName} (${email})` : displayName;

      const roles = user.roles || user.userRoles;
      rolesElement.textContent = normalizeRoles(roles);
      rolesElement.title = rolesElement.textContent;

      const avatarValue = user.avatarText || computeAvatarText(displayName, email);
      avatarText.textContent = avatarValue;
    }

    setProfileHandler(profileHandler);
    setAdminHandler(adminHandler);
    setAdminVisible(Boolean(adminHandler));

    return {
      update,
      setProfileHandler,
      setAdminHandler,
      setAdminVisible,
      getRoot: () => root,
    };
  }

  global.AuthUserCard = {
    mount,
  };
})(window);
