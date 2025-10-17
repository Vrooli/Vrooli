#!/usr/bin/env bash
# farmOS Installation Helper
# Handles automatic installation completion for farmOS

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Check if farmOS needs installation
farmos::install::check_needed() {
    local response=$(timeout 5 curl -sf -I "${FARMOS_BASE_URL}/api" 2>/dev/null)
    if echo "$response" | grep -q "install.php"; then
        return 0  # Installation needed
    fi
    return 1  # Already installed
}

# Complete farmOS installation
farmos::install::complete() {
    echo "Completing farmOS installation..."
    
    # Check if already installed
    if ! farmos::install::check_needed; then
        echo "farmOS is already installed"
        return 0
    fi
    
    # Use drush inside the container to complete installation
    echo "Running farmOS site installation (this may take 2-3 minutes)..."
    
    docker exec farmos drush site:install farm \
        --db-url="pgsql://${FARMOS_DB_USER}:${FARMOS_DB_PASSWORD}@${FARMOS_DB_HOST}:${FARMOS_DB_PORT}/${FARMOS_DB_NAME}" \
        --site-name="Vrooli Farm Management" \
        --account-name="${FARMOS_ADMIN_USER}" \
        --account-pass="${FARMOS_ADMIN_PASSWORD}" \
        --account-mail="${FARMOS_ADMIN_EMAIL}" \
        --yes \
        2>/dev/null
    
    if [[ $? -eq 0 ]]; then
        echo "farmOS installation completed successfully"
        
        # Enable required modules
        echo "Enabling farmOS modules..."
        docker exec farmos drush en -y \
            farm_api \
            farm_field \
            farm_equipment \
            farm_animal \
            farm_activity \
            farm_observation \
            farm_harvest \
            farm_quick \
            restui \
            jsonapi \
            2>/dev/null
        
        # Clear caches
        echo "Clearing caches..."
        docker exec farmos drush cache:rebuild 2>/dev/null
        
        # Load demo data if enabled
        if [[ "${FARMOS_DEMO_DATA}" == "true" ]]; then
            echo "Loading demo data..."
            farmos::install::load_demo_data
        fi
        
        return 0
    else
        echo "Error: farmOS installation failed"
        return 1
    fi
}

# Load demo data using Drush
farmos::install::load_demo_data() {
    echo "Creating demo farm content..."
    
    # Create demo fields using drush
    docker exec farmos drush eval '
        use Drupal\asset\Entity\Asset;
        use Drupal\log\Entity\Log;
        use Drupal\taxonomy\Entity\Term;
        
        // Create demo fields
        $fields = [
            ["name" => "North Field", "size" => 25],
            ["name" => "South Field", "size" => 15],
            ["name" => "East Pasture", "size" => 30],
            ["name" => "Greenhouse A", "size" => 5],
        ];
        
        foreach ($fields as $field_data) {
            $field = Asset::create([
                "type" => "land",
                "land_type" => "field",
                "name" => $field_data["name"],
                "status" => "active",
            ]);
            $field->save();
            echo "Created field: " . $field_data["name"] . "\n";
        }
        
        // Create demo equipment
        $equipment = [
            ["name" => "John Deere Tractor", "model" => "5075E"],
            ["name" => "Combine Harvester", "model" => "S660"],
            ["name" => "Irrigation System", "model" => "Center Pivot"],
        ];
        
        foreach ($equipment as $equip_data) {
            $asset = Asset::create([
                "type" => "equipment",
                "name" => $equip_data["name"],
                "status" => "active",
                "notes" => ["value" => "Model: " . $equip_data["model"]],
            ]);
            $asset->save();
            echo "Created equipment: " . $equip_data["name"] . "\n";
        }
        
        // Create demo livestock
        $livestock = [
            ["name" => "Dairy Herd", "count" => 20, "breed" => "Holstein"],
            ["name" => "Beef Cattle", "count" => 15, "breed" => "Angus"],
            ["name" => "Chicken Flock", "count" => 100, "breed" => "Rhode Island Red"],
        ];
        
        foreach ($livestock as $animal_data) {
            $asset = Asset::create([
                "type" => "animal",
                "name" => $animal_data["name"],
                "status" => "active",
                "animal_type" => "group",
                "notes" => ["value" => $animal_data["count"] . " " . $animal_data["breed"]],
            ]);
            $asset->save();
            echo "Created livestock: " . $animal_data["name"] . "\n";
        }
        
        // Create demo activity logs
        $activities = [
            ["name" => "Field Preparation", "type" => "activity"],
            ["name" => "Planted Corn", "type" => "seeding"],
            ["name" => "Applied Fertilizer", "type" => "input"],
            ["name" => "Irrigation Run", "type" => "activity"],
            ["name" => "Pest Inspection", "type" => "observation"],
        ];
        
        foreach ($activities as $activity_data) {
            $log = Log::create([
                "type" => $activity_data["type"],
                "name" => $activity_data["name"],
                "status" => "done",
                "timestamp" => time(),
            ]);
            $log->save();
            echo "Created log: " . $activity_data["name"] . "\n";
        }
        
        echo "Demo data loading completed\n";
    ' 2>/dev/null
    
    return 0
}

# Auto-complete installation on first start
farmos::install::auto_setup() {
    if farmos::install::check_needed; then
        echo "farmOS requires initial setup..."
        farmos::install::complete
    fi
}