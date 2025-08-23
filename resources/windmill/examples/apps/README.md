# Windmill App Examples

Apps are low-code UI applications built with Windmill's drag-and-drop interface. They provide interactive user interfaces that can execute scripts and flows, making automation accessible to non-technical users.

## Available Examples

### 1. User Management Dashboard (`admin-dashboard.json`)

A comprehensive admin panel for user management with real-time metrics and bulk operations.

**Key Features:**
- **Metrics Cards**: Total users, active today, new this week, suspended
- **Searchable Table**: Filter and sort users
- **CRUD Operations**: Create, read, update, delete users
- **Bulk Actions**: Select multiple users for operations
- **Activity Feed**: Real-time admin activity log
- **Export Functionality**: Export user data to CSV

**Components Used:**
- Metric cards with live data
- Searchable data table with pagination
- Modal dialogs for forms
- Activity list with real-time updates
- Action buttons with confirmations

**Use Cases:**
- SaaS admin panels
- Enterprise user management
- Customer service tools
- HR systems

### 2. Customer Onboarding Form (`data-entry-form.json`)

Multi-step form application with validation, file uploads, and progress tracking.

**Key Features:**
- **5-Step Process**: Company → Contact → Business → Documents → Review
- **Progress Indicator**: Visual stepper showing completion
- **Field Validation**: Real-time validation with error messages
- **File Uploads**: Secure document upload with type restrictions
- **Auto-save**: Prevents data loss with periodic saving
- **Conditional Fields**: Dynamic form based on selections

**Form Steps:**
1. **Company Information**: Legal name, registration, industry
2. **Contact Details**: Primary contact, addresses
3. **Business Details**: Revenue, employees, services needed
4. **Documents**: Business license, tax documents
5. **Review & Submit**: Summary with edit capabilities

**Use Cases:**
- Customer onboarding
- Vendor registration
- Employee onboarding
- Loan applications
- Partner applications

### 3. System Monitoring Dashboard (`monitoring-dashboard.json`)

Real-time monitoring dashboard for system health and performance metrics.

**Key Features:**
- **Live Metrics**: CPU, Memory, Disk, Network usage
- **Interactive Charts**: Line graphs, bar charts with real-time updates
- **Service Status**: List of services with health indicators
- **Log Viewer**: Real-time log streaming with filters
- **Alert Management**: Active alerts with acknowledgment
- **WebSocket Support**: Live data updates without refresh

**Dashboard Sections:**
- **Header**: Overall health status and last update
- **Alerts Banner**: Active alert count with quick access
- **Metrics Grid**: Key performance indicators
- **Charts**: Historical performance data
- **Services**: Service status with restart capabilities
- **Logs**: Filtered log viewer with syntax highlighting

**Use Cases:**
- Infrastructure monitoring
- Application performance monitoring
- Security operations center
- DevOps dashboards
- IoT device monitoring

## App Architecture

### Component Hierarchy

```
App
├── Layout Container
│   ├── Header
│   ├── Navigation
│   ├── Main Content
│   │   ├── Data Components
│   │   ├── Forms
│   │   └── Charts
│   └── Footer
├── Modals
│   ├── Forms
│   └── Confirmations
└── State Management
```

### Data Flow

1. **Initial Load**: Scripts run on app load
2. **User Interaction**: Components trigger scripts
3. **State Updates**: Results update app state
4. **UI Refresh**: Components re-render with new data
5. **Real-time Updates**: WebSockets or polling for live data

## Building Apps in Windmill

### Visual App Builder

1. **Create New App**: Navigate to Apps → New App
2. **Choose Template**: Start from scratch or template
3. **Drag Components**: Add from component library
4. **Configure Properties**: Set component properties
5. **Bind Data**: Connect to scripts and state
6. **Add Interactions**: Define click handlers, etc.
7. **Preview**: Test in preview mode
8. **Deploy**: Save and share URL

### Available Components

#### Display Components
- **Text**: Labels, headings, paragraphs
- **Metric Cards**: KPI displays with trends
- **Tables**: Data grids with sorting/filtering
- **Charts**: Line, bar, pie, scatter plots
- **Lists**: Formatted lists with templates

#### Input Components
- **Forms**: Text, number, select, checkbox
- **File Upload**: Drag-and-drop file uploads
- **Search**: Search bars with autocomplete
- **Date Picker**: Calendar date selection
- **Sliders**: Range input controls

#### Layout Components
- **Container**: Flexible layout containers
- **Grid**: Responsive grid layouts
- **Tabs**: Tabbed interfaces
- **Accordion**: Collapsible sections
- **Modal**: Dialog windows

