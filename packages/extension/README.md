# Vrooli Browser Extension

Browser extension for Vrooli, enabling seamless integration with web browsers.

## Overview

This package contains the browser extension for Vrooli, providing enhanced functionality and integration with web browsers. It allows users to interact with Vrooli features directly from their browser interface.

## Technology Stack

- **Framework**: React
- **Language**: TypeScript
- **Build Tool**: Vite
- **Testing**: Chai/Mocha/Sinon
- **Browser APIs**: WebExtensions API
- **State Management**: React Context

## Directory Structure

```
extension/
├── src/
│   ├── background/   # Background scripts
│   ├── index.tsx     # Main code
├── public/           # Static assets
```

## Features

- Browser toolbar integration
- Context menu integration
- Content script injection
- Cross-browser compatibility
- Secure communication with Vrooli API
- Local storage management

## Getting Started

1. Install dependencies:
   ```bash
   pnpm install
   ```

2. Start development:
   ```bash
   pnpm dev
   ```

3. Build extension:
   ```bash
   pnpm build
   ```

4. Load extension in browser:
   - Chrome: Open chrome://extensions/
   - Enable Developer mode
   - Load unpacked extension from `dist/` directory

## Development

### Component Development

- Use React functional components
- Follow TypeScript best practices
- Implement proper error handling
- Use browser storage APIs appropriately
- Handle permissions correctly

### Browser APIs

- Tabs API
- Storage API
- Runtime messaging
- Context menus
- Browser action
- Content scripts

## Building and Testing

### Development Build

```bash
pnpm dev
```

### Production Build

```bash
pnpm build
```

### Testing

```bash
pnpm test
```

## Browser Support

- Chrome/Chromium
- Firefox
- Edge
- Safari (planned)

## Security

- Content Security Policy (CSP)
- Cross-origin security
- Secure storage
- Permission management
- API authentication

## Contributing

Please refer to the main project's [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

### Development Guidelines

1. Follow browser extension best practices
2. Test across supported browsers
3. Handle errors gracefully
4. Maintain backward compatibility
5. Document API usage

## Documentation

- [Architecture Overview](../../ARCHITECTURE.md)
- [API Documentation](../docs/api/README.md)
- [Browser Extension Guide](./docs/EXTENSION.md)

## Best Practices

- Minimize extension permissions
- Optimize performance
- Handle browser differences
- Follow security guidelines
- Implement proper error handling
- Use appropriate storage methods 