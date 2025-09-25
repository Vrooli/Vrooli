#!/bin/bash
# Apply Date Night Planner schema using available PostgreSQL tools

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
SCHEMA_FILE="$SCENARIO_DIR/initialization/storage/postgres/schema.sql"
SEED_FILE="$SCENARIO_DIR/initialization/storage/postgres/seed.sql"

# Try to find a way to execute SQL
echo "🔍 Looking for PostgreSQL execution method..."

# Method 1: Try using Python psycopg2
if command -v python3 >/dev/null 2>&1; then
    echo "Attempting to use Python psycopg2..."
    python3 << EOF
import sys
try:
    import psycopg2
    conn = psycopg2.connect(
        host='localhost',
        port='5433',
        database='vrooli',
        user='vrooli',
        password='lUq9qvemypKpuEeXCV6Vnxak1'
    )
    cur = conn.cursor()
    
    # Read and execute schema file
    with open('$SCHEMA_FILE', 'r') as f:
        schema_sql = f.read()
    
    # Execute the schema
    cur.execute(schema_sql)
    conn.commit()
    
    # Verify schema was created
    cur.execute("SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'date_night_planner')")
    exists = cur.fetchone()[0]
    
    if exists:
        print("✅ Schema 'date_night_planner' created successfully")
        
        # List tables
        cur.execute("""
            SELECT table_name 
            FROM information_schema.tables 
            WHERE table_schema = 'date_night_planner' 
            ORDER BY table_name
        """)
        tables = cur.fetchall()
        print("\n📊 Created tables:")
        for table in tables:
            print(f"  - {table[0]}")
            
        # Apply seed data if exists
        import os
        if os.path.exists('$SEED_FILE'):
            print("\n🌱 Applying seed data...")
            with open('$SEED_FILE', 'r') as f:
                seed_sql = f.read()
            cur.execute(seed_sql)
            conn.commit()
            print("✅ Seed data applied")
    else:
        print("❌ Schema creation failed")
        sys.exit(1)
        
    cur.close()
    conn.close()
    print("\n🎉 Database initialization complete!")
    
except ImportError:
    print("❌ psycopg2 not installed. Installing...")
    import subprocess
    subprocess.run([sys.executable, "-m", "pip", "install", "psycopg2-binary"], check=True)
    print("Please run this script again.")
    sys.exit(1)
except Exception as e:
    print(f"❌ Error: {e}")
    sys.exit(1)
EOF
    exit $?
fi

echo "❌ No suitable PostgreSQL execution method found"
echo "Please install psql or python3 with psycopg2"
exit 1