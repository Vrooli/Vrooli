# Kids Dashboard 🎮🌈

A safe, fun, and colorful dashboard for children to access kid-friendly Vrooli scenarios.

## 🎯 Purpose

The Kids Dashboard provides a curated, age-appropriate interface for children aged 5-12 to interact with Vrooli scenarios. It filters out system/development tools and only shows fun, educational, and safe content.

## ✨ Features

- **Colorful, animated UI** with fun character mascot (Vrooli the Robot)
- **Category-based filtering** - only shows scenarios tagged as "kid-friendly"
- **Iframe isolation** for secure scenario execution
- **Age-appropriate content** with no access to system tools
- **Touch-friendly interface** for tablets and touchscreens
- **Fun animations and sound effects** (optional)

## 🚀 Quick Start

```bash
# Setup the scenario
./scripts/manage.sh setup

# Run the dashboard
vrooli scenario run kids-dashboard
```

The dashboard will be available at `http://localhost:3500`

## 🎨 Included Scenarios

Currently includes these kid-friendly scenarios:
- **Retro Games** 🕹️ - Classic arcade games
- **Picker Wheel** 🎯 - Fun decision spinner
- **Study Buddy** 📚 - Homework helper (with safe mode)

## 🔐 Safety Features

- Only displays explicitly tagged kid-friendly scenarios
- Iframe sandboxing prevents system access
- No data collection from children (COPPA compliant)
- Parental controls ready (future integration)

## 🏗️ Architecture

- **Frontend**: React with Tailwind CSS and Framer Motion
- **Backend**: Go API for scenario discovery and filtering
- **Security**: Iframe isolation with restricted permissions
- **Discovery**: Automatic scanning of scenario catalog

## 👨‍👩‍👧‍👦 For Parents

This dashboard ensures your children only access appropriate content within Vrooli. All scenarios are vetted and tagged before appearing in the dashboard.

## 🎮 Adding New Kid-Friendly Scenarios

To make a scenario appear in the kids dashboard, add `"kid-friendly"` to its tags in the scenario's `.vrooli/service.json`:

```json
{
  "service": {
    "tags": [
      "existing-tags",
      "kid-friendly"
    ]
  }
}
```

## 🌟 Future Enhancements

- Parental controls dashboard
- Time limits and usage tracking
- Educational progress tracking
- Multiple child profiles
- More themes and mascot characters

## 📝 License

Part of the Vrooli platform - creating safe, fun experiences for the whole family!