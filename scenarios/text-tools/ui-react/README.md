# Text Tools React UI

This is a modern React/Vite/TypeScript rewrite of the Text Tools UI, maintaining the same functionality and design while using modern web development tools.

## Features

- **Dark Theme**: VS Code-inspired dark theme for developer-friendly interface
- **Text Comparison**: Side-by-side diff viewer with multiple comparison modes
- **File Upload**: Drag and drop or click to upload text files
- **Responsive Design**: Works on desktop and mobile devices
- **TypeScript**: Full type safety throughout the application
- **Modern Tooling**: Vite for fast development and builds

## Technology Stack

- **React 18**: Modern React with hooks
- **TypeScript**: Type-safe development
- **Vite**: Fast build tool and dev server
- **CSS3**: Custom CSS with CSS variables for theming
- **ESLint**: Code linting and formatting

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Lint code
npm run lint
```

## Architecture

- **App.tsx**: Main application component with navigation and layout
- **components/DiffPanel.tsx**: Text comparison functionality
- **styles/**: Global styles and component-specific CSS
- **utils/**: Utility functions (future expansion)
- **hooks/**: Custom React hooks (future expansion)

## API Integration

The UI connects to the text-tools API running on port specified by the backend:
- Health checks: `/api/health`
- Text comparison: `/api/v1/text/diff`
- API documentation: `/api/docs`

## Building for Production

The production build is automatically generated when the parent UI package runs `npm run build`, which builds the React app and serves the static files through the Express server.

## Styling

The application uses a "BORING" but functional design philosophy:
- Technical, developer-focused appearance
- Dense information layout
- Monospace fonts for code/text areas
- Subtle animations only for state transitions
- Dark color scheme matching VS Code

## Future Enhancements

Placeholder components are ready for:
- Search functionality
- Text transformation tools
- Content extraction
- Text analysis features
- Pipeline builder