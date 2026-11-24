package main

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGraphPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions GraphPermissions
		userID      string
		createdBy   string
		level       PermissionLevel
		want        bool
	}{
		{
			name: "creator has read access",
			permissions: GraphPermissions{
				Public: false,
			},
			userID:    "user1",
			createdBy: "user1",
			level:     PermissionRead,
			want:      true,
		},
		{
			name: "creator has write access",
			permissions: GraphPermissions{
				Public: false,
			},
			userID:    "user1",
			createdBy: "user1",
			level:     PermissionWrite,
			want:      true,
		},
		{
			name: "public graph readable by anyone",
			permissions: GraphPermissions{
				Public: true,
			},
			userID:    "user2",
			createdBy: "user1",
			level:     PermissionRead,
			want:      true,
		},
		{
			name: "public graph not writable by non-owner",
			permissions: GraphPermissions{
				Public: true,
			},
			userID:    "user2",
			createdBy: "user1",
			level:     PermissionWrite,
			want:      false,
		},
		{
			name: "allowed user has read access",
			permissions: GraphPermissions{
				Public:       false,
				AllowedUsers: []string{"user2", "user3"},
			},
			userID:    "user2",
			createdBy: "user1",
			level:     PermissionRead,
			want:      true,
		},
		{
			name: "allowed user no write access",
			permissions: GraphPermissions{
				Public:       false,
				AllowedUsers: []string{"user2"},
			},
			userID:    "user2",
			createdBy: "user1",
			level:     PermissionWrite,
			want:      false,
		},
		{
			name: "editor has read access",
			permissions: GraphPermissions{
				Public:  false,
				Editors: []string{"user2"},
			},
			userID:    "user2",
			createdBy: "user1",
			level:     PermissionRead,
			want:      true,
		},
		{
			name: "editor has write access",
			permissions: GraphPermissions{
				Public:  false,
				Editors: []string{"user2"},
			},
			userID:    "user2",
			createdBy: "user1",
			level:     PermissionWrite,
			want:      true,
		},
		{
			name: "non-allowed user no read access to private graph",
			permissions: GraphPermissions{
				Public:       false,
				AllowedUsers: []string{"user3"},
			},
			userID:    "user2",
			createdBy: "user1",
			level:     PermissionRead,
			want:      false,
		},
		{
			name: "non-editor no write access",
			permissions: GraphPermissions{
				Public:  false,
				Editors: []string{"user3"},
			},
			userID:    "user2",
			createdBy: "user1",
			level:     PermissionWrite,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create SQL mock: %v", err)
			}
			defer db.Close()

			permissionsJSON, err := json.Marshal(tt.permissions)
			if err != nil {
				t.Fatalf("marshal permissions: %v", err)
			}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT created_by, permissions\n\t\tFROM graphs\n\t\tWHERE id = $1")).
				WithArgs("graph1").
				WillReturnRows(sqlmock.NewRows([]string{"created_by", "permissions"}).AddRow(tt.createdBy, string(permissionsJSON)))

			got, err := CheckGraphPermission(db, "graph1", tt.userID, tt.level)
			if err != nil {
				t.Fatalf("CheckGraphPermission() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Permission check failed: got %v, want %v", got, tt.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("SQL expectations: %v", err)
			}
		})
	}
}

func TestGraphPermissionsJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		wantPublic  bool
		wantEditors int
		wantUsers   int
		wantError   bool
	}{
		{
			name:        "default public permissions",
			jsonData:    `{"public": true}`,
			wantPublic:  true,
			wantEditors: 0,
			wantUsers:   0,
			wantError:   false,
		},
		{
			name:        "private with editors",
			jsonData:    `{"public": false, "editors": ["user1", "user2"]}`,
			wantPublic:  false,
			wantEditors: 2,
			wantUsers:   0,
			wantError:   false,
		},
		{
			name:        "private with allowed users",
			jsonData:    `{"public": false, "allowed_users": ["user1", "user2", "user3"]}`,
			wantPublic:  false,
			wantEditors: 0,
			wantUsers:   3,
			wantError:   false,
		},
		{
			name:        "full permissions",
			jsonData:    `{"public": false, "allowed_users": ["viewer1"], "editors": ["editor1", "editor2"]}`,
			wantPublic:  false,
			wantEditors: 2,
			wantUsers:   1,
			wantError:   false,
		},
		{
			name:       "invalid JSON",
			jsonData:   `{invalid json}`,
			wantPublic: false,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var permissions GraphPermissions
			err := json.Unmarshal([]byte(tt.jsonData), &permissions)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error parsing JSON, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error parsing JSON: %v", err)
				return
			}

			if permissions.Public != tt.wantPublic {
				t.Errorf("Public = %v, want %v", permissions.Public, tt.wantPublic)
			}

			if len(permissions.Editors) != tt.wantEditors {
				t.Errorf("Editors count = %v, want %v", len(permissions.Editors), tt.wantEditors)
			}

			if len(permissions.AllowedUsers) != tt.wantUsers {
				t.Errorf("AllowedUsers count = %v, want %v", len(permissions.AllowedUsers), tt.wantUsers)
			}
		})
	}
}

func TestPermissionLevels(t *testing.T) {
	tests := []struct {
		name  string
		level PermissionLevel
		want  int
	}{
		{
			name:  "read permission level",
			level: PermissionRead,
			want:  0,
		},
		{
			name:  "write permission level",
			level: PermissionWrite,
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.level) != tt.want {
				t.Errorf("PermissionLevel = %v, want %v", tt.level, tt.want)
			}
		})
	}
}
