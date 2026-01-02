/**
 * Scheduling documentation content
 *
 * Covers:
 * - What scheduled workflows are
 * - Creating and managing schedules
 * - Cron expressions
 * - Monitoring scheduled runs
 */

export const SCHEDULING_CONTENT = `
# Scheduling

Automate your workflows to run on a schedule. Perfect for regular testing, data collection, or any repetitive browser automation task.

---

## What Is Scheduling?

Scheduling lets you run workflows automatically at specified times without manual intervention. Use it for:

- **Automated testing** - Run e2e tests every night
- **Data collection** - Scrape prices or inventory hourly
- **Monitoring** - Check website uptime every 15 minutes
- **Reporting** - Generate reports on a daily or weekly basis

---

## Creating a Schedule

### Step 1: Open Schedules

Navigate to **Settings > Schedules** and click **"Create Schedule"**.

### Step 2: Configure Basic Settings

| Field | Description |
| --- | --- |
| **Name** | Descriptive name (e.g., "Nightly E2E Tests") |
| **Description** | Optional details about what this schedule does |
| **Workflow** | Select which workflow to run |
| **Start URL** | Override the workflow's starting URL (optional) |

### Step 3: Set the Timing

Choose when the schedule runs using presets or custom cron:

#### Quick Presets

| Preset | Cron | Description |
| --- | --- | --- |
| **Every 15 minutes** | \`*/15 * * * *\` | Four times per hour |
| **Hourly** | \`0 * * * *\` | At the top of every hour |
| **Every 6 hours** | \`0 */6 * * *\` | Four times per day |
| **Twice daily** | \`0 9,17 * * *\` | 9 AM and 5 PM |
| **Daily (9 AM)** | \`0 9 * * *\` | Once per day at 9 AM |
| **Weekly (Monday 9 AM)** | \`0 9 * * 1\` | Once per week |

#### Custom Cron

For precise control, enter a cron expression:

\`\`\`
┌──────────── minute (0-59)
│ ┌────────── hour (0-23)
│ │ ┌──────── day of month (1-31)
│ │ │ ┌────── month (1-12)
│ │ │ │ ┌──── day of week (0-6, Sunday=0)
│ │ │ │ │
* * * * *
\`\`\`

**Examples:**

| Cron | Meaning |
| --- | --- |
| \`0 9 * * 1-5\` | 9 AM on weekdays |
| \`30 8,12,18 * * *\` | 8:30 AM, 12:30 PM, 6:30 PM daily |
| \`0 0 1 * *\` | Midnight on the 1st of each month |
| \`*/5 * * * *\` | Every 5 minutes |
| \`0 3 * * 0\` | 3 AM every Sunday |

### Step 4: Set the Timezone

Select the timezone for schedule timing. Common options:
- UTC (Coordinated Universal Time)
- America/New_York (Eastern Time)
- America/Los_Angeles (Pacific Time)
- Europe/London (GMT/BST)
- Asia/Tokyo (Japan Standard Time)

### Step 5: Activate

Toggle **Active** to enable the schedule immediately, or leave it off to configure now and activate later.

---

## Managing Schedules

### Schedule Status

Each schedule shows its current state:

| Badge | Meaning |
| --- | --- |
| **Active** | Schedule is running on its configured timing |
| **Paused** | Schedule exists but won't run until resumed |

### Schedule Information

- **Cron Description** - Human-readable timing (e.g., "Every weekday at 9:00 AM")
- **Next Run** - When the schedule will run next
- **Last Run** - When it last ran and the result (success/failure)

### Actions

| Action | Description |
| --- | --- |
| **Trigger Now** | Run the workflow immediately (active schedules only) |
| **Pause** | Stop scheduled runs without deleting |
| **Resume** | Restart a paused schedule |
| **Edit** | Modify schedule settings |
| **Delete** | Permanently remove the schedule |

---

## Monitoring Scheduled Runs

### Execution History

Each scheduled run creates an execution record:
1. Go to the workflow's executions list
2. Scheduled runs are marked with a schedule icon
3. View results, screenshots, and logs

### Last Run Status

The schedule card shows the last run's outcome:
- **Success** - Workflow completed without errors
- **Failure** - One or more steps failed
- **Running** - Currently in progress (for long workflows)

### Troubleshooting Failed Runs

1. Click on the schedule to see last run details
2. Review the execution log for error messages
3. Check if selectors have changed on the target site
4. Verify the session still has valid authentication

---

## Best Practices

### Naming Conventions

Use descriptive names that indicate:
- **What** the schedule does
- **When** it runs
- **Which environment** it targets

Good: "Production Login Test - Hourly"
Bad: "Schedule 1"

### Session Management

- Use a dedicated session for scheduled workflows
- Ensure the session has saved authentication if needed
- Periodically verify session validity (cookies expire)

### Error Handling

- Configure retry settings in workflow defaults
- Enable "Continue on Error" for non-critical steps
- Set up notifications for failed runs (if supported)

### Timing Considerations

- Avoid peak traffic times if testing production
- Stagger multiple schedules to reduce load
- Consider timezone differences for global teams

### Resource Usage

- More frequent schedules use more server resources
- Limit concurrent scheduled runs if possible
- Use longer intervals for non-critical monitoring

---

## Common Use Cases

### Automated E2E Testing

\`\`\`
Schedule: "Nightly E2E Suite"
Cron: 0 2 * * * (2 AM daily)
Workflow: Full login and checkout flow
Session: Test Account
\`\`\`

### Price Monitoring

\`\`\`
Schedule: "Competitor Price Check"
Cron: 0 */4 * * * (every 4 hours)
Workflow: Scrape competitor pricing page
Session: None (public page)
\`\`\`

### Uptime Monitoring

\`\`\`
Schedule: "Website Health Check"
Cron: */15 * * * * (every 15 minutes)
Workflow: Load homepage, verify key elements
Session: None (public page)
\`\`\`

### Weekly Reports

\`\`\`
Schedule: "Weekly Analytics Screenshot"
Cron: 0 9 * * 1 (Monday 9 AM)
Workflow: Login to analytics, screenshot dashboard
Session: Analytics Account
\`\`\`

---

## Limitations

- Schedules require the application to be running
- Maximum frequency varies by subscription tier
- Very long workflows may timeout on tight schedules
- Concurrent schedule limit depends on system resources
`;