#### Action Components
- **Button**: Trigger scripts or navigation
- **Link**: Internal or external navigation
- **Menu**: Dropdown action menus

### Data Binding

Windmill uses a reactive binding system:

```javascript
// Bind to script output
${scripts.get_users.output}

// Bind to component state
${components.search_bar.value}

// Bind to app state
${state.user_filter}

// Conditional rendering
${state.isLoading ? 'Loading...' : 'Ready'}

// Array mapping
${data.users.map(u => u.name).join(', ')}
```

### Event Handling

Configure component interactions:

```javascript
// Run script on click
onClick: {
  runScript: "f/admin/delete_user",
  inputs: {
    user_id: "${row.id}"
  }
}

// Update state
onChange: {
  setState: {
    filter: "${value}"
  }
}

// Open modal
onClick: {
  openModal: "edit_user_modal",
  data: "${row}"
}

// Multiple actions
onClick: [
  { validateForm: "user_form" },
  { runScript: "f/admin/save_user" },
  { closeModal: true }
]
```

## Best Practices

### UI/UX Design

1. **Consistent Layout**: Use standard layouts
2. **Clear Navigation**: Obvious user flow
3. **Responsive Design**: Works on all devices
4. **Loading States**: Show progress indicators
5. **Error Handling**: Clear error messages

### Performance

1. **Lazy Loading**: Load data as needed
2. **Pagination**: Handle large datasets
3. **Debouncing**: Limit API calls on input
4. **Caching**: Cache static data
5. **Virtual Scrolling**: For long lists

### State Management

1. **Minimal State**: Only store necessary data
2. **State Structure**: Organize logically
3. **Derived State**: Calculate from base state
4. **State Persistence**: Save user preferences
5. **State Reset**: Clear state appropriately

### Security

1. **Input Validation**: Validate all user input
2. **Authentication**: Require login for sensitive apps
3. **Authorization**: Check permissions
4. **XSS Prevention**: Sanitize displayed data
5. **Secure Storage**: Don't store secrets in UI

## Advanced Patterns

### 1. Master-Detail View
Split screen with list and details:
```
List Component → Selection → Detail View
                     ↓
                Update State → Refresh Details
```

### 2. Wizard Pattern
Multi-step process with validation:
```
Step 1 → Validate → Step 2 → Validate → ... → Submit
   ↑                    ↑                         ↓
   ←────── Previous ────┴──────────────────── Next
```

### 3. Dashboard Grid
Responsive metric dashboard:
```
┌─────────┬─────────┬─────────┬─────────┐
│ Metric 1│ Metric 2│ Metric 3│ Metric 4│
├─────────┴─────────┴─────────┴─────────┤
│           Main Chart Area              │
├─────────────────┬─────────────────────┤
│   Data Table    │    Side Panel       │
└─────────────────┴─────────────────────┘
```

### 4. Real-time Updates
Live data with WebSockets:
```
WebSocket Connection
        ↓
Message Handler → Parse Data → Update State
                                     ↓
                              Re-render UI
```

## Styling and Theming

### CSS Customization
```css
/* Component styles */
{
  "style": {
    "backgroundColor": "#f5f5f5",
    "padding": "16px",
    "borderRadius": "8px",
    "boxShadow": "0 2px 4px rgba(0,0,0,0.1)"
  }
}
```

### Theme Variables
- Primary/Secondary colors
- Font families and sizes
- Spacing units
- Border radius
- Shadow definitions

### Dark Mode Support
Apps can support dark mode with:
- Theme-aware color variables
- Conditional styling
- User preference detection

## Testing Apps

### Manual Testing
1. **Preview Mode**: Test without saving
2. **Device Testing**: Check responsive design
3. **User Flow**: Test complete workflows
4. **Edge Cases**: Test error conditions
5. **Performance**: Check with large datasets

### Automated Testing
- Use Windmill's API to test app endpoints
- Selenium/Playwright for UI testing
- Load testing for performance

## Deployment

### Access Control
1. **Public Apps**: No authentication required
2. **Private Apps**: Require Windmill login
3. **Role-based**: Restrict by user role
4. **Custom Auth**: Integrate with SSO

### Embedding
Apps can be embedded in other sites:
```html
<iframe src="https://windmill.dev/apps/u/username/app-name" 
        width="100%" 
        height="600">
</iframe>
```

### Versioning
- Apps are versioned automatically
- Rollback to previous versions
- A/B testing with versions

## Next Steps

1. **Import Examples**: Load example apps into Windmill
2. **Customize**: Modify for your needs
3. **Build New Apps**: Create your own interfaces
4. **Connect Scripts**: Link to your automation
5. **Share**: Deploy and share with users

For the underlying automation, see [Scripts](../scripts/) and [Flows](../flows/).