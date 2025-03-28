# Vrooli UI

The frontend application for Vrooli, built with React, TypeScript, and Material-UI.

## Overview

This package contains the user interface for Vrooli, implementing a modern, responsive web application that supports both desktop and mobile experiences. It's built as a Progressive Web App (PWA) for optimal performance and offline capabilities.

## Technology Stack

- **Framework**: React 18+
- **Language**: TypeScript
- **Build Tool**: Vite
- **UI Library**: Material-UI (MUI)
- **State Management**: Zustand, React contexts
- **Testing**: Mocha + Chai + Sinon + React Testing Library
- **Component Documentation**: Storybook
- **PWA Support**: Workbox
- **Internationalization**: i18next
- **Forms**: Formik

## Directory Structure

```
ui/
├── src/
│   ├── api/          # API client and endpoints
│   ├── assets/       # Static assets
│   ├── components/   # Reusable components
│   ├── contexts/     # React contexts
│   ├── forms/        # Form components and validation
│   ├── hooks/        # Custom React hooks
│   ├── icons/        # SVG icons and icon components
│   ├── route/        # Routing configuration
│   ├── stores/       # State management
│   ├── utils/        # Utility functions
│   ├── views/        # Page components
│   ├── i18n.ts      # Internationalization setup
│   └── sw-template.js # Service worker template
├── public/           # Static assets
├── .storybook/       # Storybook configuration
├── config/           # Configuration files
└── tests/            # Test files
```

## Getting Started

1. Install dependencies:
   ```bash
   yarn install
   ```

2. Start development server:
   ```bash
   yarn dev
   ```

3. Run tests:
   ```bash
   yarn test
   ```

4. Build for production:
   ```bash
   yarn build
   ```

## Development

### Key Features

- **Component-Based Architecture**: Modular components for reusability
- **Type Safety**: Comprehensive TypeScript types
- **Responsive Design**: Mobile-first approach
- **Accessibility**: WCAG compliance
- **Internationalization**: Multi-language support via i18next
- **Theme Support**: Light/dark mode
- **PWA Features**: 
  - Offline support
  - App-like experience
  - Push notifications
  - Background sync
  - Cache management
- **Performance Optimization**:
  - Code splitting
  - Lazy loading
  - Asset optimization
  - Web vitals monitoring

### Development Tools

- **Storybook**: Component development and documentation
- **ESLint**: Code quality and consistency
- **SWC**: Fast compilation
- **Hot Module Replacement**: Quick development feedback

### Service Worker

The application uses a custom service worker (`sw-template.js`) for:
- Offline functionality
- Cache management
- Background sync
- Push notifications
- Asset caching strategies

Configure service worker behavior in `sw-template.js` and register it using `serviceWorkerRegistration.js`.

### Internationalization

The app uses i18next for internationalization:
- Multiple language support
- Lazy-loaded translations
- Pluralization
- Number and date formatting
- RTL support

Configure languages in `i18n.ts` and use translation keys in components:
```typescript
import { useTranslation } from 'react-i18next';

const MyComponent = () => {
  const { t } = useTranslation();
  return <div>{t('key.to.translate')}</div>;
};
```

## Building and Deployment

### Development

```bash
yarn dev
```

### Production

```bash
yarn build
yarn serve
```

### Docker

Development:
```bash
docker-compose up ui
```

Production:
```bash
docker-compose -f docker-compose-prod.yml up ui
```

## Testing

- **Unit Tests**: Component and utility testing
  ```bash
  yarn test:unit
  ```
- **Integration Tests**: Feature testing
  ```bash
  yarn test:integration
  ```
- **Visual Testing**: Storybook snapshots
  ```bash
  yarn test:visual
  ```
- **Accessibility Testing**: Automated a11y checks
  ```bash
  yarn test:a11y
  ```
- **E2E Tests**: End-to-end testing with Playwright
  ```bash
  yarn test:e2e
  ```

## Contributing

Please refer to the main project's [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## Documentation

- [Architecture Overview](../../ARCHITECTURE.md)
- [API Documentation](../docs/api/README.md)
- [Component Documentation](http://localhost:6006) (Storybook, run locally) 