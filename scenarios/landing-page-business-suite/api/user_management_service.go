package main

import "landing-page-business-suite-api/internal/administration"

type (
	UserManagementStore   = administration.UserManagementStore
	UserManagementService = administration.UserManagementService
)

func NewUserManagementService(db UserManagementStore) *UserManagementService {
	return administration.NewUserManagementService(db)
}
