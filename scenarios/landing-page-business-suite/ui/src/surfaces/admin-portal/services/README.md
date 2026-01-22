# Admin Portal Service Layer

This directory contains business logic services for the admin portal. The architecture follows a layered approach that separates concerns between API calls, data transformations, and UI state management.

## Architecture Pattern

```
┌─────────────────┐
│   Components    │  Pure rendering, event wiring only
└────────┬────────┘
         │
┌────────▼────────┐
│     Hooks       │  Reactive state management
│  (use*.ts)      │  Wraps controllers, provides React state
└────────┬────────┘
         │
┌────────▼────────┐
│   Controllers   │  Page-specific orchestration
│  (*Controller)  │  Bridges services to form state
└────────┬────────┘
         │
┌────────▼────────┐
│    Services     │  Pure functions for API calls,
│  (*.service.ts) │  data transformations, validation
└─────────────────┘
```

## Guidelines

### Services (*.service.ts)

Services contain pure functions that:
- Make API calls (fetching, saving data)
- Transform data between API and UI formats
- Validate input data
- Filter and sort collections
- Build form state from API responses

Services should:
- Be stateless
- Not use React hooks or components
- Be easily testable in isolation
- Have single responsibilities

### Controllers (*Controller.ts)

Controllers (in the `/controllers` directory) orchestrate:
- Multiple service calls
- Page-specific business logic
- Complex data aggregation

### Hooks (use*.ts)

Hooks (in the `/hooks` directory) provide:
- Reactive state management
- Loading/error states
- Side effect management
- Connection between services and React components

### Components

Components should:
- Focus on rendering and event handling
- Delegate business logic to hooks
- Not contain complex data transformations
- Use hooks for all state management

## File Organization

```
services/
├── README.md              # This file
├── pricing.service.ts     # Plan filtering, sorting, demo injection
├── billing.service.ts     # Stripe/billing operations
├── variant.service.ts     # Variant CRUD, snapshots
└── section.service.ts     # Section CRUD, ordering, localStorage

hooks/
├── useBillingForm.ts      # Reactive billing form state
├── useVariantForm.ts      # Reactive variant form state
└── useSectionForm.ts      # Reactive section form state
```

## Example Usage

```typescript
// Service: Pure function
export function filterPricesByInterval(
  prices: PlanOption[],
  interval: 'month' | 'year' | 'other'
): PlanOption[] {
  return prices.filter((price) => {
    const normalized = normalizeInterval(price.billing_interval);
    return normalized === interval;
  });
}

// Hook: Reactive state
export function useBillingForm() {
  const [bundles, setBundles] = useState<BundleCatalogEntry[]>([]);
  const [loading, setLoading] = useState(true);

  const loadBundles = useCallback(async () => {
    setLoading(true);
    const data = await fetchBundleCatalog();
    setBundles(data.bundles);
    setLoading(false);
  }, []);

  return { bundles, loading, loadBundles };
}

// Component: Pure rendering
function BillingSettings() {
  const { bundles, loading, loadBundles } = useBillingForm();

  if (loading) return <LoadingSpinner />;

  return <BundleList bundles={bundles} onRefresh={loadBundles} />;
}
```

## Benefits

1. **Testability**: Services can be tested without React
2. **Reusability**: Logic can be shared across components
3. **Maintainability**: Clear separation of concerns
4. **Readability**: Components stay focused on UI
5. **Performance**: Memoization is easier to apply correctly
