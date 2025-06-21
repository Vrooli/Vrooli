# UI State Fixtures

This directory contains fixtures for testing various UI states and state transitions. These fixtures help test how components handle different states like loading, error, success, and empty states.

## Purpose

UI state fixtures provide:
- **Consistent state representations** across all components
- **State transition testing** for complex UI flows
- **Error state validation** with realistic error scenarios
- **Loading state management** with progress indicators
- **Empty state handling** with appropriate user guidance

## Architecture

UI state fixtures follow this pattern:

```typescript
export interface UIStateFactory<TState> {
    createLoadingState(context?: LoadingContext): TState;
    createErrorState(error: AppError): TState;
    createSuccessState(data: any, message?: string): TState;
    createEmptyState(): TState;
    
    // State transitions
    transitionToLoading(currentState: TState): TState;
    transitionToSuccess(currentState: TState, data: any): TState;
    transitionToError(currentState: TState, error: AppError): TState;
}
```

## State Types

### 1. Loading States
Different types of loading states with context:
- **Initial Loading**: First time data load
- **Refreshing**: Updating existing data
- **Submitting**: Form submission in progress
- **Background Loading**: Loading additional data

### 2. Error States
Comprehensive error scenarios:
- **Validation Errors**: Form field validation failures
- **Network Errors**: Connection and API failures  
- **Permission Errors**: Access denied scenarios
- **Server Errors**: Backend failures and timeouts

### 3. Success States
Various success scenarios:
- **Creation Success**: After successful data creation
- **Update Success**: After successful data updates
- **Deletion Success**: After successful data removal
- **Action Success**: After successful user actions

### 4. Empty States
Different empty state scenarios:
- **No Data**: When no items exist
- **Filtered Empty**: When filters return no results
- **Search Empty**: When search returns no results
- **Permission Empty**: When user lacks access to data

## Bookmark State Examples

### Loading States
```typescript
export interface BookmarkLoadingState extends BookmarkUIState {
    isLoading: true;
    loadingContext?: {
        operation: 'creating' | 'updating' | 'deleting' | 'fetching';
        progress?: number;
        message?: string;
    };
}
```

### Error States
```typescript
export interface BookmarkErrorState extends BookmarkUIState {
    isLoading: false;
    error: {
        code: string;
        message: string;
        recoverable: boolean;
        retryAction?: () => void;
        details?: any;
    };
}
```

### Success States
```typescript
export interface BookmarkSuccessState extends BookmarkUIState {
    isLoading: false;
    bookmark: Bookmark;
    successMessage?: string;
    actionCompleted?: 'created' | 'updated' | 'deleted';
}
```

## State Transitions

### State Machine Pattern
```typescript
export class BookmarkStateMachine {
    private currentState: BookmarkUIState;
    
    constructor(initialState: BookmarkUIState) {
        this.currentState = initialState;
    }
    
    transition(action: StateAction, payload?: any): BookmarkUIState {
        switch (action) {
            case 'START_LOADING':
                return this.transitionToLoading(payload);
                
            case 'LOAD_SUCCESS':
                return this.transitionToSuccess(payload);
                
            case 'LOAD_ERROR':
                return this.transitionToError(payload);
                
            case 'RESET':
                return this.transitionToEmpty();
                
            default:
                return this.currentState;
        }
    }
}
```

### Transition Validation
```typescript
export class StateTransitionValidator {
    validateTransition(from: BookmarkUIState, to: BookmarkUIState): {
        valid: boolean;
        reason?: string;
    } {
        // Validate that state transitions make sense
        if (from.isLoading && to.isLoading) {
            return { valid: false, reason: 'Cannot transition from loading to loading' };
        }
        
        if (from.error && to.bookmark && !to.error) {
            // Valid: Error state to success state after retry
            return { valid: true };
        }
        
        return { valid: true };
    }
}
```

## Context-Aware States

### Loading Context
```typescript
export interface LoadingContext {
    operation: string;
    progress?: number;
    message?: string;
    estimatedDuration?: number;
    cancellable?: boolean;
    onCancel?: () => void;
}
```

### Error Context  
```typescript
export interface ErrorContext {
    code: string;
    message: string;
    recoverable: boolean;
    retryCount?: number;
    maxRetries?: number;
    retryAction?: () => Promise<void>;
    fallbackAction?: () => void;
    userReported?: boolean;
}
```

### Success Context
```typescript
export interface SuccessContext {
    message?: string;
    action: string;
    data?: any;
    nextActions?: Array<{
        label: string;
        action: () => void;
    }>;
    autoHide?: {
        delay: number;
        onHide: () => void;
    };
}
```

## Usage Examples

### Component State Testing
```typescript
describe('BookmarkButton Component', () => {
    it('should display loading state correctly', () => {
        const loadingState = bookmarkStates.createLoadingState({
            operation: 'creating',
            message: 'Creating bookmark...'
        });
        
        render(<BookmarkButton state={loadingState} />);
        
        expect(screen.getByText('Creating bookmark...')).toBeInTheDocument();
        expect(screen.getByRole('button')).toBeDisabled();
    });
    
    it('should display error state with retry option', () => {
        const errorState = bookmarkStates.createErrorState({
            code: 'NETWORK_ERROR',
            message: 'Unable to create bookmark',
            recoverable: true,
            retryAction: jest.fn()
        });
        
        render(<BookmarkButton state={errorState} />);
        
        expect(screen.getByText('Unable to create bookmark')).toBeInTheDocument();
        expect(screen.getByText('Retry')).toBeInTheDocument();
    });
});
```

