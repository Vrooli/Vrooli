# 🎨 Vrooli UI Package

> **Progressive Web App frontend** - Modern React application with Material-UI, sophisticated state management, and comprehensive PWA capabilities.

> 📖 **Quick Links**: [Component Architecture](#component-architecture) | [State Management](#state-management) | [Development Guide](#development--debugging) | [Testing Strategy](#testing-framework)

---

## 🎯 Purpose & Architecture Role

The `@vrooli/ui` package is the **complete frontend experience** for Vrooli, providing:

- **🎨 Modern React Application** - React 18+ with TypeScript and Material-UI
- **📱 Progressive Web App** - Offline support, push notifications, app-like experience
- **🔄 Real-time Features** - WebSocket integration for live chat and notifications
- **🌍 Internationalization** - Multi-language support with i18next
- **♿ Accessibility** - WCAG compliance and comprehensive a11y features
- **⚡ Performance Optimized** - Code splitting, lazy loading, and caching strategies

This package creates the **user-facing experience** that connects to Vrooli's three-tier AI architecture.

---

## 📦 Package Architecture

### Core Structure

```
ui/src/
├── 🧩 components/         # Reusable UI components
│   ├── buttons/          # Interactive action components
│   ├── dialogs/          # Modal and popup interfaces
│   ├── inputs/           # Form input components
│   ├── lists/            # Data display and collection components
│   └── navigation/       # Routing and menu components
├── 🎣 hooks/             # Custom React hooks and state logic
├── 🏪 stores/            # Zustand state management
├── 🛣️ views/             # Page-level components
├── 🎨 forms/             # Form management and validation
├── 🔧 utils/             # Utility functions and helpers
└── 🌐 api/               # API client and data fetching
```

### Implementation Files

| Component | Location | Purpose |
|-----------|----------|---------|
| **React Hooks** | `src/hooks/` | Custom state management and side effects |
| **Zustand Stores** | `src/stores/` | Global application state |
| **Component Library** | `src/components/` | Reusable UI building blocks |
| **Page Views** | `src/views/` | Complete page implementations |
| **API Integration** | `src/api/` | Server communication and data fetching |
| **Forms System** | `src/forms/` | Form handling with Formik integration |

> 📖 **Related Documentation**: 
> - [UI Performance Guide](../../docs/ui/ui-performance.md)
> - [PWA Configuration](../../docs/ui/pwa-and-twa.md)
> - [Component Testing Strategy](#testing-framework)

---

## 🧩 Component Architecture

### Component Categories

The UI package follows a **hierarchical component architecture** with clear separation of concerns:

```typescript
// Low-level UI components
import { TextInput, LoadableButton } from '@vrooli/ui/components/inputs';

// Mid-level feature components  
import { ChatMessageInput, ObjectActionMenu } from '@vrooli/ui/components';

// High-level page components
import { DashboardView, ChatView } from '@vrooli/ui/views';
```

### Key Component Categories

| Category | Examples | Purpose |
|----------|----------|---------|
| **Input Components** | `TextInput`, `CodeInput`, `AdvancedInput` | Form fields and data entry |
| **Display Components** | `MarkdownDisplay`, `StatsCompact`, `LineGraph` | Content presentation |
| **Navigation Components** | `Navbar`, `BottomNav`, `CommandPalette` | User navigation |
| **Dialog Components** | `AlertDialog`, `ShareObjectDialog`, `FindObjectDialog` | Modal interactions |
| **List Components** | `ObjectList`, `ResourceList`, `SearchList` | Data collections |
| **Button Components** | `LoadableButton`, `ShareButton`, `RunButton` | User actions |

### Advanced Component Features

```typescript
// Managed objects with automatic state synchronization
import { useManagedObject } from '@vrooli/ui/hooks';

const ChatView = ({ chatId }) => {
  const { object: chat, isLoading } = useManagedObject<Chat>({
    objectType: 'Chat',
    objectId: chatId,
  });
  
  // Chat automatically syncs with server state
  return <ChatBubbleTree chat={chat} />;
};

// Socket-based real-time features
import { useSocketChat } from '@vrooli/ui/hooks';

const RealTimeChat = () => {
  const { messages, sendMessage } = useSocketChat(chatId);
  // Real-time message synchronization
};
```

> 📖 **Component Guide**: See [Component Development Guidelines](../../docs/ui/component-development.md)

---

## 🎣 React Hooks System

### Sophisticated State Management Hooks

The UI package provides **production-ready hooks** for complex state scenarios:

```typescript
// Managed object state with server synchronization
import { useManagedObject } from '@vrooli/ui/hooks';

// Auto-save functionality with debouncing
import { useAutoSave } from '@vrooli/ui/hooks';

// Undo/redo with stack management
import { useUndoRedo } from '@vrooli/ui/hooks';

// Socket-based real-time communication
import { useSocketChat, useSocketUser } from '@vrooli/ui/hooks';

// Advanced form state management
import { useFindMany, useUpsertFetch } from '@vrooli/ui/hooks';
```

### Key Hook Categories

| Hook Category | Purpose | Examples |
|---------------|---------|----------|
| **Data Fetching** | Server communication | `useFetch`, `useLazyFetch`, `useFindMany` |
| **State Management** | Component state | `useManagedObject`, `useUndoRedo`, `useAutoSave` |
| **Real-time** | Socket communication | `useSocketChat`, `useSocketUser`, `useSocketConnect` |
| **UI Interaction** | User interface | `usePopover`, `useMenu`, `useHotkeys` |
| **Form Handling** | Form state | `useFormCache`, `useTranslatedFields` |
| **Performance** | Optimization | `useDebounce`, `useThrottle`, `useStableCallback` |

### Custom Hook Examples

```typescript
// Advanced object management with caching
const ProjectView = ({ projectId }) => {
  const { object: project, isLoading, refetch } = useManagedObject({
    objectType: 'Project',
    objectId: projectId,
    transform: (data) => ({
      ...data,
      // Custom transformations
      computedStats: calculateProjectStats(data)
    })
  });
};

// Real-time chat with message management
const ChatInterface = () => {
  const { 
    messages, 
    sendMessage, 
    isConnected,
    participants 
  } = useSocketChat(chatId);
  
  const { messageTree, selectedMessage } = useMessageTree(messages);
  const { canEdit, canDelete } = useMessageActions(selectedMessage);
};
```

> 📖 **Hooks Guide**: See [Custom Hooks Documentation](../../docs/ui/hooks-system.md)

---

## 🏪 State Management with Zustand

### Global Application State

The UI package uses **Zustand** for performant, simple state management:

```typescript
// Chat state management
import { useActiveChatStore } from '@vrooli/ui/stores';

const ChatComponent = () => {
  const { 
    activeChatId,
    chats,
    setActiveChat,
    updateChat 
  } = useActiveChatStore();
  
  // State automatically persists and syncs
};

// Layout and UI state
import { useLayoutStore } from '@vrooli/ui/stores';

const NavigationComponent = () => {
  const { 
    isNavOpen,
    sidebarWidth,
    setNavOpen 
  } = useLayoutStore();
};
```

### Store Categories

| Store | Purpose | Key State |
|-------|---------|-----------|
| **activeChatStore** | Chat management | `activeChatId`, `chats`, `isTyping` |
| **layoutStore** | UI layout | `isNavOpen`, `sidebarWidth`, `theme` |
| **notificationsStore** | Notifications | `unreadCount`, `notifications` |
| **debugStore** | Development tools | `isDebugMode`, `debugInfo` |
| **formCacheStore** | Form persistence | Cached form data across sessions |

### State Management Patterns

```typescript
// Persistent state with localStorage
const chatStore = create(
  persist(
    (set, get) => ({
      activeChatId: null,
      setActiveChat: (id) => set({ activeChatId: id }),
      // State automatically persists to localStorage
    }),
    { name: 'chat-storage' }
  )
);

// Computed state with selectors
const { recentChats } = useActiveChatStore(
  (state) => ({
    recentChats: Object.values(state.chats)
      .filter(chat => chat.lastMessageAt)
      .sort((a, b) => b.lastMessageAt - a.lastMessageAt)
      .slice(0, 10)
  })
);
```

> 📖 **State Management**: See [Zustand Integration Guide](../../docs/ui/state-management.md)

---

## 🔌 API Integration & Data Fetching

### GraphQL-Style API Client

The UI package provides sophisticated API integration with **GraphQL-style selectors**:

```typescript
import { fetchData } from '@vrooli/ui/api';

// Fetch with custom field selection
const userData = await fetchData({
  endpoint: 'user/profile',
  select: {
    id: true,
    name: true,
    emails: {
      id: true,
      email: true,
      verified: true
    },
    // Deeply nested selections
    projects: {
      id: true,
      name: true,
      versions: {
        id: true,
        versionLabel: true
      }
    }
  }
});

// Automatic response parsing and type safety
const { data, errors } = ServerResponseParser.parse(response);
```

### Real-time Communication

```typescript
// WebSocket integration for real-time features
import { useSocketConnect, useSocketChat } from '@vrooli/ui/hooks';

const RealTimeApp = () => {
  // Automatic connection management
  const { isConnected, reconnect } = useSocketConnect();
  
  // Real-time chat with automatic message sync
  const { 
    messages, 
    sendMessage, 
    participants,
    isTyping 
  } = useSocketChat(chatId);
};
```

### API Features

| Feature | Description | Usage |
|---------|-------------|-------|
| **Field Selection** | GraphQL-style data fetching | Reduce payload size, optimize queries |
| **Response Parsing** | Automatic error handling | Consistent error states across app |
| **Caching** | Smart request caching | Reduce server load, faster UI |
| **Real-time** | WebSocket communication | Live chat, notifications, status updates |
| **Offline Support** | PWA caching strategies | Work without internet connection |

> 📖 **API Guide**: See [API Integration Patterns](../../docs/ui/api-integration.md)

---

## 📱 Progressive Web App Features

### PWA Capabilities

The UI package provides **complete PWA functionality**:

```typescript
// Service worker registration with caching strategies
// sw-template.js provides:
// - Offline functionality
// - Background sync
// - Push notifications
// - Asset caching
// - Update management

// Push notification integration
import { subscribeToPush, sendNotification } from '@vrooli/ui/utils/push';

// Offline detection and handling
import { useOnlineStatus } from '@vrooli/ui/hooks';

const OfflineIndicator = () => {
  const isOnline = useOnlineStatus();
  return !isOnline ? <OfflineBanner /> : null;
};
```

### PWA Features

| Feature | Implementation | Benefits |
|---------|---------------|----------|
| **Offline Support** | Service worker caching | App works without internet |
| **Push Notifications** | Web Push API | Real-time user engagement |
| **App Installation** | Manifest file | Native app-like experience |
| **Background Sync** | Service worker sync | Data sync when online |
| **Asset Caching** | Workbox strategies | Fast loading, reduced bandwidth |

### Performance Optimizations

```typescript
// Code splitting with lazy loading
const DashboardView = lazy(() => import('./views/main/DashboardView'));
const ChatView = lazy(() => import('./views/objects/chat/ChatView'));

// Optimized rendering with React.memo
const MemoizedComponent = memo(ExpensiveComponent);

// Virtual scrolling for large lists
import { FixedSizeList } from 'react-window';

const LargeList = ({ items }) => (
  <FixedSizeList
    height={400}
    itemCount={items.length}
    itemSize={50}
  >
    {({ index, style }) => (
      <div style={style}>{items[index]}</div>
    )}
  </FixedSizeList>
);
```

> 📖 **PWA Guide**: See [PWA Configuration & Features](../../docs/ui/pwa-and-twa.md)

---

## 🌍 Internationalization System

### Multi-Language Support

```typescript
import { useTranslation } from 'react-i18next';

const LocalizedComponent = () => {
  const { t, i18n } = useTranslation();
  
  return (
    <div>
      <h1>{t('common:welcome')}</h1>
      <p>{t('dashboard:description', { name: userName })}</p>
      
      {/* Dynamic language switching */}
      <button onClick={() => i18n.changeLanguage('es')}>
        Español
      </button>
    </div>
  );
};

// Translated form validation
import { useTranslatedFields } from '@vrooli/ui/hooks';

const TranslatedForm = () => {
  const { translatedFields } = useTranslatedFields({
    fieldName: 'description',
    languages: ['en', 'es', 'fr'],
    required: ['en']
  });
};
```

### i18n Features

| Feature | Implementation | Purpose |
|---------|---------------|---------|
| **Dynamic Loading** | Lazy-loaded translations | Reduce initial bundle size |
| **Pluralization** | ICU format support | Handle complex grammar rules |
| **Interpolation** | Variable substitution | Dynamic content in translations |
| **Namespace Organization** | Modular translation files | Maintainable translation structure |
| **RTL Support** | Bidirectional text | Support Arabic, Hebrew, etc. |

---

## 🚀 Development & Debugging

### Development Setup

```bash
# Install dependencies
pnpm install

# Start development server
pnpm dev

# Run with specific port
VITE_PORT=3000 pnpm dev

# Build for production
pnpm build

# Preview production build
pnpm preview
```

### Development Tools

```typescript
// Debug store for development insights
import { useDebugStore } from '@vrooli/ui/stores';

const DebugPanel = () => {
  const { 
    isDebugMode, 
    debugInfo,
    enableDebug 
  } = useDebugStore();
  
  // Development-only debug information
  if (process.env.NODE_ENV !== 'development') return null;
  
  return <DebugComponent debugInfo={debugInfo} />;
};

// Performance monitoring
import { reportWebVitals } from '@vrooli/ui';

reportWebVitals((metric) => {
  // Track Core Web Vitals
  console.log(metric);
});
```

### Hot Module Replacement

The development server supports **fast refresh** for:
- React components with state preservation
- CSS and styling changes
- Hook updates with state retention
- TypeScript compilation

### Environment Configuration

```typescript
// Environment-specific configuration
const config = {
  development: {
    apiUrl: 'http://localhost:5555',
    debugMode: true,
    logLevel: 'debug'
  },
  production: {
    apiUrl: 'https://api.vrooli.com',
    debugMode: false,
    logLevel: 'error'
  }
};
```

> 📖 **Development Guide**: See [UI Development Workflows](../../docs/devops/development-workflows.md#ui-development)

---

## 🧪 Testing Framework

### Comprehensive Testing Strategy

The UI package uses **Vitest** with comprehensive testing utilities:

```bash
# Run all tests
pnpm test

# Watch mode for development
pnpm test-watch

# Coverage report
pnpm test-coverage

# Visual regression testing with Storybook
pnpm test:visual
```

### Testing Utilities

```typescript
// Component testing with React Testing Library
import { render, screen, fireEvent } from '@testing-library/react';
import { TestWrapper } from '@vrooli/ui/__test/testUtils';

test('ChatMessageInput sends message on submit', async () => {
  const mockSendMessage = vi.fn();
  
  render(
    <TestWrapper>
      <ChatMessageInput onSendMessage={mockSendMessage} />
    </TestWrapper>
  );
  
  const input = screen.getByRole('textbox');
  fireEvent.change(input, { target: { value: 'Test message' } });
  fireEvent.submit(input);
  
  expect(mockSendMessage).toHaveBeenCalledWith('Test message');
});

// Hook testing
import { renderHook, act } from '@testing-library/react';
import { useManagedObject } from '@vrooli/ui/hooks';

test('useManagedObject fetches and caches data', async () => {
  const { result } = renderHook(() => 
    useManagedObject({ objectType: 'Project', objectId: '123' })
  );
  
  expect(result.current.isLoading).toBe(true);
  
  await waitFor(() => {
    expect(result.current.object).toBeDefined();
    expect(result.current.isLoading).toBe(false);
  });
});
```

### Testing Categories

| Test Type | Tools | Purpose |
|-----------|-------|---------|
| **Unit Tests** | Vitest + Testing Library | Component and hook behavior |
| **Integration Tests** | MSW + Testing Library | API integration and workflows |
| **Visual Tests** | Storybook + Chromatic | UI appearance and consistency |
| **Accessibility Tests** | axe-core + Testing Library | WCAG compliance |
| **E2E Tests** | Playwright | Complete user workflows |

### Mock Service Worker (MSW)

```typescript
// API mocking for tests and Storybook
import { rest } from 'msw';

export const handlers = [
  rest.get('/api/chat/:id', (req, res, ctx) => {
    return res(
      ctx.json({
        id: req.params.id,
        messages: mockMessages,
        participants: mockParticipants
      })
    );
  }),
];

// Storybook integration
export const parameters = {
  msw: {
    handlers: [...handlers],
  },
};
```

> 📖 **Testing Guide**: See [UI Testing Strategy](../../docs/testing/writing-tests.md#ui-testing)

---

## 🎨 Styling & Theming

### Material-UI Integration

```typescript
// Custom theme with Vrooli branding
import { createTheme, ThemeProvider } from '@mui/material/styles';

const vroolicTheme = createTheme({
  palette: {
    primary: {
      main: '#1976d2',
    },
    secondary: {
      main: '#dc004e',
    },
    mode: 'light', // or 'dark'
  },
  typography: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
  },
});

// Theme switching
import { useLayoutStore } from '@vrooli/ui/stores';

const ThemeToggle = () => {
  const { isDarkMode, toggleTheme } = useLayoutStore();
  
  return (
    <IconButton onClick={toggleTheme}>
      {isDarkMode ? <LightModeIcon /> : <DarkModeIcon />}
    </IconButton>
  );
};
```

### CSS-in-JS with Emotion

```typescript
import { styled } from '@mui/material/styles';

const StyledChatBubble = styled('div')<{ isOwn: boolean }>(
  ({ theme, isOwn }) => ({
    padding: theme.spacing(1, 2),
    borderRadius: theme.spacing(1),
    backgroundColor: isOwn 
      ? theme.palette.primary.main 
      : theme.palette.grey[200],
    color: isOwn 
      ? theme.palette.primary.contrastText 
      : theme.palette.text.primary,
    maxWidth: '70%',
    alignSelf: isOwn ? 'flex-end' : 'flex-start',
  })
);
```

---

## 📚 Component Documentation with Storybook

### Interactive Component Development

```bash
# Start Storybook
pnpm storybook

# Build static Storybook
pnpm build-storybook
```

### Story Examples

```typescript
// Component stories for documentation
import type { Meta, StoryObj } from '@storybook/react';
import { ChatMessageInput } from './ChatMessageInput';

const meta: Meta<typeof ChatMessageInput> = {
  title: 'Components/Chat/ChatMessageInput',
  component: ChatMessageInput,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    placeholder: 'Type a message...',
    disabled: false,
  },
};

export const WithMentions: Story = {
  args: {
    ...Default.args,
    enableMentions: true,
    participants: mockParticipants,
  },
};
```

---

## 🔗 Package Integration

### Server Integration

```typescript
// API communication with shared types
import { type ChatShape, type UserShape } from '@vrooli/shared';
import { fetchData } from '@vrooli/ui/api';

// Type-safe API calls
const chat: ChatShape = await fetchData({
  endpoint: 'chat/findByid',
  input: { id: chatId },
  select: chatSelect,
});
```

### Background Jobs Integration

```typescript
// Push notification handling
import { subscribeToPush } from '@vrooli/ui/utils/push';

// Subscribe to background job notifications
const subscription = await subscribeToPush({
  applicationServerKey: process.env.VITE_VAPID_PUBLIC_KEY,
});
```

---

## 📚 Related Documentation

### Architecture References
- **[Three-Tier Architecture](../../docs/architecture/execution/README.md)** - Complete architectural overview  
- **[UI Performance Guide](../../docs/ui/ui-performance.md)** - Performance optimization strategies
- **[PWA Documentation](../../docs/ui/pwa-and-twa.md)** - Progressive Web App features

### Development Guides
- **[Component Development](../../docs/ui/component-development.md)** - Building reusable components
- **[State Management](../../docs/ui/state-management.md)** - Zustand integration patterns
- **[API Integration](../../docs/ui/api-integration.md)** - Server communication patterns

### Design & UX
- **[UI Design System](../../docs/ui/design-system.md)** - Component guidelines and standards
- **[Accessibility Guide](../../docs/ui/accessibility.md)** - WCAG compliance and a11y features
- **[Mobile Experience](../../docs/ui/mobile-responsive.md)** - Responsive design patterns

---

## 🎯 Key Features Summary

### **🎨 Modern React Architecture**
- React 18+ with TypeScript and Material-UI
- Sophisticated state management with Zustand
- Advanced hooks system for complex state scenarios
- Component-driven architecture with Storybook documentation

### **📱 Progressive Web App**
- Complete offline functionality with service worker
- Push notifications and background sync
- App-like experience with manifest and installation
- Performance optimization with caching strategies

### **🌐 Real-time & International**
- WebSocket integration for live features
- Comprehensive i18n with 10+ languages
- Real-time chat and notification systems
- RTL support and cultural localization

### **🔬 Developer Experience**
- Hot module replacement with state preservation
- Comprehensive testing with Vitest and Testing Library
- Visual regression testing with Storybook
- Advanced debugging tools and performance monitoring

### **♿ Accessibility & Performance**
- WCAG 2.1 AA compliance
- Virtual scrolling for large datasets
- Code splitting and lazy loading
- Core Web Vitals optimization

---

*This package delivers the complete user experience for Vrooli's collaborative AI platform, combining modern web technologies with sophisticated user interface patterns.*