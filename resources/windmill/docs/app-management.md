# Windmill App Management

This document describes the app management functionality added to the Windmill resource scripts.

## Current Status

Windmill provides a programmatic API for creating apps with the following endpoints:
- `POST /api/w/{workspace}/apps/create` - Create a new app
- `POST /api/w/{workspace}/apps/create_raw` - Create app with raw JS/CSS
- `GET /api/w/{workspace}/apps/list` - List all apps
- `GET /api/w/{workspace}/apps/get/p/{path}` - Get app by path
- `GET /api/w/{workspace}/apps/exists/{path}` - Check if app exists

**Note**: Some Windmill versions have a workspace foreign key constraint issue that may prevent app creation via API. In such cases, manual import through the UI is still required.

## Implemented Features

### 1. List Available Apps

```bash
./manage.sh --action list-apps
```

Lists all example apps available in the `examples/apps/` directory with their descriptions.

### 2. Prepare App for Import

```bash
./manage.sh --action prepare-app --app-name admin-dashboard
```

This command:
- Copies the app JSON definition to a working directory
- Extracts required scripts that need to be created first
- Generates detailed import instructions
- Creates files in `~/windmill-apps/` by default

### 3. Deploy App via API

```bash
./manage.sh --action deploy-app --app-name admin-dashboard --workspace demo
```

This command:
- Converts the app definition to Windmill's expected format
- Deploys the app via the API
- Reports success or provides guidance if constraints prevent deployment

### 4. Check App API Status

```bash
./manage.sh --action check-app-api
```

Verifies that app management API endpoints are available and working.

## App Examples

We've created three comprehensive app examples:

### Admin Dashboard (`admin-dashboard.json`)
- User management interface
- Real-time metrics and activity monitoring
- Bulk operations and data export
- Modal forms for CRUD operations

### Customer Onboarding Form (`data-entry-form.json`)
- Multi-step form with progress tracking
- Field validation and conditional logic
- File upload capabilities
- Auto-save functionality

### System Monitoring Dashboard (`monitoring-dashboard.json`)
- Real-time system metrics
- Interactive charts and visualizations
- Service status monitoring
- Alert management

## Current Workflow

Since programmatic app creation isn't available yet, the workflow is:

1. **Prepare the App**
   ```bash
   ./manage.sh --action prepare-app --app-name admin-dashboard
   ```

2. **Create Required Scripts**
   - Review the generated `*-required-scripts.txt` file
   - Create each script in Windmill using the UI or API

3. **Import the App**
   - Access Windmill UI at http://localhost:5681
   - Navigate to Apps → New App
   - Manually recreate the app using the JSON as reference
   - Or check if an import feature has been added

## API Usage

The app management API is now available:

```bash
# Deploy an app to a workspace
./manage.sh --action deploy-app --app-name admin-dashboard --workspace demo

# Check API availability
./manage.sh --action check-app-api
```

The implementation includes:
- Automatic format conversion from our examples to Windmill's structure
- Error handling for workspace constraint issues
- Fallback guidance for manual import when needed

## Technical Details

### App Definition Structure

Apps are defined as JSON with:
- `layout`: Component hierarchy
- `state`: Initial state configuration
- `scripts`: Connected backend scripts
- `modals`: Dialog definitions
- `required_scripts`: Dependencies

### Component Types

- **Display**: Text, tables, charts, metrics
- **Input**: Forms, file uploads, search
- **Layout**: Containers, grids, tabs
- **Actions**: Buttons, links, menus

### Data Binding

Uses template syntax:
```javascript
${scripts.get_users.output}
${state.current_page}
${form.values.email}
```

## Monitoring for Updates

Check these resources for app API availability:
- [Windmill Documentation](https://docs.windmill.dev)
- [GitHub Releases](https://github.com/windmill-labs/windmill/releases)
- [API Reference](https://docs.windmill.dev/docs/core_concepts/api)

## Benefits for AI Agents

This implementation enables AI agents to:
1. Understand available app templates
2. Deploy apps programmatically via API
3. Handle deployment errors gracefully
4. Fall back to manual import instructions when needed
5. Check API availability and constraints

The app creation API is functional, though some versions may have constraints that require manual intervention.