### State Transition Testing
```typescript
describe('Bookmark State Transitions', () => {
    it('should transition through states correctly', async () => {
        const { result } = renderHook(() => useBookmarkState());
        
        // Start with empty state
        expect(result.current.state.isLoading).toBe(false);
        expect(result.current.state.bookmark).toBeNull();
        
        // Transition to loading
        act(() => {
            result.current.startBookmarking();
        });
        expect(result.current.state.isLoading).toBe(true);
        
        // Transition to success
        const mockBookmark = bookmarkFixtures.createMockResponse();
        act(() => {
            result.current.bookmarkCreated(mockBookmark);
        });
        expect(result.current.state.isLoading).toBe(false);
        expect(result.current.state.bookmark).toBe(mockBookmark);
    });
});
```

### Form State Integration
```typescript
describe('Bookmark Form States', () => {
    it('should handle form submission states', async () => {
        const formState = bookmarkStates.createFormState({
            isSubmitting: false,
            errors: {},
            touched: {},
            values: bookmarkFixtures.createFormData('minimal')
        });
        
        render(<BookmarkForm initialState={formState} />);
        
        // Submit form
        await user.click(screen.getByRole('button', { name: 'Save' }));
        
        // Should show submitting state
        expect(screen.getByText('Saving...')).toBeInTheDocument();
    });
});
```

## State Persistence

### Local Storage Integration
```typescript
export class BookmarkStateManager {
    private storageKey = 'bookmark-ui-state';
    
    saveState(state: BookmarkUIState): void {
        // Only save non-transient state
        const persistableState = {
            availableLists: state.availableLists,
            showListSelection: state.showListSelection,
            // Don't persist loading/error states
        };
        
        localStorage.setItem(this.storageKey, JSON.stringify(persistableState));
    }
    
    loadState(): Partial<BookmarkUIState> | null {
        const saved = localStorage.getItem(this.storageKey);
        return saved ? JSON.parse(saved) : null;
    }
}
```

### State Hydration
```typescript
export class StateHydrator {
    hydrateBookmarkState(
        savedState: Partial<BookmarkUIState>,
        currentBookmark?: Bookmark
    ): BookmarkUIState {
        return {
            isLoading: false,
            bookmark: currentBookmark || null,
            error: null,
            isBookmarked: !!currentBookmark,
            availableLists: savedState.availableLists || [],
            showListSelection: savedState.showListSelection || false
        };
    }
}
```

## Performance Optimization

### State Memoization
```typescript
export class MemoizedStateFactory {
    private stateCache = new Map<string, BookmarkUIState>();
    
    createState(key: string, factory: () => BookmarkUIState): BookmarkUIState {
        if (!this.stateCache.has(key)) {
            this.stateCache.set(key, factory());
        }
        return this.stateCache.get(key)!;
    }
    
    invalidateState(key: string): void {
        this.stateCache.delete(key);
    }
}
```

### Shallow State Updates
```typescript
export class ShallowStateUpdater {
    updateState<T extends Record<string, any>>(
        currentState: T,
        updates: Partial<T>
    ): T {
        // Only create new object if something actually changed
        const hasChanges = Object.keys(updates).some(
            key => currentState[key] !== updates[key]
        );
        
        return hasChanges ? { ...currentState, ...updates } : currentState;
    }
}
```

## Error Recovery Patterns

### Retry Logic
```typescript
export class RetryStateManager {
    createRetryableErrorState(
        error: AppError,
        retryCount: number = 0,
        maxRetries: number = 3
    ): BookmarkErrorState {
        return {
            isLoading: false,
            bookmark: null,
            error: {
                ...error,
                recoverable: retryCount < maxRetries,
                retryCount,
                maxRetries,
                retryAction: retryCount < maxRetries ? this.createRetryAction() : undefined
            },
            isBookmarked: false,
            availableLists: [],
            showListSelection: false
        };
    }
    
    private createRetryAction(): () => Promise<void> {
        return async () => {
            // Implement retry logic
            console.log('Retrying bookmark operation...');
        };
    }
}
```

### Fallback States
```typescript
export class FallbackStateProvider {
    createFallbackState(originalError: AppError): BookmarkUIState {
        return {
            isLoading: false,
            bookmark: null,
            error: null,
            isBookmarked: false,
            availableLists: [],
            showListSelection: false,
            // Add fallback UI indicators
            fallbackActive: true,
            fallbackMessage: 'Using offline mode'
        };
    }
}
```

## Best Practices

### DO's ✅
- Create states that match real user scenarios
- Include context information for better UX
- Test state transitions thoroughly
- Use consistent state shapes across components
- Implement proper error recovery
- Cache states appropriately for performance
- Validate state transitions

### DON'Ts ❌
- Create states that can't occur in real usage
- Skip error state testing
- Mutate state objects directly
- Store sensitive data in UI state
- Create overly complex state hierarchies
- Ignore loading state UX
- Forget to handle edge cases

## Troubleshooting

### Common Issues
1. **State Inconsistency**: Use state validators to catch inconsistent states
2. **Memory Leaks**: Clear state subscriptions and caches appropriately
3. **Race Conditions**: Implement proper state sequence validation
4. **Performance Issues**: Use memoization and shallow updates
5. **Type Safety**: Ensure state types match component expectations

### Debug Strategies
- Log state transitions in development
- Use Redux DevTools for complex state debugging  
- Implement state validation in tests
- Monitor state update frequency
- Test edge case state combinations

This comprehensive state management approach ensures reliable and user-friendly UI behavior across all components.